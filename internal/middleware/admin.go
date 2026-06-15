// Package middleware provides HTTP middleware for the voter portal.
package middleware

import (
	"net/http"
	"strings"
)

// RequireAdminKey returns a middleware that enforces a static admin API key.
// The key must be sent as "Authorization: Bearer <key>".
// If ADMIN_API_KEY is empty the server refuses to start (enforced in main.go).
func RequireAdminKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(auth, "Bearer ")
			if !ok || token != key {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"admin key required"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
