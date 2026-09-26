# Go Data Profiler Platform

Production-oriented, database-native data profiling and data-quality platform written in Go.

## What this project does

The profiler connects to a relational source database, discovers its schema, calculates statistical and quality metrics, stores a profile snapshot, and compares snapshots over time.

The core lifecycle is:

**Connect → Discover → Measure → Validate → Explain → Store → Compare → Act**

It is designed as an API-first service so profiling can run asynchronously and be integrated into data engineering, migration, QA, governance, and observability workflows.

## Current capabilities

- REST API with versioning
- PostgreSQL, MySQL, and SQL Server source adapters
- Automatic schema/table/column discovery
- Row count, null count/percentage, distinct count/percentage
- Empty and whitespace-only value detection
- Numeric statistics:
  - min/max
  - average/sum
  - variance/stddev
  - skewness/kurtosis
  - median and p50/p75/p90/p95/p99
  - zero count
  - outliers
  - histogram
- Text profiling:
  - min/max/average length
  - top values
  - value patterns
  - entropy/normalized entropy
- Date/time profiling:
  - min/max
  - future-date detection
  - date range
  - monthly distribution
- PII/sensitive-data heuristic detection with confidence
- Duplicate-row detection
- Numeric correlations
- Foreign-key relationship discovery
- Configurable data-quality rules and quality score
- Persistent asynchronous profiling jobs
- Idempotency-key support
- Bounded worker pool and bounded source DB connection pool
- Startup queued-job recovery and stale-running-job recovery
- Profile history and drift detection
- Prometheus metrics and structured logging
- API-level source DSN redaction
- PostgreSQL persistence for jobs and profile snapshots
- Docker Compose local deployment
- Kubernetes/AWS deployment architecture foundations

## Quick start

### Prerequisites

- Go
- Docker and Docker Compose
- A source PostgreSQL, MySQL, or SQL Server database accessible from the profiler

### Start locally

```bash
git clone https://github.com/adm-l/data_profiler.git
cd data_profiler

go test ./...
go vet ./...

docker compose up --build -d
docker compose ps
docker compose logs --tail=100 profiler-api
```

Check the API:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/metrics
```

## Create a profiling job

Example using PostgreSQL:

```bash
curl --location 'http://localhost:8080/api/v1/profile-jobs' \
  --header 'Content-Type: application/json' \
  --data-raw '{
    "source": {
      "type": "postgres",
      "dsn": "postgres://postgres@host.docker.internal:5432/user_db?sslmode=disable"
    },
    "schema": "public",
    "table": "users2",
    "sample_size": 1000
  }'
```

The API returns a job ID.

Poll the job:

```bash
curl http://localhost:8080/api/v1/profile-jobs/<JOB_ID>
```

Retrieve the completed profile:

```bash
curl http://localhost:8080/api/v1/profile-jobs/<JOB_ID>/result
```

## Idempotency

Use `Idempotency-Key` when a client may retry the same profiling request:

```bash
curl --location 'http://localhost:8080/api/v1/profile-jobs' \
  --header 'Content-Type: application/json' \
  --header 'Idempotency-Key: users2-profile-v1' \
  --data-raw '{
    "source": {
      "type": "postgres",
      "dsn": "postgres://postgres@host.docker.internal:5432/user_db?sslmode=disable"
    },
    "schema": "public",
    "table": "users2",
    "sample_size": 1000
  }'
```

## History and drift

Profile history:

```bash
curl 'http://localhost:8080/api/v1/profiles/history?schema=public&table=users2&limit=20'
```

Drift:

```bash
curl 'http://localhost:8080/api/v1/profiles/drift?schema=public&table=users2'
```

Drift can report:

- added/removed columns
- type or nullability changes
- distinct-count changes
- significant null/distinct percentage changes
- outlier changes
- PII classification changes
- duplicate changes
- quality-score changes

## Dirty-data validation fixture

A PostgreSQL fixture is available at:

```
examples/dirty_profile_test.sql
```

It is intended to exercise NULLs, empty strings, whitespace-only values, duplicates, numeric values, zero/negative values, decimals, and timestamps.

Load it into a standalone PostgreSQL database:

```bash
psql "$DATABASE_URL" -f examples/dirty_profile_test.sql
```

Then profile the fixture and inspect:

- completeness and null percentages
- distinctness/cardinality
- empty and whitespace values
- numeric distributions and percentiles
- date ranges
- duplicate rows
- quality checks

## Architecture

```text
                         Client / UI
                             |
                             v
                    Profiler HTTP API
                             |
                  +----------+----------+
                  |                     |
                  v                     v
          Metadata PostgreSQL       Source DB
          jobs + profiles        PostgreSQL/MySQL/
                  ^               SQL Server
                  |
          Persistent Job State
                  |
          Bounded Worker Pool
                  |
           Profiling Engine
                  |
       +----------+----------+
       |          |          |
     Basic     Advanced    Quality/
     Metrics   Metrics      PII
       |          |          |
       +----------+----------+
                  |
            Profile Snapshot
                  |
          History + Drift
