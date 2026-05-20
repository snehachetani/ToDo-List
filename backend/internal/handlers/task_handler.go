package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/snehachetani/todo-list/internal/middleware"
	"github.com/snehachetani/todo-list/internal/models"
)

// TaskHandler groups all task-related HTTP endpoints.
type TaskHandler struct {
	db *sql.DB
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(db *sql.DB) *TaskHandler {
	return &TaskHandler{db: db}
}

// List handles GET /api/v1/tasks
//
// Query parameters (all optional):
//   - completed=true|false  — filter by completion status
//   - priority=low|medium|high — filter by priority
//
// Only returns tasks owned by the authenticated user (from JWT context).
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	filter := models.TaskFilter{}

	// Parse optional query parameters for filtering
	if c := r.URL.Query().Get("completed"); c != "" {
		completed := strings.EqualFold(c, "true")
		filter.Completed = &completed
	}
	if p := r.URL.Query().Get("priority"); p != "" {
		priority := models.Priority(p)
		filter.Priority = &priority
	}

	tasks, err := models.GetTasksByUser(h.db, userID, filter)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "could not retrieve tasks")
		return
	}

	// Return an empty array (not null) when there are no tasks.
	// null would confuse frontend code that calls .length or .map().
	if tasks == nil {
		tasks = []*models.Task{}
	}

	JSONSuccess(w, tasks)
}

// Create handles POST /api/v1/tasks
//
// Request body (JSON):
//
//	{
//	  "title": "Buy groceries",       // required, max 200 chars
//	  "description": "Milk and eggs", // optional, max 2000 chars
//	  "priority": "high",             // optional: low|medium|high (default: medium)
//	  "due_date": "2024-12-31T00:00:00Z" // optional ISO 8601
//	}
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input models.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	task, err := models.CreateTask(h.db, userID, &input)
	if err != nil {
		// Validation errors (missing title, bad priority) → 400 Bad Request
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 201 Created — tells the client a new resource was created
	JSONCreated(w, task)
}

// Get handles GET /api/v1/tasks/{id}
// Returns a single task. Returns 404 if not found, 403 if owned by another user.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := extractID(r)

	task, err := models.GetTaskByID(h.db, taskID, userID)
	if err != nil {
		handleTaskError(w, err)
		return
	}

	JSONSuccess(w, task)
}

// Update handles PUT /api/v1/tasks/{id}
//
// Supports partial updates — only fields present in the JSON body are changed.
// Example: send just {"completed": true} to mark a task done without touching title.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := extractID(r)

	var input models.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	task, err := models.UpdateTask(h.db, taskID, userID, &input)
	if err != nil {
		if errors.Is(err, models.ErrTaskNotFound) || errors.Is(err, models.ErrNotOwner) {
			handleTaskError(w, err)
			return
		}
		JSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	JSONSuccess(w, task)
}

// Delete handles DELETE /api/v1/tasks/{id}
// Returns 204 No Content on success (standard REST convention for DELETE).
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := extractID(r)

	if err := models.DeleteTask(h.db, taskID, userID); err != nil {
		handleTaskError(w, err)
		return
	}

	JSONNoContent(w)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// extractID extracts the task ID from the URL path.
// For path /api/v1/tasks/abc-123, this returns "abc-123".
// Uses path value extraction compatible with Go 1.22's enhanced ServeMux.
func extractID(r *http.Request) string {
	return r.PathValue("id")
}

// handleTaskError maps model-layer errors to the correct HTTP status codes.
func handleTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrTaskNotFound):
		JSONError(w, http.StatusNotFound, "task not found")
	case errors.Is(err, models.ErrNotOwner):
		// Return 404 rather than 403 — this way attackers can't probe for valid IDs.
		JSONError(w, http.StatusNotFound, "task not found")
	default:
		JSONError(w, http.StatusInternalServerError, "internal server error")
	}
}
