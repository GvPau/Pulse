package scheduler

import (
	"context"
	"log"
	"pulse/internal/monitor"
	"sync"
	"time"
	"uuid"
)

type nextRunUpdate struct {
	monitorID uuid.UUID
	nextRun   time.Time
}

type Scheduler struct {
	monitorRepo *monitor.Repository
	jobs        chan<- Job         // Only writes into the channel
	q           *queue             // Priority queue for scheduling
	events      chan Event         // Channel for receiving monitor events
	flush       chan nextRunUpdate // Batch updates for nextRun
}

func NewScheduler(monitorRepo *monitor.Repository, jobs chan<- Job) *Scheduler {
	return &Scheduler{
		monitorRepo: monitorRepo,
		jobs:        jobs,
		q:           newQueue(),
		events:      make(chan Event, 2),           // Buffered channel to avoid blocking
		flush:       make(chan nextRunUpdate, 256), // 256 Max of updates per second in flush
	}
}

func (s *Scheduler) loadActiveMonitors(ctx context.Context) {
	monitors, err := s.monitorRepo.ListActive(ctx)
	if err != nil {
		log.Printf("scheduler: failed to load active monitors %v", err)
		return
	}

	for _, m := range monitors {
		next := time.Now()
		if m.NextRun != nil {
			next = *m.NextRun
		}

		s.q.push(&entry{
			monitorID:       m.ID,
			nextRun:         next,
			intervalSeconds: m.IntervalSeconds,
		})
	}

	log.Printf("scheduler: loaded %d active monitors", len(monitors))
}

// dispatch sends a job to the worker without blocking the scheduler loop.
func (s *Scheduler) dispatch(ctx context.Context, monitorID uuid.UUID) {
	select {
	case s.jobs <- Job{MonitorID: monitorID}:
	case <-ctx.Done():
	}

}

func (s *Scheduler) Notify(ctx context.Context, ev Event) {
	select {
	case s.events <- ev:
	default:
		log.Printf("scheduler: event queue full, dropping event for monitor %s", ev.MonitorId)
	}
}

func (s *Scheduler) handleEvent(ctx context.Context, ev Event) {
	switch ev.Type {
	case "add":
		m, err := s.monitorRepo.GetMonitorById(ctx, ev.MonitorId)

		if err != nil || !m.Active {
			return
		}

		next := time.Now()
		if m.NextRun != nil {
			next = *m.NextRun
		}

		s.q.push(&entry{monitorID: m.ID, nextRun: next, intervalSeconds: m.IntervalSeconds})
		log.Printf("scheduler: added monitor %s to queue", ev.MonitorId)

	case "update":
		m, err := s.monitorRepo.GetMonitorById(ctx, ev.MonitorId)
		if err != nil || !m.Active {
			s.q.remove(ev.MonitorId) // no-op since the monitor is not active or doesn't exist
			log.Printf("scheduler: removed monitor %s (inactive or gone)", ev.MonitorId)
			return
		}

		next := time.Now().Add(time.Duration(m.IntervalSeconds) * time.Second)
		if err := s.monitorRepo.UpdateNextRun(ctx, ev.MonitorId, next); err != nil {
			log.Printf("scheduler: failed to update next run for monitor %s: %v", ev.MonitorId, err)
		}

		s.q.upsert(&entry{monitorID: m.ID, nextRun: next, intervalSeconds: m.IntervalSeconds})
		log.Printf("scheduler: updated monitor %s in queue", ev.MonitorId)

	case "remove":
		s.q.remove(ev.MonitorId)
		log.Printf("scheduler: removed monitor %s from queue", ev.MonitorId)

	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.loadActiveMonitors(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go s.runFlusher(ctx, &wg)
	defer func() {
		wg.Wait()
		close(s.jobs)
	}()

	for {
		e := s.q.top()
		if e == nil {
			// No monitors in queue - wait for an event or shutdown
			select {
			case ev := <-s.events:
				s.handleEvent(ctx, ev)
				continue
			case <-ctx.Done():
				log.Printf("scheduler: shutting down")
				return
			}
		}

		// Sleep until the next monitor is due to run or until the context is canceled
		timer := time.NewTimer(time.Until(e.nextRun))
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Printf("scheduler: shutting down")
			return
		case <-timer.C:

			// Send the job to the worker in a goroutine so the scheduler keeps dispatching even if the worker is busy.
			wg.Add(1)
			go func(monitorID uuid.UUID) {
				defer wg.Done()
				s.dispatch(ctx, monitorID)
			}(e.monitorID)

			// Reschedule in memory using the interval store in the entry (no DB call)
			if e.intervalSeconds <= 0 {
				log.Printf("scheduler: monitor %s has invalid interval %d, removing from queue", e.monitorID, e.intervalSeconds)
				s.q.remove(e.monitorID)
				continue
			}

			newNext := time.Now().Add(time.Duration(e.intervalSeconds) * time.Second)
			s.q.update(e.monitorID, newNext)

			// Queue persisted next_run for the batched flusher (non-blocking)
			select {
			case s.flush <- nextRunUpdate{monitorID: e.monitorID, nextRun: newNext}:
			default:
				log.Printf("scheduler: flush queue full, skipping db next_run for monitor %s", e.monitorID)
			}
			log.Printf("scheduler: rescheduled monitor %s to run at %s", e.monitorID, newNext.Format(time.RFC3339))

		case ev := <-s.events:
			timer.Stop() // Stop the timer to avoid leaking resources
			s.handleEvent(ctx, ev)
		}
	}
}

func (s *Scheduler) runFlusher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	pending := make(map[uuid.UUID]time.Time)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	flush := func(flushCtx context.Context) {
		if len(pending) == 0 {
			return
		}

		patches := make([]monitor.NextRunPatch, 0, len(pending))
		for id, next := range pending {
			patches = append(patches, monitor.NextRunPatch{ID: id, NextRun: next})
		}
		pending = make(map[uuid.UUID]time.Time)
		if err := s.monitorRepo.UpdateNextRuns(flushCtx, patches); err != nil {
			log.Printf("scheduler: batch next_run update failed %v", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush(context.WithoutCancel(ctx))
			return
		case u := <-s.flush:
			pending[u.monitorID] = u.nextRun
		case <-ticker.C:
			flush(ctx)
		}
	}

}
