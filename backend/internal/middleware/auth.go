package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type to avoid collisions in the request context.
type contextKey string

const userIDKey contextKey = "userID"

// Auth is a middleware that validates the JWT Bearer token in the
// Authorization header. If valid, the user's ID is injected into the
// request context so downstream handlers can retrieve it without re-parsing.
//
// If the token is missing or invalid, it responds with 401 Unauthorized and
// stops the request chain — the actual handler never runs.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the Bearer token from the Authorization header.
			// Expected format: "Authorization: Bearer <token>"
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"success":false,"error":"authorization header required"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"success":false,"error":"invalid authorization format, use: Bearer <token>"}`, http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			// Parse and validate the token.
			// jwt.ParseWithClaims verifies: signature, expiry, and claims structure.
			token, err := jwt.ParseWithClaims(
				tokenString,
				&jwt.RegisteredClaims{},
				func(t *jwt.Token) (interface{}, error) {
					// Verify the signing algorithm
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(jwtSecret), nil
				},
			)

			if err != nil || !token.Valid {
				http.Error(w, `{"success":false,"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Extract the user ID from the "sub" (subject) claim.
			claims, ok := token.Claims.(*jwt.RegisteredClaims)
			if !ok || claims.Subject == "" {
				http.Error(w, `{"success":false,"error":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			// Inject the user ID into the request context.
			// Handlers retrieve it with GetUserID(r.Context()).
			ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID retrieves the authenticated user's ID from the request context.
// Returns an empty string if not set (i.e. request didn't go through Auth middleware).
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}
