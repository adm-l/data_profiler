# Scaling strategy

## Small databases
Use the in-process queue and bounded column workers.

## Medium production
Use Redis as a durable-ish work queue and run multiple API/worker replicas. Persist every job in PostgreSQL before enqueueing.

## High volume
Use Kafka/MSK for profiling-job events. API publishes `profile.requested`; workers consume it with consumer groups. Store results in PostgreSQL plus S3 for historical snapshots.

## Very large tables
Do not blindly run `COUNT(DISTINCT ...)` and full distributions on every column. Introduce:
- configurable row sampling
- approximate cardinality (HyperLogLog)
- scan budgets
- priority columns
- incremental profiling
- partition-aware scans
- warehouse-native statistics where available
