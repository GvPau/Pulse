package stream

import (
	"pulse/internal/auth"

	"github.com/go-chi/chi/v5"
)

func Router(hub *Hub) func(c chi.Router) {
	return func(r chi.Router) {
		r.Use(auth.Middleware)
		handler := newHandler(hub)
		r.Get("/stream", handler.Stream)
	}
}
