package user

import (
	"github.com/go-chi/chi/v5"
)

func Router(service *Service) func(r chi.Router) {
	return func(r chi.Router) {
		handler := NewHandler(service)

		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
	}
}
