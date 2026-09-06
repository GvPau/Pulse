package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"pulse/internal/database"
	"pulse/internal/incident"
	"pulse/internal/monitor"
	"pulse/internal/scheduler"
	"pulse/internal/user"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type api struct {
	router    *chi.Mux
	pool      *pgxpool.Pool
	scheduler *scheduler.Scheduler
	worker    *scheduler.Worker
}

func newAPI(ctx context.Context) (*api, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL environment variable is not set")
	}

	pool, err := database.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	log.Printf("Database up")

	// Dependecies
	userRepo := user.NewRepository(pool)
	monitorRepo := monitor.NewRepository(pool)
	incidentRepo := incident.NewRepository(pool)

	userService := user.NewService(userRepo)
	incidentService := incident.NewService(incidentRepo)

	// Scheduler and Worker
	const numWorkers = 3
	jobs := make(chan scheduler.Job, numWorkers)
	sched := scheduler.NewScheduler(monitorRepo, jobs)
	wrk := scheduler.NewWorker(monitorRepo, incidentRepo, jobs,
		func(ctx context.Context, ev scheduler.Event) { sched.Notify(ctx, ev) })

	monitorService := monitor.NewService(monitorRepo,
		func(ctx context.Context, id uuid.UUID) {
			// Notify the scheduler about the new monitor
			sched.Notify(ctx, scheduler.Event{Type: "add", MonitorId: id})
		},
		func(ctx context.Context, id uuid.UUID) {
			// Notify the scheduler about the updated monitor
			sched.Notify(ctx, scheduler.Event{Type: "update", MonitorId: id})
		},
		func(ctx context.Context, id uuid.UUID) {
			// Notify the scheduler about the deleted monitor
			sched.Notify(ctx, scheduler.Event{Type: "remove", MonitorId: id})
		})

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/auth", user.Router(userService))
	r.Route("/monitors", monitor.Router(monitorService))
	r.Route("/incidents", incident.Router(incidentService))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	return &api{router: r, pool: pool, scheduler: sched, worker: wrk}, nil
}
