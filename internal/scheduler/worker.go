package scheduler

import (
	"context"
	"errors"
	"log"
	"pulse/internal/checker"
	"pulse/internal/incident"
	"pulse/internal/monitor"
	"time"
	"uuid"
)

type Worker struct {
	monitorRepo  *monitor.Repository
	incidentRepo *incident.Repository
	jobs         <-chan Job
}

func NewWorker(monitorRepo *monitor.Repository, incidentRepo *incident.Repository, jobs <-chan Job) *Worker {
	return &Worker{monitorRepo: monitorRepo, incidentRepo: incidentRepo, jobs: jobs}
}

func (w *Worker) Run(ctx context.Context) {
	for job := range w.jobs {
		w.Process(ctx, job)
	}
}

func (w *Worker) handleIncident(ctx context.Context, m *monitor.Monitor, success bool) {
	if success {
		// Recovered -> resolve any active incident
		active, err := w.incidentRepo.FindActiveByMonitor(ctx, m.ID)
		if err == nil {
			w.incidentRepo.Resolve(ctx, active.ID, time.Now())
			log.Printf("worker: incident %s resolved for monitor %s", active.ID, m.ID)
		}
		return
	}

	// Failing -> count consecutive failures
	count, err := w.monitorRepo.CountConsecutiveFailures(ctx, m.ID)
	if err != nil {
		log.Printf("worker: failed to count failures for monitor %s: %v", m.ID, err)
		return
	}

	// Open an incident only when the threshold is reached and none is active
	if count >= m.FailureThreshold {
		_, err := w.incidentRepo.FindActiveByMonitor(ctx, m.ID)
		if errors.Is(err, incident.ErrNotFound) {
			inc := &incident.Incident{
				ID:           uuid.New(),
				MonitorID:    m.ID,
				StartedAt:    time.Now(),
				Status:       "active",
				FailureCount: count,
				CreatedAt:    time.Now(),
			}
			w.incidentRepo.Create(ctx, inc)
			log.Printf("worker: incident opened for monitor %s (failures=%d)", m.ID, count)
		}

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

	// 5. Handle incident state
	w.handleIncident(ctx, m, result.Success)

	log.Printf("worker: monitor %s checked, success=%v", m.ID, result.Success)
}
