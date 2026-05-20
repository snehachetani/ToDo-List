package handlers_test

// Integration tests for auth endpoints.
// We use httptest.NewRecorder() to capture responses without starting a real server.
// We use an in-memory SQLite DB so tests are fast and clean up automatically.
//
// HOW to run:
//   cd backend && go test ./internal/handlers/... -v

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/snehachetani/todo-list/internal/database"
	"github.com/snehachetani/todo-list/internal/handlers"
)

// newTestDB creates an in-memory SQLite database for testing.
// The ":memory:" path tells SQLite to keep everything in RAM.
// The database is automatically discarded when the test ends.
func newTestDB(t *testing.T) interface{ Close() error } {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRegister_Success(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	h := handlers.NewAuthHandler(db, "test-secret-32-characters-minimum!!", 24*time.Hour)

	body := `{"email":"test@example.com","password":"password123"}`
	req  := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}
	if data["token"] == "" || data["token"] == nil {
		t.Error("expected JWT token in response")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	h := handlers.NewAuthHandler(db, "test-secret-32-characters-minimum!!", 24*time.Hour)

	body := `{"email":"dup@example.com","password":"password123"}`

	// First registration — should succeed
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	h.Register(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("first register failed with %d", rr1.Code)
	}

	// Second registration with same email — should fail with 409 Conflict
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.Register(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate email, got %d", rr2.Code)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	h := handlers.NewAuthHandler(db, "test-secret-32-characters-minimum!!", 24*time.Hour)

	body := `{"email":"nobody@example.com","password":"wrongpassword"}`
	req  := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	secret := "test-secret-32-characters-minimum!!"
	h := handlers.NewAuthHandler(db, secret, 24*time.Hour)

	// Register first
	regBody := `{"email":"login@example.com","password":"mypassword123"}`
	regReq  := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	httptest.NewRecorder() // discard
	h.Register(httptest.NewRecorder(), regReq)

	// Now login
	loginBody := `{"email":"login@example.com","password":"mypassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
