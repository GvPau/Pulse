package auth

import (
	"context"
	"net/http"
	"strings"
	"uuid"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}
		tokenString, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := ParseToken(tokenString)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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
