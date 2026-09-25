# Go Data Profiler Platform

Production-oriented, database-agnostic data profiling and data-quality platform written in Go.

## Included

- REST API with versioning
- PostgreSQL, MySQL, SQL Server adapters
- Snowflake-ready adapter boundary (driver integration isolated behind the adapter interface)
- Automatic schema/table/column discovery
- Generic column profiling: nulls, distinctness, min/max, average, lengths, empty strings, top values
- Concurrent profiling with bounded worker pool
- Async profiling jobs and status tracking
- Data-quality rules and score
- PII/sensitive-data heuristic detection
- Profile snapshots and reusable drift-comparison package
- Prometheus metrics and structured logging
- PostgreSQL persistence for jobs/results/snapshots
- In-process bounded job queue; Redis/Kafka integration architecture is documented for production scaling
- Kafka integration boundary documented for event-driven deployments
- Docker Compose for local development
- Kubernetes manifests with probes, HPA, ConfigMap and Secret
- AWS deployment architecture notes for RDS/EKS/ElastiCache/MSK/S3/CloudWatch
- Unit tests for profiler, quality, PII, drift and API basics

## Quick start

```bash
docker compose up --build
```

API: `http://localhost:8080`
Metrics: `http://localhost:8080/metrics`

Create a profile job:

```bash
curl -X POST http://localhost:8080/api/v1/profile-jobs \
  -H 'Content-Type: application/json' \
  -d '{"source":{"type":"postgres","dsn":"postgres://postgres:postgres@host.docker.internal:5432/sample?sslmode=disable"},"schema":"public","table":"customers"}'
```

Then:

```bash
curl http://localhost:8080/api/v1/profile-jobs/{job-id}
```

### Phase 2 dirty-data validation

A PostgreSQL fixture is included at `examples/dirty_profile_test.sql`. It intentionally contains NULLs, empty strings, whitespace-only strings, duplicates, zero/negative values, decimals, and timestamps.

```bash
psql "$DATABASE_URL" -f examples/dirty_profile_test.sql
```

Profile `public.profile_test_dirty` with the API and verify:

- text columns: `null_count`, `empty_count`, `whitespace_count`, min/max/average length
- numeric columns: `min`, `max`, `avg`, `zero_count`, standard deviation and percentiles
- date/time columns: `min` and `max`
- all columns: `distinct_count` and null percentages


## Architecture

```text
Client/UI
   |
REST API (/api/v1)
   |
Job Service ---- Metrics/Logs/Tracing boundary
   |
Queue (Redis / in-process)
   |
Worker Pool
   |
Profiler Engine
   |
DatabaseAdapter
   +-- PostgreSQL
   +-- MySQL
   +-- SQL Server
   +-- Snowflake adapter boundary
   |
Source DBs

Profiler results -> Store -> snapshots -> drift -> quality/PII
```

## Production notes

1. Never expose source DB credentials directly to untrusted clients. In production, pass a secret reference and resolve it from AWS Secrets Manager/Kubernetes Secrets.
2. Run profiling with a least-privileged read-only source account.
3. Large tables should use sampling/approximate distinct counts and configurable scan budgets. The implementation exposes a sampling strategy boundary for the next optimization stage.
4. PII detection is heuristic and must not be treated as a compliance decision without validation.
5. Use TLS for every external connection in production.

## Commands

```bash
go test ./...
go vet ./...
go run ./cmd/server
```
