# Go Data Profiler — Master Technical Copy

A living technical reference for architecture, implementation, operation, analysis, benchmarking, and future evolution.

Repository: adm-l/data_profiler
Language: Go
Focus: API-first relational database profiling

## 1. Executive Summary
The Go Data Profiler is an API-first service that connects to relational databases, profiles tables, measures data quality, detects selected privacy-sensitive patterns, discovers relationships, stores historical snapshots, and reports drift. It keeps large data operations close to the database and uses Go for orchestration, concurrency, API handling, persistence, and result assembly. Current capabilities include PostgreSQL, MySQL, and SQL Server adapters; asynchronous persistent jobs; idempotency; startup queue recovery; stale-job recovery; numeric, text, and date/time profiling; quality checks; PII evidence; duplicate detection; numeric correlations; foreign-key discovery; profile history; and drift reporting.

## 2. Why This Tool Was Built
The practical question is: What is inside this table, how healthy is it, and what changed since the previous observation?
Instead of manually writing many SQL queries for nulls, cardinality, distributions, numeric statistics, duplicates, relationships, and quality checks, a client creates a profiling job and receives one structured result.
The project is also a production-oriented Go engineering exercise covering SQL, database adapters, concurrency, asynchronous jobs, reliability, security, observability, API design, and scaling.

Problems addressed:
- Manual profiling is repetitive and inconsistent.
- Large datasets make application-side row processing expensive.
- Long-running profiling should not block HTTP requests.
- Retries should not accidentally create duplicate work.
- Historical snapshots are needed to identify real change.
- Database-specific SQL should stay inside adapters.

## 3. Who Can Use It
- Data engineers assessing source systems before pipelines or migrations.
- Backend engineers investigating production data quality.
- Analytics and BI teams validating datasets.
- Platform teams building catalog or observability workflows.
- Migration teams comparing source and target databases.
- Developers testing intentionally messy database fixtures.
- Engineers demonstrating Go, SQL, concurrency, reliability, and system-design skills.

## 4. Current Capability Map
Execution: persistent asynchronous jobs, bounded workers, bounded source connections, timeouts, queue recovery, stale-running recovery, and idempotency.
Profiling: row counts, nulls, distinctness, empty/whitespace values, numeric statistics, quantiles, histograms, text lengths, patterns, entropy, and dates.
Quality: configurable thresholds, checks, warnings, and quality score.
Privacy: header and value evidence for selected PII-like classes.
Relationships: duplicate rows, numeric correlations, and foreign-key discovery.
History: profile snapshots and drift reporting.
Operations: health, readiness, metrics, Docker Compose, and redacted API job responses.

## 5. Architecture From Scratch
Client/UI -> Profiler API -> Metadata DB and Source DB.
The metadata database stores jobs and profile snapshots. The source database remains the system of record for actual data.
Worker path: Client -> HTTP API -> persistent job state -> bounded worker pool -> database adapter -> profiling queries -> TableProfile -> PII/quality/drift -> snapshot.

## 6. End-to-End Job Lifecycle
1. Client submits POST /api/v1/profile-jobs.
2. API validates and persists the job.
3. Job ID enters a bounded queue.
4. Worker marks it running and creates a profiling timeout.
5. Adapter discovers metadata and executes profiling queries.
6. Service assembles statistics, quality checks, PII evidence, duplicates, correlations, and relationships.
7. Result is compared with the previous profile.
8. Snapshot is saved.
9. Job becomes completed or failed.
10. Client polls the job and retrieves the result separately.

## 7. Building the Tool Step by Step
Start with domain models and a Store interface.
Create PostgreSQL, MySQL, and SQL Server adapters so database-specific SQL is isolated.
Build a reliable synchronous profiler first: row count, nulls, distinctness, min/max, and basic type statistics.
Move execution into persistent asynchronous jobs.
Add idempotency, bounded queues, bounded DB pools, startup recovery, stale-job recovery, and timeouts.
Layer quality checks and PII evidence over the core profile.
Persist snapshots and compare them for drift.
Optimize expensive metrics only after correctness using sampling, query pushdown, profiling tiers, and controlled concurrency.

