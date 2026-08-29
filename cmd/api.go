package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"pulse/internal/database"
	"pulse/internal/monitor"
	"pulse/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type api struct {
	router *chi.Mux
	pool   *pgxpool.Pool
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
	userService := user.NewService(userRepo)

	monitorRepo := monitor.NewRepository(pool)
	monitorService := monitor.NewService(monitorRepo)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/auth", user.Router(userService))
	r.Route("/monitors", monitor.Router(monitorService))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	return &api{router: r, pool: pool}, nil
}
