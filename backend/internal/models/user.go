// Package models contains the data structures and database access functions
// for all domain entities (User, Task).
//
// WHY put DB queries in models, not handlers?
//   - Handlers should only deal with HTTP (parse request, call model, write response).
//   - If you later switch from SQLite to PostgreSQL, only models.go changes.
//   - Models can be unit-tested without starting an HTTP server.
package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents an authenticated user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // json:"-" means this field is NEVER serialised to JSON
	CreatedAt    time.Time `json:"created_at"`
}

// ErrEmailTaken is returned when trying to register with an already-used email.
var ErrEmailTaken = errors.New("email already registered")

// ErrUserNotFound is returned when no user matches the given credentials.
var ErrUserNotFound = errors.New("user not found")

// ErrInvalidPassword is returned when a login attempt uses the wrong password.
var ErrInvalidPassword = errors.New("invalid email or password")

// CreateUser registers a new user.
//
// It:
//  1. Validates the email format (basic check)
//  2. Validates the password length
//  3. Hashes the password with bcrypt (cost=12 — secure but not too slow)
//  4. Inserts the new user row
//
// Returns ErrEmailTaken if the email is already registered.
func CreateUser(db *sql.DB, email, password string) (*User, error) {
	// ── Validation ────────────────────────────────────────────────────────────
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(email) < 5 {
		return nil, fmt.Errorf("invalid email address")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return nil, fmt.Errorf("password too long (max 128 characters)")
	}

	// ── Password hashing ──────────────────────────────────────────────────────
	// bcrypt is the industry standard for passwords:
	// it is slow by design (cost=12 means ~100ms), making brute-force impractical.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// ── Database insert ───────────────────────────────────────────────────────
	user := &User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
	}

	_, err = db.Exec(
		`INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash,
	)
	if err != nil {
		// SQLite returns a "UNIQUE constraint failed" error for duplicate emails.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	user.CreatedAt = time.Now()
	return user, nil
}

// AuthenticateUser verifies a login attempt.
// It returns the User on success, or ErrInvalidPassword on failure.
// We intentionally do NOT distinguish "email not found" from "wrong password"
// in the error — this prevents user enumeration attacks.
func AuthenticateUser(db *sql.DB, email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user User
	err := db.QueryRow(
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		// Deliberately return the same error as wrong password
		return nil, ErrInvalidPassword
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	// Compare the submitted password against the stored hash.
	// bcrypt.CompareHashAndPassword is timing-attack safe.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidPassword
	}

	return &user, nil
}

// GetUserByID fetches a user by their UUID. Used to validate JWT subjects.
func GetUserByID(db *sql.DB, id string) (*User, error) {
	var user User
	err := db.QueryRow(
		`SELECT id, email, password_hash, created_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return &user, nil
}
