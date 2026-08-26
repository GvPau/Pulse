# Pulse Architecture

Pulse is a Go backend with a React frontend and PostgreSQL database.

The backend initially runs as a single Go process containing:

- HTTP API
- Scheduler
- Worker pool
- HTTP checker

PostgreSQL is shared by all components.

## Project Structure

    pulse/
    ├── cmd/
    │   └── api/
    │       └── main.go
    │
    ├── internal/
    │   ├── monitor/
    │   │   ├── handler.go
    │   │   ├── service.go
    │   │   ├── repository.go
    │   │   └── model.go
    │   │
    │   ├── incident/
    │   │   ├── service.go
    │   │   ├── repository.go
    │   │   └── model.go
    │   │
    │   ├── user/
    │   │   ├── repository.go
    │   │   └── model.go
    │   │
    │   ├── checker/
    │   │   └── checker.go
    │   │
    │   ├── scheduler/
    │   │   └── scheduler.go
    │   │
    │   └── database/
    │       └── postgres.go
    │
    ├── migrations/
    ├── go.mod
    ├── go.sum
    ├── Dockerfile
    └── docker-compose.yml

## File Responsibilities

### `cmd/api/main.go`

Application entry point.

Responsible for initializing and starting:

- PostgreSQL connection
- Services
- Job channel
- Workers
- Scheduler
- HTTP server

It also handles graceful shutdown.

### `internal/monitor/`

Everything related to monitors.

- `handler.go` — HTTP handlers for monitor endpoints.
- `service.go` — Monitor business logic.
- `repository.go` — PostgreSQL queries for monitors.
- `model.go` — Monitor data structures.

### `internal/incident/`

Everything related to incidents.

- `service.go` — Incident creation, updating and resolution.
- `repository.go` — PostgreSQL queries for incidents.
- `model.go` — Incident data structures.

### `internal/user/`

User-related database operations.

- `repository.go` — PostgreSQL queries for users.
- `model.go` — User data structures.

Authentication will be added later and will remain separate from the monitoring domain.

### `internal/checker/checker.go`

Responsible only for executing HTTP checks.

It receives a monitor and returns the result:

- success/failure
- HTTP status code
- response time
- error

The checker does not know about HTTP API handlers or React.

### `internal/scheduler/scheduler.go`

Responsible for deciding **when a monitor needs to be checked**.

The scheduler:

1. Loads active monitors.
2. Keeps track of when each monitor should run next.
3. Creates a job when a monitor is due.
4. Sends the job to the job channel.
5. Updates the monitor's next execution time.

The scheduler does not execute HTTP requests itself.

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
