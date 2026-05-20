package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/snehachetani/todo-list/internal/models"
)

// AuthHandler groups the authentication endpoints and their dependencies.
// Using a struct (instead of standalone functions) makes dependencies explicit
// and makes the handler easy to test by injecting a mock DB.
type AuthHandler struct {
	db        *sql.DB
	jwtSecret string
	jwtExpiry time.Duration
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(db *sql.DB, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

// registerRequest is the expected JSON body for POST /api/v1/auth/register.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is the expected JSON body for POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse is the JSON body returned on successful auth.
type authResponse struct {
	Token string      `json:"token"`
	User  *models.User `json:"user"`
}

// Register handles POST /api/v1/auth/register
//
// Step-by-step:
//  1. Decode the JSON body
//  2. Create the user (model validates + hashes password)
//  3. Generate a JWT
//  4. Return the token + user (without password hash)
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := models.CreateUser(h.db, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEmailTaken):
			JSONError(w, http.StatusConflict, "email already registered")
		default:
			JSONError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	token, err := h.generateJWT(user.ID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	JSONCreated(w, authResponse{Token: token, User: user})
}

// Login handles POST /api/v1/auth/login
//
// Step-by-step:
//  1. Decode the JSON body
//  2. Authenticate the user (model verifies bcrypt hash)
//  3. Generate a JWT
//  4. Return the token + user
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := models.AuthenticateUser(h.db, req.Email, req.Password)
	if err != nil {
		// Always return 401 for auth failures — never 404 (prevents user enumeration)
		JSONError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.generateJWT(user.ID)
	if err != nil {
		JSONError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	JSONSuccess(w, authResponse{Token: token, User: user})
}

// generateJWT creates a signed JWT token for the given user ID.
// The token contains:
//   - sub: the user's UUID (used as the identity in protected endpoints)
//   - iat: issued-at timestamp
//   - exp: expiry timestamp (from config.JWTExpiry)
func (h *AuthHandler) generateJWT(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(h.jwtExpiry)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
