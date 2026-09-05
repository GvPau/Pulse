# Scheduler — status & next session

## Done (integrated min-heap scheduler)
Date: 2026-09-01

- `internal/scheduler/queue.go` — min-heap (`container/heap`) priority queue ordered by `nextRun`, with `byID` map for O(1) update/remove.
- `internal/scheduler/scheduler.go` — `Scheduler` holds `q *queue`. Wake-on-next with `time.NewTimer(time.Until(nextRun))` + `select` on `ctx.Done()`, dispatch in goroutine, reschedule via `update` + `UpdateNextRun`.
- `cmd/api.go` — `NewScheduler(monitorRepo, jobs)` (no interval param). `time` import removed.

## Done (hot CRUD)
Date: 2026-09-02

Solved the circular dependency (scheduler imports monitor; monitor cannot import scheduler) using **callbacks**:

- `internal/scheduler/event.go` — `Event{ Type, MonitorID }`.
- `internal/scheduler/scheduler.go`:
  - `events chan Event` (buffered, size 2) added to Scheduler.
  - `Notify(ctx, ev)` — non-blocking send (`select` + `default`) so the HTTP handler never blocks.
  - `handleEvent(ctx, ev)` — switch on Type:
    - "add" → fetch monitor, push into queue with nextRun=now if active.
    - "update" → fetch monitor; if inactive/gone remove; else recompute nextRun + `upsert`.
    - "remove" → remove from queue.
  - `Run()` select now has 3 arms: `ctx.Done`, `timer.C`, `<-s.events` (stops timer + handleEvent). When queue empty (`e==nil`), waits only on events / ctx.Done.
- `internal/scheduler/queue.go` — added `upsert` (insert if absent, else update priority).
- `internal/monitor/service.go` — `Service` holds `onCreate/onUpdate/onDelete func(ctx, id)`. `NewService` takes them. Callbacks called AFTER successful repo op (Create/Update/Delete).
- `cmd/api.go` — wires callbacks to `sched.Notify(ctx, Event{...})`. `monitorService` created AFTER scheduler.

## Done (worker pool)
Date: 2026-09-03

- `cmd/main.go` — launch N workers: `for i := 0; i < N; i++ { go app.worker.Run(ctx) }`. `numWorkers = 3`. Safe to reuse single `*Worker` because `Run` only reads `w.jobs` (a channel); no mutable state.
- `cmd/api.go` — `jobs := make(chan scheduler.Job, numWorkers)` (buffered). Buffer lets scheduler deposit up to N jobs without waiting for a free worker.

## Verified (worker pool)
- At startup, 4 checks logged at the SAME timestamp → processed in parallel (previously sequential with 1 worker).
- Wake-on-next still exact (5s spacing).
- Slight note: 4 jobs vs 3 workers at once → one job waits for a free slot (expected).

## Concept notes
- Bottleneck was the SINGLE worker (`for job := range w.jobs` processes one `Process` end-to-end, incl. HTTP, serial).
- Unbuffered channel = "rendezvous" (sender+receiver must meet); scheduler avoided blocking by dispatching in goroutine.
- Pool => N goroutines competing on same channel => parallel checks.

## Current data flow (as of 2026-09-05)

Current runtime shape: one scheduler goroutine, N=3 worker goroutines sharing a single `jobs` channel, the HTTP server in its own goroutine, and a separate events channel between the monitor CRUD and the scheduler.

- **One shared `jobs` channel** (`cmd/api.go:51`, buffered, size = `numWorkers` = 3). The scheduler writes into it as `chan<- Job`; every worker reads from it as `<-chan Job`. Each `Job` is delivered to a single available receiver (Go channel load-balancing) — there is no per-worker channel.
- **Scheduler** (`internal/scheduler/scheduler.go`): loads active monitors into the min-heap queue on startup; each loop iteration takes the heap top, sleeps with a one-shot `time.NewTimer(time.Until(nextRun))`, and `select{}`s between `timer.C`, events, and shutdown. On fire: spawns a goroutine that runs `dispatch`, then (in the loop body) reschedules by `GetMonitorById` → recompute `next = now + interval` → `UpdateNextRun` (DB) → `q.update`. `dispatch` is a non-blocking write (`select { jobs <- Job; ctx.Done() }`).
- **Workers** (`internal/scheduler/worker.go`): block on `for job := range w.jobs` and run `Process` on receipt: (1) re-read monitor, (2) `checker.Check`, (3) build `Check`, (4) `SaveMonitorCheck`, (5) `handleIncident` (success → resolve active incident; failure → open incident only when consecutive failures >= threshold and none active).
- **Events** (`cmd/api.go:56-67`, `scheduler.go:59-105`): monitor `Create`/`Update`/`Delete` call `onCreate/onUpdate/onDelete` callbacks (after the repo op, in `internal/monitor/service.go`), which call `sched.Notify(ctx, Event{Type: "add"|"update"|"remove", MonitorID})`. `Notify` sends into `s.events` (buffered, size 2) **non-blocking** — if the buffer is full the event is dropped (logged). The scheduler loop receives from `s.events` and calls `handleEvent`: `add` → push; `update` → upsert + `UpdateNextRun`; `remove` → remove from queue. Only the `monitor` entity notifies; users/incidents do not.
- **Shutdown** (graceful): on Ctrl+C the scheduler returns from `Run`, waits (`WaitGroup`) for every in-flight dispatch goroutine, and closes `s.jobs`; workers exit their `for range`; `main` shuts the HTTP server and closes the DB pool.

