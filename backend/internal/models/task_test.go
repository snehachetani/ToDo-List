package models_test

// WHY tests?
//   - They prove the validation logic works correctly before touching a DB.
//   - They run in CI/CD on every push — catch regressions automatically.
//   - Go's testing package is built-in, no extra dependencies needed.
//
// HOW to run:
//   cd backend && go test ./internal/models/... -v

import (
	"testing"

	"github.com/snehachetani/todo-list/internal/models"
)

// ── CreateTaskInput validation ─────────────────────────────────────────────

func TestCreateTaskInput_Validate_EmptyTitle(t *testing.T) {
	input := &models.CreateTaskInput{Title: ""}
	if err := input.Validate(); err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestCreateTaskInput_Validate_TitleTooLong(t *testing.T) {
	// 201 characters — just over the limit
	longTitle := string(make([]byte, 201))
	input := &models.CreateTaskInput{Title: longTitle}
	if err := input.Validate(); err == nil {
		t.Fatal("expected error for title > 200 chars, got nil")
	}
}

func TestCreateTaskInput_Validate_InvalidPriority(t *testing.T) {
	input := &models.CreateTaskInput{Title: "Test task", Priority: "urgent"}
	if err := input.Validate(); err == nil {
		t.Fatal("expected error for invalid priority 'urgent', got nil")
	}
}

func TestCreateTaskInput_Validate_DefaultPriority(t *testing.T) {
	// Priority should default to 'medium' when not provided
	input := &models.CreateTaskInput{Title: "Test task"}
	if err := input.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if input.Priority != models.PriorityMedium {
		t.Errorf("expected default priority 'medium', got %q", input.Priority)
	}
}

func TestCreateTaskInput_Validate_ValidInput(t *testing.T) {
	testCases := []struct {
		name  string
		input models.CreateTaskInput
	}{
		{"low priority",    models.CreateTaskInput{Title: "Buy groceries", Priority: "low"}},
		{"medium priority", models.CreateTaskInput{Title: "Read a book",   Priority: "medium"}},
		{"high priority",   models.CreateTaskInput{Title: "Pay rent",      Priority: "high"}},
		{"with description", models.CreateTaskInput{Title: "Study Go", Description: "Chapter 5", Priority: "high"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			if err := input.Validate(); err != nil {
				t.Errorf("unexpected validation error for %q: %v", tc.name, err)
			}
		})
	}
}

func TestCreateTaskInput_Validate_TrimTitle(t *testing.T) {
	input := &models.CreateTaskInput{Title: "  spaces around  ", Priority: "medium"}
	if err := input.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Title != "spaces around" {
		t.Errorf("expected trimmed title, got %q", input.Title)
	}
}

// ── UpdateTaskInput validation ─────────────────────────────────────────────

func TestUpdateTaskInput_Validate_EmptyTitle(t *testing.T) {
	empty := ""
	input := &models.UpdateTaskInput{Title: &empty}
	if err := input.Validate(); err == nil {
		t.Fatal("expected error for empty title update, got nil")
	}
}

func TestUpdateTaskInput_Validate_NilFields_OK(t *testing.T) {
	// A completely nil update (no fields set) should be valid
	input := &models.UpdateTaskInput{}
	if err := input.Validate(); err != nil {
		t.Fatalf("unexpected error for nil update: %v", err)
	}
}

func TestUpdateTaskInput_Validate_InvalidPriority(t *testing.T) {
	bad := models.Priority("critical")
	input := &models.UpdateTaskInput{Priority: &bad}
	if err := input.Validate(); err == nil {
		t.Fatal("expected error for invalid priority in update, got nil")
	}
}
