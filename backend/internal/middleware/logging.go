// Package middleware provides HTTP middleware functions.
//
// Middleware is a function that wraps an http.Handler to add behaviour
// before and/or after the actual handler runs. They form a "chain":
//
//	request → logging → cors → rateLimit → auth → actualHandler → response
//
// Each middleware can: read the request, short-circuit with an error response,
// modify the request context, then call the next handler.
package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code
// so the logging middleware can record it after the handler runs.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging logs every incoming HTTP request with method, path, status, and duration.
//
// WHY structured logging?
//   - In production you need to know: what endpoints are slow? which are erroring?
//   - Logs are searchable in AWS CloudWatch or any log aggregation service.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture the status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler in the chain
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf(
			"%-6s %-40s %d %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
		)
	})
}
