package user

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Router(pool *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		repo := NewRepository(pool)
		service := NewService(repo)
		handler := NewHandler(service)

		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
	}
}
