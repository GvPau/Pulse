package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	CodeInvalidRequest = "invalid_request"
	CodeUnauthorized   = "unauthorized"
	CodeNotFound       = "not_found"
	CodeConflict       = "conflict"
	CodeInternal       = "internal_error"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type errorResponse struct {
	Error Error `json:"error"`
}

type ValidationError struct {
	Message string
	Fields  map[string]string
}

type PageParams struct {
	Page  int
	Limit int
}

type PageMeta struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}

type ListResponse struct {
	Data       any      `json:"data"`
	Pagination PageMeta `json:"pagination"`
}

func (e *ValidationError) Error() string { return e.Message }

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encoding error", http.StatusInternalServerError)
	}
}

func WriteErrorDetails(w http.ResponseWriter, status int, code, message string, details any) {
	WriteJSON(w, status, errorResponse{Error: Error{Code: code, Message: message, Details: details}})
}

func WriteValidationError(w http.ResponseWriter, ve *ValidationError) {
	WriteErrorDetails(w, http.StatusBadRequest, CodeInvalidRequest, ve.Message, ve.Fields)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteErrorDetails(w, status, code, message, nil)
}

func ParsePageParams(r *http.Request) (*PageParams, error) {
	page, limit := 1, 20
	fields := map[string]string{}

	if raw := r.URL.Query().Get("page"); raw != "" {
		if v, err := strconv.Atoi(raw); err != nil || v < 1 {
			fields["page"] = "must be a number >= 1"
		} else {
			page = v
		}
	}

	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err != nil || v < 1 || v > 100 {
			fields["limit"] = "must be a number between 1 and 100"
		} else {
			limit = v
		}
	}

	if len(fields) > 0 {
		return nil, &ValidationError{Message: "invalid pagination parameters", Fields: fields}
	}
	return &PageParams{Page: page, Limit: limit}, nil
}

func WriteList(w http.ResponseWriter, p *PageParams, total int, data any) {
	hasMore := p.Page*p.Limit < total
	WriteJSON(w, http.StatusOK, ListResponse{
		Data: data,
		Pagination: PageMeta{
			Page:    p.Page,
			Limit:   p.Limit,
			Total:   total,
			HasMore: hasMore,
		},
	})
}
