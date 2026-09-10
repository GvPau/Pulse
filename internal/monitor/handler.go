package monitor

import (
	"encoding/json"
	"errors"
	"net/http"
	"pulse/internal/auth"
	"pulse/internal/httpx"
	"strconv"
	"uuid"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req Monitor
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid request body")
		return
	}

	monitor, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		var ve *httpx.ValidationError
		if errors.As(err, &ve) {
			httpx.WriteValidationError(w, ve)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to create monitor")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, monitor)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))

	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid monitor id")
		return
	}

	monitor, err := h.service.Get(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "monitor not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to get monitor")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, monitor)
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

	params := ListParams{
		Page:   pp.Page,
		Limit:  pp.Limit,
		Q:      r.URL.Query().Get("q"),
		Sort:   r.URL.Query().Get("sort"),
		Order:  r.URL.Query().Get("order"),
		Window: DefaultWindow,
	}

	if raw := r.URL.Query().Get("window"); raw != "" {
		if _, ok := windowByKey(raw); !ok {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "window must be one of: 24h, 7d, 30d, 90d")
			return
		}
		params.Window = raw
	}

	if raw := r.URL.Query().Get("active"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "active must be true or false")
			return
		}
		params.Active = &b
	}

	if params.Sort != "" {
		if _, ok := monitorSortColumns[params.Sort]; !ok {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "sort must be one of: name, created_at, interval_seconds")
			return
		}
	}
	if params.Order != "" && params.Order != "asc" && params.Order != "desc" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "order must be asc or desc")
		return
	}

	monitors, total, err := h.service.List(r.Context(), userID, params)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to list monitors")
		return
	}

	httpx.WriteList(w, pp, total, monitors)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid monitor id")
		return
	}

	var req Monitor
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid request body")
		return
	}

	if err := h.service.Update(r.Context(), userID, id, &req); err != nil {
		var ve *httpx.ValidationError
		if errors.As(err, &ve) {
			httpx.WriteValidationError(w, ve)
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "monitor not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to update monitor")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid monitor id")
		return
	}

	if err := h.service.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "monitor not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to delete monitor")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

func (h *Handler) ListChecks(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid monitor id")
		return
	}

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

	params := CheckListParams{Page: pp.Page, Limit: pp.Limit}

	if raw := r.URL.Query().Get("success"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "success must be true or false")
			return
		}
		params.Success = &b
	}

	checks, total, err := h.service.ListChecks(r.Context(), userID, id, params)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "monitor not found")
			return
		}

		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to list checks")
		return
	}

	httpx.WriteList(w, pp, total, checks)

}