## 8. Fresh-Machine Installation and Run
Prerequisites: Go, Docker/Compose, and a reachable PostgreSQL, MySQL, or SQL Server source.
Typical startup:
    git clone <repository>
    cd data_profiler
    go test ./...
    go vet ./...
    docker compose up --build -d
    docker compose ps
    curl http://localhost:8080/healthz
    curl http://localhost:8080/readyz
Docker initialization scripts run only when a PostgreSQL data directory is initialized. Existing volumes do not automatically replay newly added init SQL files. Production should use a real migration runner.

## 9. Database Requirements
The source database must be reachable and the supplied account must have sufficient read and metadata permissions. The metadata database is PostgreSQL in the current Compose design. Supported source adapters target PostgreSQL, MySQL, and SQL Server. For production, use read-only source accounts where practical. Profiling speed depends on indexes, table width, table size, statistics, database load, query plans, and network latency.

## 10. Basic API Example
A representative request:
    curl -X POST http://localhost:8080/api/v1/profile-jobs -H 'Content-Type: application/json' -H 'Idempotency-Key: users2-profile-001'
Then poll:
    curl http://localhost:8080/api/v1/profile-jobs/<job-id>
Retrieve:
    curl http://localhost:8080/api/v1/profile-jobs/<job-id>/result

## 11. Idempotency
The jobs table stores an idempotency key and enforces uniqueness for non-null keys.
    ALTER TABLE jobs ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
    CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_idempotency_key ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
A stable key should be reused when the same logical request is retried.

## 12. Completeness and Null Analysis
Each column reports total rows, null count, and null percentage. A threshold should be interpreted with the individual check. A field at exactly a configured boundary can pass while still representing a meaningful business issue. Always inspect business-critical fields separately from the overall score.

## 13. Cardinality and Uniqueness
Distinct count and distinct percentage describe variation. High distinctness can indicate identifiers; low distinctness can indicate categories or suspicious constants. Cardinality is contextual.

## 14. Numeric Profiling
Numeric columns can include min, max, average, sum, variance, skewness, kurtosis, standard deviation, median, p50, p75, p90, p95, p99, zero count, outlier statistics, and histograms. Quantiles reveal tails that averages can hide. Non-text columns no longer expose meaningless string-length statistics.

## 15. Text Profiling
Text profiling includes min/max/average length, empty and whitespace-only counts, top values, patterns, and entropy. NULL, empty string, and whitespace-only values are distinct states. Normalized entropy is a diversity signal, not a semantic quality judgment.

## 16. Date/Time Profiling
Date/time profiling includes minimum and maximum values, future-date counts and percentages, date range, and monthly distribution where supported. Future dates can expose bad input, timezone errors, or legitimate future business events.

## 17. PII Detection
The detector combines header evidence with value-pattern evidence for email, phone, person-name-like labels, date-of-birth labels, address labels, government-ID-like patterns, credit-card-like values, IP addresses, and UUIDs. Value evidence is stronger than a generic header. Email syntax is validated strictly, IPv4 octets are range checked, and credit-card-like values can use Luhn validation. Confidence is an evidence score, not a legal classification. Generic labels such as name can produce false positives.

## 18. Duplicate Row Detection
Duplicate detection groups rows across relevant columns to find repeated records. It can be expensive on wide or very large tables. Future designs should support configurable modes, sampling, fingerprints, or partition-aware analysis.

## 19. Correlations
Numeric correlations help discover related measures and unexpected coupling. The implementation caps the number of numeric columns considered. Correlation is a diagnostic signal, not proof of causation.

## 20. Relationship Discovery
Foreign-key discovery uses database metadata to identify table relationships where supported. Relationships add structural context that column-level statistics alone cannot provide.

## 21. Quality Rules and Score
Rules can include null thresholds, dominant-value thresholds, high-cardinality thresholds, and controls for empty or whitespace values. The quality score is a summary indicator and must be read with individual checks.

## 22. How to Analyze a Real Result
For a five-row users2 example, an ID can be fully unique while a name field has a NULL and whitespace-only value and an email field has a NULL or malformed value.
Analyze in this order:
1. Total rows and sampling.
2. Null percentages.
3. Distinctness.
4. Empty and whitespace values.
5. Numeric distribution and outliers.
6. Text patterns.
7. PII evidence.
8. Quality checks.
9. History and drift.
The outcome should be a prioritized description of what needs investigation, cleaning, constraints, or monitoring.

