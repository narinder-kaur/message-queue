# Metrics API Documentation

Base URL: `/api/v1`

Authentication
- The metrics API supports Bearer token authentication via the `Authorization` header.
- Set the token using the `AUTH_TOKEN` environment variable in the `metrics` service. If empty, auth is disabled.

Endpoints

1) GET /api/v1/gpus

- Summary: Returns a paginated list of GPU IDs known to the system.
- Query parameters:
  - `limit` (int, optional) — page size, defaults to `DEFAULT_PAGE_SIZE` (see `.env.example`).
  - `page` (int, optional) — page number (1-based), default `1`.
  - `sort` (string, optional) — `asc` or `desc` (default `asc`).
- Response (200):

```json
{
  "total": 42,
  "page": 1,
  "limit": 10,
  "items": ["gpu-1","gpu-2",...]
}
```

- Error responses:
  - 400 Bad Request — invalid pagination parameters
  - 401 Unauthorized — when `AUTH_TOKEN` is set and missing/invalid bearer token
  - 500 Internal Server Error — store/query failures


2) GET /api/v1/gpus/{id}/telemetry

- Summary: Query telemetry for a specific GPU.
- Path parameters:
  - `id` (string) — GPU identifier
- Query parameters:
  - `start_time` (string, optional) — RFC3339 timestamp (e.g. `2025-07-18T13:42:33Z`)
  - `end_time` (string, optional) — RFC3339 timestamp
  - `limit` (int, optional) — page size
  - `page` (int, optional) — page number
  - `sort` (string, optional) — `asc` or `desc`
- Response (200):

```json
{
  "total": 123,
  "page": 1,
  "limit": 50,
  "items": [
    {"timestamp":"2025-07-18T13:42:33Z","gpu_id":"gpu-1","metrics":{...}},
    ...
  ]
}
```

- Error responses:
  - 400 Bad Request — missing `id` in path or invalid time format
  - 401 Unauthorized — when `AUTH_TOKEN` is set and missing/invalid bearer token
  - 500 Internal Server Error — store/query failures


Health endpoints
- GET `/healthz` — returns 200/OK
- GET `/ready` — returns 200/ready

Content-Type
- All successful JSON responses are served with `Content-Type: application/json`.

Examples (curl)

List GPUs (page 1):

```sh
curl -H "Authorization: Bearer ${AUTH_TOKEN}" "http://localhost:8080/api/v1/gpus?limit=20&page=1"
```

Query telemetry between time ranges:

```sh
curl -H "Authorization: Bearer ${AUTH_TOKEN}" "http://localhost:8080/api/v1/gpus/gpu-1/telemetry?start_time=2025-07-18T13:00:00Z&end_time=2025-07-18T14:00:00Z"
```

Schema reference
- `ListResponse`:
  - `total` (int)
  - `page` (int)
  - `limit` (int)
  - `items` ([]string)

- `QueryResponse`:
  - `total` (int)
  - `page` (int)
  - `limit` (int)
  - `items` ([]object) — telemetry records (shape may vary depending on ingestion)
