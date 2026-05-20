// Command todo-app is the entry point for the Todo List API server.
//
// Startup sequence:
//  1. Load configuration from environment variables
//  2. Connect to SQLite database and run migrations
//  3. Build the HTTP router with all routes and middleware
//  4. Start the HTTP server
//
// WHY keep main.go thin?
// main.go's job is to wire things together — like a composer.
// Business logic lives in models/, HTTP logic in handlers/, config in config/.
// This makes each piece independently testable and replaceable.
package main

import (
	"log"
	"net/http"
	"os"

	// Load .env file into environment variables automatically.
	"github.com/joho/godotenv"

	"github.com/snehachetani/todo-list/internal/config"
	"github.com/snehachetani/todo-list/internal/database"
	"github.com/snehachetani/todo-list/internal/handlers"
	"github.com/snehachetani/todo-list/internal/middleware"
)

func main() {
	// ── Step 1: Load .env (development only) ──────────────────────────────────
	// godotenv.Load() reads .env and sets environment variables.
	// If .env doesn't exist (e.g. in production Docker container) it just logs
	// a warning and continues — real env vars are already set by ECS task def.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables directly")
	}

	// ── Step 2: Load and validate configuration ────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		// Fatal means: log the error and call os.Exit(1).
		// We WANT to fail hard at startup if config is wrong.
		// It's much better to crash now than to serve requests with broken config.
		log.Fatalf("❌ Configuration error: %v", err)
	}
	log.Printf("Starting Todo API | env=%s port=%s", cfg.Env, cfg.Port)

	// ── Step 3: Connect to database ───────────────────────────────────────────
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("❌ Database error: %v", err)
	}
	defer db.Close()

	// ── Step 4: Create handlers ────────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret, cfg.JWTExpiry)
	taskHandler := handlers.NewTaskHandler(db)

	// ── Step 5: Build the router ──────────────────────────────────────────────
	//   "GET /path"    — matches only GET requests to /path
	//   "POST /path"   — matches only POST requests to /path
	//   "/path/{id}"   — {id} is a path parameter, read with r.PathValue("id")
	mux := http.NewServeMux()

	// ── Public routes (no auth required) ──────────────────────────────────────
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// ── Protected routes (JWT required) ───────────────────────────────────────
	// The auth middleware wraps just the task routes, not auth routes.
	authMW := middleware.Auth(cfg.JWTSecret)

	mux.Handle("GET /api/v1/tasks", authMW(http.HandlerFunc(taskHandler.List)))
	mux.Handle("POST /api/v1/tasks", authMW(http.HandlerFunc(taskHandler.Create)))
	mux.Handle("GET /api/v1/tasks/{id}", authMW(http.HandlerFunc(taskHandler.Get)))
	mux.Handle("PUT /api/v1/tasks/{id}", authMW(http.HandlerFunc(taskHandler.Update)))
	mux.Handle("DELETE /api/v1/tasks/{id}", authMW(http.HandlerFunc(taskHandler.Delete)))

	// ── Health check ───────────────────────────────────────────────────────────
	// ECS/ALB uses this to determine if the container is healthy.
	// Returns 200 OK if the server is running.
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// ── Serve the frontend (static files) ─────────────────────────────────────
	// Go serves the HTML/CSS/JS from the ./static/ directory.
	// The single-page app handles all UI in the browser — no server-side rendering.
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
		log.Printf("Serving frontend from %s", staticDir)
	}

	// ── Step 6: Wrap with global middleware ────────────────────────────────────
	// Middleware order matters! Outer wrappers run FIRST on request, LAST on response.
	// Logging → CORS → RateLimit → router
	//   Logging sees every request (including CORS preflight and rate-limited ones)
	//   CORS handles preflight before rate limiter
	//   RateLimit protects all routes
	var handler http.Handler = mux
	handler = middleware.RateLimit(cfg.RateLimitRPS)(handler)
	handler = middleware.CORS(cfg.AllowedOrigins)(handler)
	handler = middleware.Logging(handler)

	// ── Step 7: Start the server ───────────────────────────────────────────────
	server := &http.Server{
		Addr:    cfg.Addr(),
		Handler: handler,
	}

	log.Printf("Server listening on http://localhost%s", cfg.Addr())
	log.Printf("API docs: http://localhost%s/api/v1/health", cfg.Addr())

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server stopped: %v", err)
	}
}
