package scheduler

import (
	"context"
	"log"
	"pulse/internal/checker"
	"pulse/internal/monitor"
	"uuid"
)

type Worker struct {
	monitorRepo *monitor.Repository
	jobs        <-chan Job
}

func NewWorker(monitorRepo *monitor.Repository, jobs <-chan Job) *Worker {
	return &Worker{monitorRepo: monitorRepo, jobs: jobs}
}

func (w *Worker) Run(ctx context.Context) {
	for job := range w.jobs {
		w.Process(ctx, job)
	}
}

func (w *Worker) Process(ctx context.Context, j Job) {
	// 1. ReRead monitor
	m, err := w.monitorRepo.GetMonitorById(ctx, j.MonitorID)
	if err != nil {
		// Monitor does not exist or is not active
		log.Printf("worker: monitor %s not found, skipping: %v", j.MonitorID, err)
		return
	}

	// 2. Main Check
	result := checker.Check(ctx, m)

	// 3. Build check
	check := &monitor.Check{
		ID:             uuid.New(),
		MonitorID:      m.ID,
		StatusCode:     result.StatusCode,
		ResponseTimeMS: result.ResponseTimeMs,
		Success:        result.Success,
		Error:          result.Error,
	}

	// 4. Save
	if err := w.monitorRepo.SaveMonitorCheck(ctx, check); err != nil {
		log.Printf("worker: failed to save check for monitor %s: %v", m.ID, err)
		return
	}

	log.Printf("worker: monitor %s checked, success=%v", m.ID, result.Success)
}
