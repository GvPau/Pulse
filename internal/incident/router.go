package incident

import (
	"pulse/internal/auth"

	"github.com/go-chi/chi/v5"
)

func Router(service *Service) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(auth.Middleware)

		handler := NewHandler(service)

		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
	}
}
