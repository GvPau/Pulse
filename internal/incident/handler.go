package incident

import (
	"errors"
	"log"
	"net/http"
	"pulse/internal/auth"
	"pulse/internal/httpx"
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

	pp, err := httpx.ParsePageParams(r)
	if err != nil {
		var ve *httpx.ValidationError
		if errors.As(err, &ve) {
			httpx.WriteValidationError(w, ve)
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid parameters")
		return
	}

	params := ListParams{Page: pp.Page, Limit: pp.Limit}

	if raw := r.URL.Query().Get("monitor_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "monitor_id must be a valid uuid")
			return
		}
		params.MonitorID = &id
	}

	if raw := r.URL.Query().Get("status"); raw != "" {
		if raw != "active" && raw != "resolved" {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "status must be one of: active, resolved")
			return
		}
		params.Status = raw
	}

	incidents, total, err := h.service.List(r.Context(), userID, params)
	if err != nil {
		log.Printf("list incidents: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to list incidents")
		return
	}

	httpx.WriteList(w, pp, total, incidents)
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
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "incident not found")
			return
		}

		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to get incident")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, incident)
}
