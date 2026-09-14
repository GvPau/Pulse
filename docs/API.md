# Pulse API Reference

Single-process Go service (`cmd`). All endpoints are scoped to the
authenticated user unless noted otherwise.

## Authentication

```
POST /auth/register  {email, password}        → 200 {token}   (JWT, no expiry)
POST /auth/login     {email, password}        → 200 {token}   (JWT, no expiry)
```

Subsequent requests carry `Authorization: Bearer <token>`.
Errors follow the standard error contract (see below).

---

## Pagination

All list endpoints accept `?page=` (default 1) and `?limit=` (default 20,
max 100). Response envelope:

```json
{
  "data":        [ ... ],
  "pagination":  { "page": 1, "limit": 20, "total": 42, "has_more": true }
}
```

---

## Monitors

### `GET /monitors`

List monitors with derived status. Query params:

| Param | Type | Description |
|-------|------|-------------|
| `page` | int | Page number (1-based) |
| `limit` | int | Items per page (1–100) |
| `q` | string | Case-insensitive name substring |
| `active` | bool | Filter by active state |
| `sort` | string | `name`, `created_at`, `interval_seconds` |
| `order` | string | `asc` (default) or `desc` |
| `window` | string | `24h` (default), `7d`, `30d`, `90d` |

Returns `MonitorListResponse` (`MonitorWithStatus` items).

### `GET /monitors/{id}`

Single monitor with derived status in the requested window.

### `POST /monitors`

Create. Body = `MonitorRequest` (required fields: `name`, `url`, `method`,
`expected_status`, `interval_seconds`, `timeout_seconds`). Optional:
`active` (default false), `failure_threshold` (default 3). Emits
`monitor.created` on the SSE stream.

### `PUT /monitors/{id}`

Update. Body = `MonitorRequest`; `name` and `url` are required.
Responses: 404 `monitor not found`, 200 `{"message":"ok"}`. Emits
`monitor.updated`.

### `DELETE /monitors/{id}`

Delete. Emits `monitor.deleted`.

---

## Checks

### `GET /monitors/{id}/checks`

Paginated check history for a monitor. Query params:

| Param | Type | Description |
|-------|------|-------------|
| `page`, `limit` | int | Pagination |
| `success` | bool | Filter by result |

### `GET /monitors/{id}/metrics`

Availability summary + time series. Query param `window` (default 24h).

---

## Incidents

### `GET /incidents`

Paginated incident list, newest first. Query params:

| Param | Type | Description |
|-------|------|-------------|
| `page`, `limit` | int | Pagination |
| `monitor_id` | uuid | Filter by monitor |
| `status` | string | `active` or `resolved` |

Returns `IncidentListResponse` (Incident items).

### `GET /incidents/{id}`

Single incident.

---

## SSE Stream

### `GET /stream`

Server-Sent Events for the authenticated user. Accepts the JWT via
`Authorization: Bearer <token>` **or** `?token=<jwt>` (for browser
`EventSource`, which cannot send custom headers).

Event types (stable contract, constants in `internal/stream/constants.go`):

| Event | Payload |
|-------|---------|
| `check.completed` | Check |
| `incident.opened` | Incident |
| `incident.resolved` | Incident |
| `monitor.created` | Monitor |
| `monitor.updated` | Monitor |
| `monitor.deleted` | `{ "id": "<uuid>" }` |

Heartbeat `: ping` every 15 s. Slow clients are dropped without
affecting other subscribers.

---

## Health

| Endpoint | Meaning |
|----------|---------|
| `GET /healthz` | Liveness: always 200 |
| `GET /readyz` | Readiness: pool.Ping with 2 s timeout; 503 on failure |

---

## Error contract

All errors follow a single shape:

```json
{
  "error": {
    "code":    "invalid_request | unauthorized | not_found | conflict | internal_error",
    "message": "Human-readable description",
    "details": { "field": "error" }       // only for 400 validation errors
  }
}
```

Status codes: 400 (validation / bad request), 401 (auth), 404 (not found),
409 (conflict), 500 (internal).
