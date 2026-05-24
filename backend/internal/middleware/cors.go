package middleware

import (
	"net/http"
	"strings"
)

// CORS adds Cross-Origin Resource Sharing headers to every response.
//
// WHY CORS?
// Browsers refuse to make API calls from one origin (e.g. http://myapp.com)
// to another (e.g. http://api.myapp.com) unless the server explicitly allows it.
// CORS headers tell the browser: "yes, requests from these origins are welcome."
//
// allowedOrigins is a slice of origins like ["http://localhost:8080"].
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	// Build a fast lookup set from the allowed origins slice.
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.ToLower(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if _, ok := allowed[strings.ToLower(origin)]; ok || len(allowedOrigins) == 0 {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400") // cache preflight for 24h
			}

			// Handle preflight OPTIONS request — browsers send this before the real request.
			// We respond with 204 No Content and the CORS headers, and do NOT call next.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
