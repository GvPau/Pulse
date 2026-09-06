package user

import (
	"encoding/json"
	"log"
	"net/http"
	"pulse/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "Invalid request payload")
		return
	}

	if req.Email == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "Email and password are required")
		return
	}

	token, err := h.service.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		log.Printf("register error: %v", err) // ← esto te dice QUÉ pasó en la consola
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "Failed to register user")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "Invalid request payload")
		return
	}

	if req.Email == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "Email and password are required")
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "Invalid email or password")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tokenResponse{Token: token})
}