```

The metadata database stores profiler jobs and profile snapshots. It is separate from the source database being analyzed.

## How to read a profile

### Completeness

`null_percentage` shows how much data is missing. Compare it with the configured quality threshold.

### Cardinality

`distinct_count` and `distinct_percentage` help identify identifiers, dimensions, and unexpected duplicate-heavy columns.

### Numeric distribution

Use percentiles, standard deviation, skewness, kurtosis, histograms, and outlier statistics to understand the shape of numeric data rather than relying only on min/max.

### Text quality

Length, empty, whitespace, top-value, pattern, and entropy metrics expose formatting problems and low-information values.

### Date quality

Minimum/maximum values, future-date counts, ranges, and monthly distribution can expose invalid or unexpected temporal data.

### PII

PII classification is heuristic evidence for investigation. It must not be treated as a compliance decision without validation.

### Quality score

The score is a summary. Always inspect the individual checks to understand why the score changed.

### Drift

Use history and drift to distinguish a newly introduced issue from an existing data characteristic.

## Example result interpretation

For a five-row sample:

- An ID column with 5 distinct values and no NULLs is consistent with unique values in that sample.
- A name column with one NULL has 20% missingness.
- A whitespace-only name is different from SQL NULL and is reported separately.
- An email column containing a malformed address can show a pattern-quality issue even when most values look valid.
- A quality score should be interpreted together with its checks.
- Drift should be checked to determine whether the current condition differs from the previous snapshot.

## Database support

### PostgreSQL

Recommended source permissions are read-only access to the target schema/table plus the metadata required by the adapter.

### MySQL

Use a read-only source account with SELECT and required metadata visibility.

### SQL Server

Use a read-only source account with access to the target objects and metadata.

The profiler should not require write access to the source dataset.

## Performance design

The service deliberately bounds expensive work:

- bounded profiling worker pool
- bounded source database connection pool
- configurable source connection lifetime
- optional sampling for large tables
- capped numeric correlation analysis

Example configuration:

```text
JOB_WORKERS=4
PROFILE_SOURCE_MAX_OPEN_CONNS=4
PROFILE_SOURCE_MAX_IDLE_CONNS=2
PROFILE_SOURCE_CONN_MAX_LIFETIME=30m
```

Performance depends on source DB size, schema, indexes, query plans, network latency, enabled metrics, sampling, and concurrency. No universal performance claim should be made without controlled benchmarks.

## Performance benchmark plan

For a meaningful benchmark, vary:

| Dimension | Example values |
|---|---|
| Rows | 10K, 100K, 1M, 10M, 100M+ |
| Columns | 5, 20, 50, 100 |
| Data types | numeric, text, date, mixed |
| Metrics | basic, standard, deep |
| Sampling | off, 1K, 10K, 100K |
| Workers | 1, 2, 4, 8 |
| Database | PostgreSQL, MySQL, SQL Server |
| Network | local, same-region, higher latency |

Measure wall-clock time, DB CPU/memory, profiler CPU/memory, query count, rows scanned, and source DB load.

## Comparison with other profiling/data-quality tools

This project has a different architectural focus from several established tools:

| Tool | Primary orientation |
|---|---|
| Go Data Profiler | Database-native Go service, async jobs, quality, history/drift |
| YData Profiling | Python/Pandas/Spark exploratory profiling and reports |
| Great Expectations | Declarative data validation and expectations |
| Soda | Data quality checks and observability |
| DataPrep | Python dataframe-oriented EDA/profiling |

This is a capability comparison, not a performance ranking. Any performance comparison should use the benchmark methodology above.

## Production considerations

1. Use least-privileged, read-only source credentials.
2. Do not expose source credentials to untrusted clients.
3. Prefer secret references/secret-manager integration over plaintext credentials.
4. Use TLS for external database connections.
5. Add authentication and authorization before exposing the API broadly.
6. Add distributed job claiming/leases before running multiple worker instances.
7. Add a real migration runner with schema version tracking.
8. Add OpenTelemetry tracing around jobs and source queries.
9. Use profiling tiers and scan budgets for very large datasets.
10. Consider approximate distinct counts/quantiles for very large tables.
11. Make duplicate detection configurable because full-table grouping can be expensive.
12. Add richer drift output with before/after values and severity.

## Known limitations

- `readyz` is currently a simple readiness endpoint rather than a complete dependency health check.
- Multi-instance deployments need distributed job claiming/leases.
- Existing Docker volumes may require manual application of new SQL migrations.
- Source DSNs are redacted from API responses but still require stronger secret-management treatment in persistent metadata.
- Duplicate-row detection can be expensive on wide/large tables.
- Generic PII labels such as `name` can produce false positives and need stronger value evidence.
- A formal cross-tool performance benchmark has not yet been established.

## Development commands

```bash
go test ./...
go vet ./...
go run ./cmd/server
```

## Project mental model

**Connect → Discover → Measure → Validate → Explain → Store → Compare → Act**

The goal is not merely to calculate statistics. The goal is to turn a database into an explainable profile that engineers can use to investigate data quality, structure, privacy signals, and change over time.

## Documentation

- `docs/Go_Data_Profiler_Guide.pdf` — installation, operation, API examples, architecture, and profiling analysis guide.
- This README — project overview and practical quick reference.
- The master technical/research copy is intended to document implementation decisions, benchmarking, limitations, and future evolution as the project grows.
