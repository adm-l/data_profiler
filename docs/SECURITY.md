# Security checklist

- Do not accept raw production DB credentials from arbitrary users.
- Prefer secret references resolved server-side from AWS Secrets Manager or Kubernetes Secrets.
- Use read-only source DB users.
- Enforce TLS to source databases and metadata stores.
- Add authentication/RBAC before exposing the API outside a trusted network.
- Audit profile requests because profiling may expose sensitive metadata.
- Treat PII detection as a heuristic classifier, not a legal/compliance determination.
- Mask sensitive values in top-value output before production use.
- Add request limits and table/row scan budgets to prevent accidental full-table workloads.
