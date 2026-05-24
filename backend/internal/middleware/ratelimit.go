package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimit implements a per-IP token bucket rate limiter.
//
// WHY rate limiting?
//   - Prevents a single client from hammering your API (DDoS, brute force).
//   - On AWS, excessive traffic costs real money — a rate limiter caps that risk.
//   - Protects bcrypt login endpoints (bcrypt is intentionally slow per request).
//
// HOW the token bucket algorithm works:
//   - Each IP gets a "bucket" with a maximum capacity of `rps` tokens.
//   - Tokens refill at `rps` per second.
//   - Each request costs 1 token. If the bucket is empty, the request is rejected.
func RateLimit(rps int) func(http.Handler) http.Handler {
	type bucket struct {
		tokens   float64
		lastTime time.Time
		mu       sync.Mutex
	}

	var (
		clients sync.Map // map[string]*bucket — one bucket per IP
	)

	capacity := float64(rps)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			// Load or create the bucket for this IP.
			v, _ := clients.LoadOrStore(ip, &bucket{
				tokens:   capacity,
				lastTime: time.Now(),
			})
			b := v.(*bucket)

			b.mu.Lock()
			// Refill tokens based on elapsed time since the last request.
			now := time.Now()
			elapsed := now.Sub(b.lastTime).Seconds()
			b.tokens = min(capacity, b.tokens+elapsed*float64(rps))
			b.lastTime = now

			if b.tokens < 1 {
				b.mu.Unlock()
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"success":false,"error":"rate limit exceeded, please slow down"}`, http.StatusTooManyRequests)
				return
			}
			b.tokens--
			b.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the real client IP, checking common reverse proxy headers.
func clientIP(r *http.Request) string {
	// X-Forwarded-For is set by load balancers (AWS ALB, nginx)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// X-Real-IP is set by nginx
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