## 23. History and Drift
History stores profile snapshots for a schema/table. Drift compares recent snapshots. Current checks include added/removed columns, type/nullability changes, metric changes, outlier changes, PII changes, duplicate changes, and quality-score changes.
No-change response:
    {"changed":false,"score_delta":0,"column_changes":[]}
Endpoints:
    GET /api/v1/profiles/history?schema=public&table=users2&limit=20
    GET /api/v1/profiles/drift?schema=public&table=users2

## 24. Async Job Reliability
The worker manager uses a bounded queue and workers. Jobs are persisted so queued work can be recovered after restart. Stale-running recovery prevents crashes from permanently leaving jobs running. The current implementation does not provide a distributed worker lease across multiple instances. Production should use durable queue visibility timeouts or a database lease/claim protocol.

## 25. Security
Job API responses redact source DSNs. Source DSNs are still stored in plaintext in the current metadata design. Production should replace this with secret references or encryption. Recommended controls include TLS, least privilege, authentication/authorization, audit logging, network restrictions, safe logging, and protection of PII metadata.

## 26. Performance Design
The main strategy is database-side aggregation plus bounded Go concurrency rather than loading full tables into memory.
Current defaults include max 4 open source connections and max 2 idle connections, with configurable lifetime.
The worker queue is bounded and profiling jobs have a timeout. Sampling reduces work for large datasets when exact full-table metrics are unnecessary. Correlations are capped at 12 numeric columns.
A production design should provide profiling tiers: basic, standard, and deep.

## 27. Market Comparison
This is a capability comparison, not a performance ranking.
Go Data Profiler: Go-native, API-first database profiling with persistent jobs, SQL pushdown, quality signals, history/drift, and relational adapters.
YData Profiling: rich exploratory profiling and visual reports, primarily Python/dataframe oriented.
Great Expectations: declarative data validation and expectations.
Soda: data-quality checks, metrics, and observability workflows.
DataPrep EDA: convenient exploratory profiling/reporting for dataframe-oriented Python workflows.
The project's focus is a lightweight, Go-native, API-first database service rather than replacing every data-quality or EDA product.

## 28. Benchmark Plan
Before making any performance claim, run controlled benchmarks.
Dimensions:
- Rows: 10K, 100K, 1M, 10M, 100M+
- Columns: 5, 20, 50, 100
- Data: numeric, text, date/time, mixed
- Metrics: basic, standard, deep
- Sampling: off, 1K, 10K, 100K
- Concurrency: 1, 2, 4, 8
- Database: PostgreSQL, MySQL, SQL Server
- Network: local, same-region, higher latency
Measure wall-clock time, source DB CPU/memory, profiler CPU/memory, query count, rows scanned, bytes transferred where available, and error rate. Separate cold-cache and warm-cache runs.

## 29. Known Limitations
- Distributed worker claiming is not implemented.
- Metadata migration currently relies on Docker initialization scripts.
- Source DSNs are redacted from responses but remain plaintext in metadata storage.
- Duplicate detection can be expensive.
- Generic name headers can produce PII false positives.
- Performance has not yet been established through a controlled benchmark suite.
- Readiness should eventually verify dependencies rather than only returning an application-level ready state.

## 30. High-Value Improvement Roadmap
Reliability: real migration runner, distributed leases, durable queue.
Security: secret manager, encryption, authentication, RBAC, audit logs, tenant isolation.
Performance: profiling tiers, approximate distinct/quantiles, incremental/partition-aware profiling, smarter duplicate detection, benchmark suite.
Quality: freshness, referential integrity, domain rules, contracts, richer drift severity and before/after metrics.
Product: HTML/PDF/CSV exports, dashboard, alerts, OpenTelemetry traces, catalog integration.
PII: stronger value evidence, explainable classifications, reduced generic-header false positives.

