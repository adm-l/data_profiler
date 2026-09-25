# API

### POST /api/v1/profile-jobs

```json
{"source":{"type":"postgres","dsn":"postgres://..."},"schema":"public","table":"customers","sample_size":0}
```

Returns `202 Accepted` with a job ID.

### GET /api/v1/profile-jobs/{id}

Returns queued/running/completed/failed status.

### GET /api/v1/profile-jobs/{id}/result

Returns the table profile and quality report.

### GET /metrics

Prometheus endpoint.
