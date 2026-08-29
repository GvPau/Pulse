package monitor

import (
	"pulse/internal/auth"

	"github.com/go-chi/chi/v5"
)

func Router(service *Service) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(auth.Middleware)

		handler := NewHandler(service)

		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Put("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	}
}
