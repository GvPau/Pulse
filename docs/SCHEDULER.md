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

## Next session ideas
- Test parallelism visibly: monitor `https://httpbin.org/delay/2` (2s) — 3 slow checks finish together, not 6s serial.
- Consider extracting `numWorkers` to a shared const / env (currently duplicated in main.go & api.go).
- Consider a persistent view of `next_run` drift: with pool, check execution time approximates planned nextRun better than with 1 worker.

## Notes / observations
- Import "uuid" used everywhere but `go.mod` declares no uuid dependency — worth checking `go build`.
- `.env` is gitignored; consider an `.env.example` template.
- Minor: in `handleEvent`, when queue is full `Notify` drops events (acceptable for now given buffered size 2; revisit if monitors change faster than scheduler loops).
