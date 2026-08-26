# Database Schema

Pulse uses PostgreSQL.

The initial database is intentionally small and focused on the core monitoring functionality.

## Schema

    ┌──────────────┐
    │    users     │
    ├──────────────┤
    │ id PK        │
    │ email        │
    │ created_at   │
    │ updated_at   │
    └──────┬───────┘
           │
           │ 1:N
           ▼
    ┌──────────────────┐
    │     monitors     │
    ├──────────────────┤
    │ id PK            │
    │ user_id FK       │
    │ name             │
    │ url              │
    │ method           │
    │ interval_seconds │
    │ timeout_seconds  │
    │ expected_status  │
    │ active           │
    │ created_at       │
    │ updated_at       │
    └────────┬─────────┘
             │
             ├─────────────────────┐
             │ 1:N                 │ 1:N
             ▼                     ▼
    ┌──────────────────┐    ┌────────────────┐
    │ monitor_checks   │    │   incidents    │
    ├──────────────────┤    ├────────────────┤
    │ id PK            │    │ id PK          │
    │ monitor_id FK    │    │ monitor_id FK  │
    │ status_code      │    │ started_at     │
    │ response_time_ms │    │ resolved_at    │
    │ success          │    │ status         │
    │ error            │    │ failure_count  │
    │ checked_at       │    │ created_at     │
    └──────────────────┘    └────────────────┘

## Tables

### users

Represents a Pulse user.

| Column | Description |
|---|---|
| `id` | Unique user identifier |
| `email` | User email address |
| `created_at` | Account creation time |
| `updated_at` | Last update time |

Authentication data is intentionally kept out of this schema for now.

### monitors

Represents an endpoint that Pulse periodically monitors.

| Column | Description |
|---|---|
| `id` | Unique monitor identifier |
| `user_id` | Owner of the monitor |
| `name` | Human-readable monitor name |
| `url` | Endpoint URL |
| `method` | HTTP method |
| `interval_seconds` | Time between checks |
| `timeout_seconds` | Request timeout |
| `expected_status` | Expected HTTP status code |
| `active` | Whether the monitor is enabled |
| `created_at` | Creation time |
| `updated_at` | Last update time |

Example monitor:

    name: "Production API"
    url: "https://api.example.com/health"
    method: GET
    interval_seconds: 300
    timeout_seconds: 10
    expected_status: 200
    active: true

### monitor_checks

Represents a single execution of a monitor.

Every scheduled check creates one record.

| Column | Description |
|---|---|
| `id` | Unique check identifier |
| `monitor_id` | Monitor being checked |
| `status_code` | HTTP response status, if available |
| `response_time_ms` | Response time in milliseconds |
| `success` | Whether the check passed |
| `error` | Error message when the request fails |
| `checked_at` | Time of the check |

A check is successful when the request succeeds and the received status code matches the monitor's expected status.

A timeout or connection failure is stored as a failed check.

### incidents

Represents a period of downtime.

Multiple consecutive failed checks belong to the same incident.

Example:

    10:00  ✓
    10:05  ✓
    10:10  ✗
    10:15  ✗
    10:20  ✗
    10:25  ✓

This creates one incident:

    status: resolved
    started_at: 10:10
    resolved_at: 10:25
    failure_count: 3

| Column | Description |
|---|---|
| `id` | Unique incident identifier |
| `monitor_id` | Affected monitor |
| `started_at` | Incident start time |
| `resolved_at` | Incident resolution time |
| `status` | `active` or `resolved` |
| `failure_count` | Number of failed checks |
| `created_at` | Incident creation time |

## Relationships

- A user can have many monitors.
- A monitor can have many checks.
- A monitor can have many incidents.

The relationship can be summarized as:

    users
      └──< monitors
              ├──< monitor_checks
              └──< incidents

## Design Decisions

### Store checks instead of metrics

Metrics such as uptime percentage and average response time will be calculated from `monitor_checks`.

We should avoid storing derived values such as `uptime_percentage` directly on `monitors`.

### Store intervals in seconds

`interval_seconds` is stored as an integer.

Examples:

- `60` = 1 minute
- `300` = 5 minutes
- `900` = 15 minutes
- `3600` = 1 hour

This makes the value easy to work with from Go.

### Failed checks vs incidents

A failed check represents a single failed request.

An incident represents a period of downtime and can contain multiple failed checks.

## MVP Scope

The initial database contains only:

    users
    monitors
    monitor_checks
    incidents

Authentication, notifications, billing, API assertions, webhooks, and other future features will be added when those features are implemented.
