# 📝 Tasky — Todo List Application

A production-ready, full-stack todo list app built with **Go + SQLite + JWT auth**.
Deployed on **AWS ECS with EC2** via **GitHub Actions CI/CD**.

## Features

- **Full CRUD** — create, read, update, delete tasks
- **JWT Authentication** — register, login, user-specific task lists
- **Priority Levels** — high / medium / low with visual badges
- **Due Dates** — overdue detection with visual warnings
- **Filters** — by status (active/completed) and priority
- **Structured Logging** — method, path, status, duration on every request
- **Docker** — multi-stage build, ~15MB runtime image
- **CI/CD** — GitHub Actions → ECR → ECS deploy on every push

---

## Project Structure

```
ToDo/
├── backend/
│   ├── cmd/
│   │   └── main.go                  # Entry point — wires everything together
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go            # Reads + validates env vars
│   │   ├── database/
│   │   │   ├── database.go          # SQLite connection + WAL mode
│   │   │   └── migrations.go        # Auto schema migrations
│   │   ├── models/
│   │   │   ├── task.go              # Task CRUD + validation
│   │   │   └── user.go              # User registration + bcrypt auth
│   │   ├── handlers/
│   │   │   ├── auth_handler.go      # POST /register, POST /login
│   │   │   ├── task_handler.go      # Full task CRUD endpoints
│   │   │   └── response.go          # Shared JSON response helpers
│   │   └── middleware/
│   │       ├── auth.go              # JWT Bearer token validation
│   │       ├── cors.go              # Cross-Origin Resource Sharing
│   │       ├── logging.go           # Request logging
│   │       └── ratelimit.go         # Token bucket rate limiter
│   ├── static/
│   │   ├── index.html               # Single-page app
│   │   ├── style.css                # Premium dark theme
│   │   └── app.js                   # Vanilla JS — no framework
│   ├── go.mod
│   └── go.sum
├── .env.example                     # Copy to .env and fill in values
├── Dockerfile                       # Multi-stage build
├── docker-compose.yml               # Local development
├── .github/
│   └── workflows/
│       └── ci-cd.yml                # GitHub Actions → AWS ECS
└── README.md
```

---

## Getting Started

### Prerequisites
- [Go 1.22+](https://go.dev/dl/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for containerized run)

### Option A: Run with Go directly

```bash
# 1. Go to backend directory
cd backend

# 2. Copy env template and configure
cp .env.example .env
# Edit .env — set a real JWT_SECRET (at least 32 chars)

# 3. Download dependencies
go mod tidy

# 4. Run the server
go run ./cmd/main.go

# 5. Open the app
# http://localhost:8080
```

### Option B: Run with Docker Compose (recommended)

```bash
# 1. Build and start
docker compose up --build

# 2. Open the app
# http://localhost:8080
```

---

## API Reference

All API responses follow this shape:
```json
{ "success": true,  "data": <payload> }
{ "success": false, "error": "message" }
```

### Authentication

| Method | Endpoint | Body | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/auth/register` | `{email, password}` | Register a new user |
| POST | `/api/v1/auth/login` | `{email, password}` | Login, receive JWT |

### Tasks (requires `Authorization: Bearer <token>` header)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/tasks` | List tasks (filter: `?completed=true`, `?priority=high`) |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks/:id` | Get single task |
| PUT | `/api/v1/tasks/:id` | Update task (partial — only send changed fields) |
| DELETE | `/api/v1/tasks/:id` | Delete task |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Returns `{"status":"ok"}` — used by ECS health checks |

---

## Environment Variables

Copy `.env.example` to `.env` and set:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `ENV` | `development` | `development` or `production` |
| `DB_PATH` | `./todos.db` | SQLite file path |
| `JWT_SECRET` | *(required)* | Min 32 chars — CHANGE THIS in production |
| `JWT_EXPIRY` | `24h` | Token lifetime (e.g. `1h`, `24h`, `7d`) |
| `ALLOWED_ORIGINS` | `http://localhost:8080` | CORS origins (comma-separated) |
| `RATE_LIMIT_RPS` | `100` | Max requests per second per IP |

---

## Tests

```bash
cd backend

# Run all tests
go test ./... -v

# Run with race detector (recommended)
go test -race ./... -v
```

---

**Happy organizing! 🎉**
