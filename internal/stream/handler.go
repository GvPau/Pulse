package stream

import (
	"fmt"
	"net/http"
	"pulse/internal/auth"
	"pulse/internal/httpx"
	"time"
)

type Handler struct {
	hub *Hub
}

func newHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := h.hub.Subscribe(userID)
	defer h.hub.Unsubscribe(userID, ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case frame := <-ch:
			w.Write(frame)
			flusher.Flush()
		case <-ticker.C:
			w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}

}
