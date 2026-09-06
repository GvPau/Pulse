package auth

import (
	"context"
	"net/http"
	"pulse/internal/httpx"
	"strings"
	"uuid"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "Authorization header is missing")
			return
		}
		tokenString, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "Authorization header must use the Bearer scheme")
			return
		}
		claims, err := ParseToken(tokenString)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey).(uuid.UUID)
	return id
}
