# Pulse Architecture

Pulse is a Go backend with a React frontend and PostgreSQL database. Frontends live in separate repos (not built yet).

The backend initially runs as a single Go process containing:

- HTTP API
- Scheduler
- Worker pool
- HTTP checker

PostgreSQL is shared by all components.

## Project Structure

    pulse/
    ├── cmd/
    │   ├── main.go          (entry point + launch)
    │   └── api.go           (dependency wiring / composition)
    │
    ├── internal/
    │   ├── monitor/
    │   │   ├── handler.go
    │   │   ├── router.go
    │   │   ├── service.go
    │   │   ├── repository.go
    │   │   ├── monitor.go
    │   │   └── monitor_check_repository.go
    │   │
    │   ├── incident/
    │   │   ├── handler.go
    │   │   ├── router.go
    │   │   ├── service.go
    │   │   ├── repository.go
    │   │   └── incident.go
    │   │
    │   ├── user/
    │   │   ├── handler.go
    │   │   ├── router.go
    │   │   ├── service.go
    │   │   ├── repository.go
    │   │   └── user.go
    │   │
    │   ├── auth/
    │   │   ├── jwt.go
    │   │   └── middleware.go
    │   │
    │   ├── scheduler/
    │   │   ├── scheduler.go
    │   │   ├── queue.go
    │   │   ├── worker.go
    │   │   ├── job.go
    │   │   └── event.go
    │   │
    │   ├── checker/
    │   │   └── checker.go
    │   │
    │   ├── shared/
    │   │   └── model.go
    │   │
    │   └── database/
    │       └── postgres.go
    │
    ├── migrations/
    ├── docs/
    ├── go.mod
    ├── go.sum
    ├── docker-compose.yml
    └── (no Dockerfile yet — single process, one pod planned)

## File Responsibilities

### `cmd/main.go`

Application entry point. Signals a context, starts scheduler + worker goroutines and the HTTP server, and handles graceful shutdown.

### `cmd/api.go`

Composition/wiring (not a handler): builds repositories, services, the scheduler and workers, and the chi router. It also wires the monitor service to the scheduler via event callbacks (`add` / `update` / `remove`) and gives the workers the same `sched.Notify` so they can report ghost monitors.

### `internal/monitor/`

Everything related to monitors.

- `handler.go` — HTTP handlers for monitor endpoints.
- `router.go` — chi router.
- `service.go` — Monitor business logic + scheduler event callbacks.
- `repository.go` — PostgreSQL queries for monitors.
- `monitor.go` — Monitor data structures.
- `monitor_check_repository.go` — check result queries.

### `internal/incident/`

Everything related to incidents.

- `handler.go` — HTTP handlers.
- `router.go` — chi router.
- `service.go` — Incident creation, updating and resolution.
- `repository.go` — PostgreSQL queries for incidents.
- `incident.go` — Incident data structures.

### `internal/user/`

User-related database operations.

- `handler.go` — HTTP handlers.
- `router.go` — chi router.
- `service.go` — User business logic (registration, login).
- `repository.go` — PostgreSQL queries for users.
- `user.go` — User data structures.

### `internal/auth/`

Authentication, separate from the monitoring domain.

- `jwt.go` — JWT signing/validation.
- `middleware.go` — protects routes that require a logged-in user.

### `internal/checker/checker.go`

Responsible only for executing HTTP checks.

It receives a monitor and returns the result:

- success/failure
- HTTP status code
- response time
- error

The checker does not know about HTTP API handlers or React.

### `internal/scheduler/`

Responsible for deciding **when a monitor needs to be checked**.

- `scheduler.go` — main loop, event handling, and the `runFlusher` goroutine.
- `queue.go` — min-heap priority queue ordered by `nextRun`.
- `worker.go` — executes jobs (via the checker) and manages incidents.
- `job.go` / `event.go` — `Job` and `Event` types.

The scheduler:

