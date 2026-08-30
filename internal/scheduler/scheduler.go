package scheduler

import (
	"context"
	"log"
	"pulse/internal/monitor"
	"time"
	"uuid"
)

type Scheduler struct {
	monitorRepo *monitor.Repository
	jobs        chan<- Job    // Only writes into the channel
	interval    time.Duration // How long it runs
	nextRun     map[uuid.UUID]time.Time
}

func NewScheduler(monitorRepo *monitor.Repository, jobs chan<- Job, interval time.Duration) *Scheduler {
	return &Scheduler{
		monitorRepo: monitorRepo,
		jobs:        jobs,
		interval:    interval,
		nextRun:     make(map[uuid.UUID]time.Time),
	}
}

func (s *Scheduler) loadActiveMonitors(ctx context.Context) {
	monitors, err := s.monitorRepo.ListActive(ctx)
	if err != nil {
		log.Printf("scheduler: failed to load active monitors %v", err)
		return
	}

	for _, m := range monitors {
		// If the monitor already has a next_run, use it; otherwise start now
		if m.NextRun != nil {
			s.nextRun[m.ID] = *m.NextRun
		} else {
			s.nextRun[m.ID] = time.Now()
		}
	}

	log.Printf("scheduler: loaded %d active monitors", len(monitors))
}

func (s *Scheduler) tick(ctx context.Context) {
	now := time.Now()

	for id, next := range s.nextRun {
		if !next.After(now) {
			s.jobs <- Job{MonitorID: id}

			// Reschedule
			interval := time.Duration(0)
			if m, err := s.monitorRepo.GetMonitorById(ctx, id); err == nil {
				interval = time.Duration(m.IntervalSeconds) * time.Second
			}
			newNext := now.Add(interval)
			s.nextRun[id] = newNext
			s.monitorRepo.UpdateNextRun(ctx, id, newNext)
			log.Printf("scheduler: monitor %s next run at %s", id, newNext.Format(time.RFC3339))
		}
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.loadActiveMonitors(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick(ctx)
		case <-ctx.Done():
			log.Printf("scheduler: shutting down")
			return
		}
	}
}
