# ═══════════════════════════════════════════════════════════════════════════
# Dockerfile — Multi-stage build
#
# WHY multi-stage?
#   Stage 1 (builder): has the Go compiler installed (~700MB image)
#   Stage 2 (runtime): only has the compiled binary (~15MB image)
#
# This makes the image:
#   - Much smaller → faster pull times when ECS starts a new task
#   - More secure   → no compiler, no source code, no build tools in production
# ═══════════════════════════════════════════════════════════════════════════

# ── Stage 1: Build ──────────────────────────────────────────────────────────
# Use the official Go image as the build environment.
# We pin to a specific version for reproducible builds.
FROM golang:1.22-alpine AS builder

# Install git (needed by some go modules during download) and ca-certificates
# (needed for HTTPS calls in build scripts).
RUN apk add --no-cache git ca-certificates

# Create a non-root user for the runtime stage (security best practice).
RUN adduser -D -g '' appuser

# Set the working directory inside the container.
WORKDIR /build

# Copy only the dependency files first — this layer is cached by Docker.
# If go.mod/go.sum haven't changed, Docker reuses this layer and skips
# the expensive `go mod download` step, making subsequent builds much faster.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the source code.
COPY . .

# Build the binary.
# CGO_ENABLED=0 — pure Go, no C dependencies (important for modernc.org/sqlite)
# GOOS=linux    — compile for Linux even if building on Windows/Mac
# -ldflags "-s -w" — strip debug info for a smaller binary
# -o todo-app   — output file name
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o todo-app \
    ./cmd/main.go

# ── Stage 2: Runtime ────────────────────────────────────────────────────────
# Start from scratch (empty) or alpine for minimal attack surface.
FROM alpine:3.19

# ca-certificates is needed for HTTPS calls from inside the container.
RUN apk add --no-cache ca-certificates tzdata

# Copy the non-root user from the builder stage.
COPY --from=builder /etc/passwd /etc/passwd

# Copy the compiled binary from the builder stage.
COPY --from=builder /build/todo-app /usr/local/bin/todo-app

# Copy the static frontend files.
COPY --from=builder /build/static ./static

# Run as the non-root user — never run as root in production.
USER appuser

# The port the app listens on. Must match PORT env var.
EXPOSE 8080

# Health check — ECS uses this to determine if the container is healthy.
# --interval: check every 30s
# --timeout:  fail if no response in 5s
# --retries:  mark unhealthy after 3 consecutive failures
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Start the application.
CMD ["/usr/local/bin/todo-app"]
