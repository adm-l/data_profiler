# AWS production topology

```text
Route53
  -> CloudFront/WAF (optional)
  -> ALB
  -> EKS Ingress
  -> Data Profiler API pods
       |\
       | +--> ElastiCache Redis (queue/cache)
       | +--> RDS PostgreSQL (metadata/results)
       | +--> Secrets Manager
       | +--> CloudWatch / Prometheus / Grafana / OpenTelemetry
       |
       +--> source databases (RDS/Aurora/Snowflake/SQL Server/etc.)
```

For event-driven scale, put Kafka/MSK between API and worker consumers. Persist raw profile snapshots to S3 for long-term history and keep queryable metadata in RDS.

Use IAM roles for service accounts (IRSA), private subnets for EKS/RDS/Redis, security groups allowing only required ports, KMS encryption, TLS, secret rotation, and a read-only source DB role.
