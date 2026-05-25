package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Priority levels for tasks.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Recurring frequency for tasks.
type Recurring string

const (
	RecurringNone    Recurring = "none"
	RecurringDaily   Recurring = "daily"
	RecurringWeekly  Recurring = "weekly"
	RecurringMonthly Recurring = "monthly"
)

// Task represents a single to-do item owned by a user.
type Task struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Completed           bool       `json:"completed"`
	Priority            Priority   `json:"priority"`
	DueDate             *time.Time `json:"due_date,omitempty"`
	Recurring           Recurring  `json:"recurring"`
	TimerDurationMinutes *int      `json:"timer_duration_minutes,omitempty"` // nil = no timer set
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// CreateTaskInput holds the validated fields for creating a new task.
type CreateTaskInput struct {
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Priority             Priority   `json:"priority"`
	DueDate              *time.Time `json:"due_date,omitempty"`
	Recurring            Recurring  `json:"recurring"`
	TimerDurationMinutes *int       `json:"timer_duration_minutes,omitempty"`
}

// UpdateTaskInput holds the fields that can be updated on an existing task.
// All fields are pointers so that we support partial updates (PATCH semantics):
// a nil pointer means "don't change this field".
type UpdateTaskInput struct {
	Title                *string    `json:"title,omitempty"`
	Description          *string    `json:"description,omitempty"`
	Completed            *bool      `json:"completed,omitempty"`
	Priority             *Priority  `json:"priority,omitempty"`
	DueDate              *time.Time `json:"due_date,omitempty"`
	Recurring            *Recurring `json:"recurring,omitempty"`
	TimerDurationMinutes *int       `json:"timer_duration_minutes,omitempty"`
}

// TaskFilter holds query parameters for listing tasks.
type TaskFilter struct {
	Completed *bool
	Priority  *Priority
}

// Sentinel errors for task operations.
var (
	ErrTaskNotFound = errors.New("task not found")
	ErrNotOwner     = errors.New("access denied: task belongs to another user")
)

// ── Validation ────────────────────────────────────────────────────────────────

func (in *CreateTaskInput) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(in.Title) > 200 {
		return fmt.Errorf("title must be 200 characters or less")
	}
	if len(in.Description) > 2000 {
		return fmt.Errorf("description must be 2000 characters or less")
	}
	if in.Priority == "" {
		in.Priority = PriorityMedium
	}
	if !isValidPriority(in.Priority) {
		return fmt.Errorf("priority must be 'low', 'medium', or 'high'")
	}
	if in.Recurring == "" {
		in.Recurring = RecurringNone
	}
	if !isValidRecurring(in.Recurring) {
		return fmt.Errorf("recurring must be 'none', 'daily', 'weekly', or 'monthly'")
	}
	if in.TimerDurationMinutes != nil && *in.TimerDurationMinutes <= 0 {
		return fmt.Errorf("timer duration must be greater than 0 minutes")
	}
	if in.TimerDurationMinutes != nil && *in.TimerDurationMinutes > 1440 {
		return fmt.Errorf("timer duration cannot exceed 24 hours (1440 minutes)")
	}
	return nil
}

func (in *UpdateTaskInput) Validate() error {
	if in.Title != nil {
		*in.Title = strings.TrimSpace(*in.Title)
		if *in.Title == "" {
			return fmt.Errorf("title cannot be empty")
		}
		if len(*in.Title) > 200 {
			return fmt.Errorf("title must be 200 characters or less")
		}
	}
	if in.Description != nil && len(*in.Description) > 2000 {
		return fmt.Errorf("description must be 2000 characters or less")
	}
	if in.Priority != nil && !isValidPriority(*in.Priority) {
		return fmt.Errorf("priority must be 'low', 'medium', or 'high'")
	}
	if in.Recurring != nil && !isValidRecurring(*in.Recurring) {
		return fmt.Errorf("recurring must be 'none', 'daily', 'weekly', or 'monthly'")
	}
	if in.TimerDurationMinutes != nil && *in.TimerDurationMinutes <= 0 {
		return fmt.Errorf("timer duration must be greater than 0 minutes")
	}
	return nil
}

func isValidPriority(p Priority) bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

func isValidRecurring(r Recurring) bool {
	return r == RecurringNone || r == RecurringDaily || r == RecurringWeekly || r == RecurringMonthly
}

// ── Database operations ───────────────────────────────────────────────────────

func CreateTask(db *sql.DB, userID string, in *CreateTaskInput) (*Task, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	task := &Task{
		ID:                  uuid.NewString(),
		UserID:              userID,
		Title:               in.Title,
		Description:         in.Description,
		Completed:           false,
		Priority:            in.Priority,
		DueDate:             in.DueDate,
		Recurring:           in.Recurring,
		TimerDurationMinutes: in.TimerDurationMinutes,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, err := db.Exec(
		`INSERT INTO tasks (id, user_id, title, description, completed, priority, due_date, recurring, timer_duration_minutes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.UserID, task.Title, task.Description,
		task.Completed, string(task.Priority), task.DueDate,
		string(task.Recurring), task.TimerDurationMinutes,
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	return task, nil
}

func GetTasksByUser(db *sql.DB, userID string, filter TaskFilter) ([]*Task, error) {
	query := `SELECT id, user_id, title, description, completed, priority, due_date,
	                 recurring, timer_duration_minutes, created_at, updated_at
	          FROM tasks WHERE user_id = ?`
	args := []any{userID}

	if filter.Completed != nil {
		query += " AND completed = ?"
		args = append(args, *filter.Completed)
	}
	if filter.Priority != nil {
		query += " AND priority = ?"
		args = append(args, string(*filter.Priority))
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func GetTaskByID(db *sql.DB, id, userID string) (*Task, error) {
	row := db.QueryRow(
		`SELECT id, user_id, title, description, completed, priority, due_date,
		        recurring, timer_duration_minutes, created_at, updated_at
		 FROM tasks WHERE id = ?`,
		id,
	)
	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if task.UserID != userID {
		return nil, ErrNotOwner
	}
	return task, nil
}

func UpdateTask(db *sql.DB, id, userID string, in *UpdateTaskInput) (*Task, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	task, err := GetTaskByID(db, id, userID)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		task.Title = *in.Title
	}
	if in.Description != nil {
		task.Description = *in.Description
	}
	if in.Completed != nil {
		task.Completed = *in.Completed
	}
	if in.Priority != nil {
		task.Priority = *in.Priority
	}
	if in.DueDate != nil {
		task.DueDate = in.DueDate
	}
	if in.Recurring != nil {
		task.Recurring = *in.Recurring
	}
	if in.TimerDurationMinutes != nil {
		task.TimerDurationMinutes = in.TimerDurationMinutes
	}

	_, err = db.Exec(
		`UPDATE tasks SET title=?, description=?, completed=?, priority=?, due_date=?,
		                  recurring=?, timer_duration_minutes=?
		 WHERE id=? AND user_id=?`,
		task.Title, task.Description, task.Completed, string(task.Priority),
		task.DueDate, string(task.Recurring), task.TimerDurationMinutes,
		id, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	task.UpdatedAt = time.Now()
	return task, nil
}

func DeleteTask(db *sql.DB, id, userID string) error {
	_, err := GetTaskByID(db, id, userID)
	if err != nil {
		return err
	}

	result, err := db.Exec(`DELETE FROM tasks WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (*Task, error) {
	var t Task
	var priority, recurring string
	err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description,
		&t.Completed, &priority, &t.DueDate,
		&recurring, &t.TimerDurationMinutes,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Priority = Priority(priority)
	t.Recurring = Recurring(recurring)
	return &t, nil
}