```
                    MAIN (goroutine) — cmd/main.go
                      go app.scheduler.Run(ctx)        (1 goroutine)
                      go app.worker.Run(ctx) x3        (3 goroutines)
                      srv.ListenAndServe()  (goroutine, HTTP :8080)
                      <-ctx.Done() => graceful shutdown (srv.Shutdown + pool.Close)
        │                          │                              │
        ▼                          ▼                              ▼
┌─────────────────┐   ┌────────────────────────┐   ┌────────────────────────────┐
│  HTTP (chi)     │   │  SCHEDULER (x1)        │   │  WORKERS (x3)              │
│  :8080          │   │                        │   │                            │
│  POST/PATCH/    │   │  loadActiveMonitors(): │   │  for job := range w.jobs   │
│  DELETE monitors│   │  ListActive -> push    │   │  (bloqueo eficiente)       │
└────────┬────────┘   └────────────┬───────────┘   └─────────────┬──────────────┘
         │                        │                             ▲
         │  CRUD (service.go)     ▼                             │
         │  onXxx callback   ┌─────────────┐                    │
         │                   │  queue      │                    │
         │                   │ min-heap    │                    │
         │                   │ por nextRun │                    │
         │                   └──────┬──────┘                    │
         │                          │ top() -> e                │
         │                          ▼                           │
         │                   time.NewTimer(Until(e.nextRun))    │
         │                          │                           │
         │                   select { ctx.Done · timer.C · <-s.events }
         │                          │        │                  │
         └──────────► s.events      │        ▼ timer.C          │
                      (buffer 2,    │   dispatch() (goroutine)  │
                      no bloqueante)│   jobs <- Job{ID}         │
         ▼                          │        │                  │
   handleEvent(ev)                  │        ▼                  │
    add   -> push                   │  + reschedule:            │
    update-> upsert + UpdateNextRun │  GetMonitorById ──────────┼───────► w.jobs
    remove-> q.remove               │  UpdateNextRun(DB)        │        (MISMO canal)
                                    │  q.update                 │        Process(ctx, job)
                                    │                           │        1. GetMonitorById
                                    │                           │        2. checker.Check
                                    │                           │        3. build Check
                                    │                           │        4. SaveMonitorCheck (DB)
                                    │                           │        5. handleIncident
                                    │                           │           success: resolve
                                    │                           │           fail: abrir si fallos
                                    │                           │                 >= threshold
                                    │                           │                 y sin incidente activo
```

- `s.jobs` and `s.events` are two separate channels. `s.events` lives inside the Scheduler; events never go into the jobs channel.
- `handleEvent` only fires from the Scheduler loop when it receives an event from `s.events` (the CRUD never touches the queue directly).

Key properties:
- `dispatch` runs in a goroutine so the scheduler never blocks even when every worker is busy; the reschedule happens in parallel with the worker's `Process`.
- The two channels are separate: `s.jobs` (scheduler → workers, one shared channel) and `s.events` (CRUD → scheduler, buffer 2, drops when full).
- `for job := range w.jobs` is an efficient blocking receive, not a spinning loop.

## Next session ideas
- Test parallelism visibly: monitor `https://httpbin.org/delay/2` (2s) — 3 slow checks finish together, not 6s serial.
- Consider extracting `numWorkers` to a shared const / env (currently duplicated in main.go & api.go).
- Consider a persistent view of `next_run` drift: with pool, check execution time approximates planned nextRun better than with 1 worker.

## Notes / observations
- Import "uuid" used everywhere but `go.mod` declares no uuid dependency — worth checking `go build`.
- `.env` is gitignored; consider an `.env.example` template.
- Minor: in `handleEvent`, when queue is full `Notify` drops events (acceptable for now given buffered size 2; revisit if monitors change faster than scheduler loops).