## 31. Future Production Architecture
API Gateway/Auth -> Profiler API -> Persistent Queue -> Worker Pools -> Source DB adapters -> Profile Results -> Metadata/Object Store -> Dashboard/Alerts/Catalog.
The queue gives durable work distribution; workers can scale horizontally; metadata storage holds job and profile state; object storage can hold large reports; the API becomes a control plane rather than the execution engine.

## 32. Operational Troubleshooting
Queued jobs: check API logs, worker startup, metadata DB connectivity, worker count, and queue capacity.
Connection failures: validate DSN, DNS, network path, firewall/security group, credentials, and source permissions.
Missing metadata columns: Docker init scripts do not replay on an existing volume. Apply the migration manually or recreate the development volume.
Example:
    docker exec -i go-profiler-db psql -U profiler -d profiler_db < migrations/003_idempotency.sql
Unexpected whitespace: inspect length, octet_length, quoted value, and normalized whitespace.

## 33. Development Validation
Run:
    go test ./...
    go vet ./...
Then execute representative integration profiles against real fixtures. Fixtures should include clean data, NULLs, empty strings, whitespace-only values, malformed emails, numeric outliers, duplicates, date anomalies, and schema changes.

## 34. Suggested Demo Dataset
Create a compact fixture with unique ids, repeated names, NULL/empty/whitespace values, valid and malformed emails, skewed amounts, low-cardinality status, future dates, duplicate rows, and a related table with a foreign key. The goal is a fixture where every major profiler feature produces a result that can be explained.

## 35. How the Result Helps a Team
A profile turns raw rows into operational evidence. Data engineers can prioritize cleaning, backend engineers can detect unexpected states, analysts can understand distributions, migration teams can compare structures, and platform teams can monitor drift.
Useful workflow: profile, investigate, fix the upstream process or schema, rerun, and verify the metric changed.

## 36. Interview Explanation
I built a Go-based API-first database profiling service. The API creates persistent asynchronous jobs, a bounded worker pool executes profiling against PostgreSQL/MySQL/SQL Server adapters, and heavy aggregation stays in the source database. The result includes completeness, cardinality, distributions, numeric statistics, text/date analysis, quality checks, PII evidence, duplicates, correlations, relationships, and historical drift. I added idempotency, recovery, bounded DB connections, DSN redaction, and profile history.
For scaling questions, explain source DB capacity, query count, table size, indexes, worker concurrency, connection pools, network latency, and enabled metrics.
For production gaps, state distributed job leasing, secret management, migration tooling, benchmark evidence, and expensive deep metrics explicitly.

## 37. Research Questions
- Which metrics provide the most value per database query?
- When should approximate cardinality or quantiles replace exact calculations?
- Which profiling operations should be incremental by partition?
- How much accuracy is lost through sampling?
- What worker concurrency maximizes throughput without harming the source DB?
- How should expensive profiling jobs be isolated from production traffic?
- How should drift changes be severity-ranked?
- How can PII classification reduce false positives?
- When should duplicate detection use fingerprints instead of exact grouping?
- How should large historical profile datasets be retained and queried?

## 38. Final Mental Model
Connect -> Discover -> Measure -> Validate -> Explain -> Store -> Compare -> Act.
Connect to the source. Discover schema. Measure with database-native queries. Validate against quality rules. Explain signals such as PII, duplicates, outliers, and relationships. Store the snapshot. Compare with history. Act on the evidence.

## 39. Master Project Checklist
API validation; PostgreSQL/MySQL/SQL Server adapters; metadata DB; persistent jobs; idempotency; bounded workers; bounded DB connections; timeouts; graceful shutdown; startup recovery; stale-job recovery; basic statistics; advanced numeric statistics; text profiling; date/time profiling; PII detection; quality score; duplicate detection; correlations; relationship discovery; history; drift; DSN redaction; Docker Compose; README; user guide.
Next: benchmark suite; distributed worker lease; secret management; real migration runner; profiling tiers; richer drift; dashboard and alerting.

## 40. References and Further Reading
Repository: adm-l/data_profiler.
Project guide: docs/Go_Data_Profiler_Guide.pdf.
The repository source code, migrations, README, and tests are the authoritative implementation state. This master copy explains architecture and reasoning and should be updated whenever major capabilities or production decisions change.
