# All-phase implementation roadmap

## Phase 1
- Go service, REST API, PostgreSQL metadata store
- Metadata discovery and concurrent column profiling
- Docker and tests

## Phase 2
- Adapter abstraction
- PostgreSQL/MySQL/SQL Server implementations
- Snowflake boundary isolated for driver integration

## Phase 3
- Completeness/uniqueness quality checks
- Quality scoring

## Phase 4
- Async worker manager
- Redis/Kafka deployment boundary
- Kubernetes probes/HPA
- Prometheus metrics endpoint
- AWS architecture

## Phase 5
- PII heuristic detection
- Snapshot storage
- Drift comparison package
- Recommended next production additions: S3 snapshot archival, OTel tracing, Kafka implementation, sampling/approximate algorithms, RBAC, audit log, UI, scheduler, notifications.
