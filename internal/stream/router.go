package stream

import (
	"net/http"
	"pulse/internal/auth"
	"pulse/internal/httpx"
	"strings"
	"uuid"

	"github.com/go-chi/chi/v5"
)

func Router(hub *Hub) func(c chi.Router) {
	return func(r chi.Router) {
		r.Use(requireUser)
		handler := newHandler(hub)
		r.Get("/stream", handler.Stream)
	}
}

// requireUser authenticates via a Bearer Authorization header (curl/Postman)
// or a `token` query parameter (browser EventSournce, which cannot send headers)
func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			userID uuid.UUID
			err    error
		)

		if header := r.Header.Get("Authorization"); header != "" {
			tokenString, ok := strings.CutPrefix(header, "Bearer ")
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "Authorization header must use the Bearer scheme")
				return
			}
			userID, err = auth.UserIDFromToken(tokenString)
		} else if token := r.URL.Query().Get("token"); token != "" {
			userID, err = auth.UserIDFromToken(token)
		} else {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing credentials: Authorization header or token query paraeter")
			return
		}

		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid or expired token")
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithUserID(r.Context(), userID)))
	})
}
