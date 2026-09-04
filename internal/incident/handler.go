package incident

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"pulse/internal/auth"
	"uuid"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	// Parse optional monitor_id filter
	var monitorID *uuid.UUID
	if raw := r.URL.Query().Get("monitor_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			http.Error(w, "invalid monitor_id", http.StatusBadRequest)
			return
		}
		monitorID = &id
	}

	incidents, err := h.service.List(r.Context(), userID, monitorID)
	if err != nil {
		log.Printf("list incidents: %v", err)
		http.Error(w, "failed to list incidents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incidents)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid incident id", http.StatusBadRequest)
		return
	}

	incident, err := h.service.Get(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "incident not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incident)
}