1. Loads active monitors.
2. Keeps track of when each monitor should run next (in memory, from each monitor's interval).
3. Creates a job when a monitor is due and sends it to the job channel.
4. Reschedules in memory without DB calls (Option B); a flusher goroutine persists `next_run` batches every second.

### `internal/database/postgres.go`

Responsible for creating and configuring the PostgreSQL connection.

### `migrations/`

Contains versioned database schema changes.

Example:

    001_create_users.sql
    002_create_monitors.sql
    003_create_monitor_checks.sql
    004_create_incidents.sql

## Request Architecture

Normal API requests are synchronous.

    React
      │
      │ HTTP
      ▼
    Handler
      │
      ▼
    Service
      │
      ▼
    Repository
      │
      ▼
    PostgreSQL
      │
      ▼
    Response

For example:

    POST /api/v1/monitors

The handler validates the request and calls the monitor service.

The service contains the business logic.

The repository performs the PostgreSQL query.

## Monitoring Architecture

Monitoring runs independently from HTTP requests.

    Scheduler
        │
        ▼
    Job Channel
        │
        ├── Worker 1
        ├── Worker 2
        └── Worker 3
              │
              ▼
           Checker
              │
              ▼
        Check Result
              │
              ▼
        Monitor Service
              │
        ┌─────┴─────┐
        ▼           ▼
    PostgreSQL   Incident Logic

## Job Queue

The initial job queue will simply be a Go channel.

    Scheduler
        │
        ▼
    chan Job
        │
        ├── Worker
        ├── Worker
        └── Worker

We do not need Redis, RabbitMQ, Kafka or another external message broker for the MVP.

If Pulse grows and requires distributed workers, an external queue can be introduced later.

## Workers

Workers wait for jobs from the channel.

When a worker receives a job:

1. Load the monitor.
2. Execute the HTTP request using the checker.
3. Store the result in `monitor_checks`.
4. Create or update an incident when necessary.

Workers run concurrently using Go goroutines.

## Different Monitor Intervals

Monitors can have different intervals.

Example:

    Monitor A → 1 minute
    Monitor B → 5 minutes
    Monitor C → 15 minutes
    Monitor D → 1 hour

The scheduler keeps track of the next execution time for each active monitor.

It does not need to create a separate ticker for every monitor.

Conceptually:

    Monitor A → next run: 12:01
    Monitor B → next run: 12:05
    Monitor C → next run: 12:15
    Monitor D → next run: 13:00

When a monitor is due, the scheduler sends a job to the channel and calculates its next execution time.

## Application Startup

`scheduler.go` is not an endpoint and is not started by an HTTP request.

`main.go` starts all application components.

    main.go
       │
       ├── Connect PostgreSQL
       ├── Create services
       ├── Create Job Channel
       ├── Start Workers
       ├── Start Scheduler
       └── Start HTTP Server

The scheduler and workers run as goroutines inside the same Go process as the API.

## Graceful Shutdown

The application uses Go's `context.Context` for shutdown.

When the application receives a shutdown signal:

    SIGTERM / Ctrl+C
            │
            ▼
    Cancel Context
            │
       ┌────┼────┐
       ▼    ▼    ▼
      API Scheduler Workers
       │    │    │
       └────┴────┘
            │
            ▼
       Close resources

The scheduler and workers must stop cleanly instead of running indefinitely.

## Initial Deployment Model

For the MVP, everything runs in one Go process:

    ┌─────────────────────────────┐
    │            Pulse            │
    │                             │
    │  HTTP API                   │
    │  Scheduler                  │
    │  Worker Pool                │
    │  HTTP Checker               │
    │                             │
    └──────────────┬──────────────┘
                   │
                   ▼
              PostgreSQL

This keeps the architecture simple.

If the application grows, the scheduler and workers can later be separated into independent services or use an external message queue.

## Core Design Principle

Keep responsibilities separated:

- **Handler** → HTTP
- **Service** → business logic
- **Repository** → PostgreSQL
- **Scheduler** → decides when
- **Worker** → executes jobs
- **Checker** → performs HTTP checks
- **Incident service** → manages incidents

The MVP should remain a single Go application with concurrent goroutines rather than introducing distributed infrastructure prematurely.
