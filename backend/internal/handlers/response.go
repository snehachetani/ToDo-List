// Package handlers provides HTTP request handlers for the Todo API.
//
// This file defines shared response helpers used by every handler.
//
// WHY consistent response format?
//   - Clients (frontend JS, mobile apps) can always parse the same shape.
//   - Errors always have a human-readable "error" field.
//   - Success responses always have a "data" field.
//
// Response shapes:
//
//	Success: { "success": true,  "data": <any> }
//	Error:   { "success": false, "error": "<message>" }
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// apiResponse is the envelope for all API responses.
type apiResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// JSON writes a JSON-encoded response with the given status code.
// It sets Content-Type and handles encoding errors gracefully.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// We already started writing the response so we can't change the status.
		// Log the error and move on.
		log.Printf("ERROR: encode json response: %v", err)
	}
}

// JSONSuccess writes { "success": true, "data": data } with status 200.
func JSONSuccess(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, apiResponse{Success: true, Data: data})
}

// JSONCreated writes { "success": true, "data": data } with status 201.
func JSONCreated(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, apiResponse{Success: true, Data: data})
}

// JSONError writes { "success": false, "error": message } with the given status.
func JSONError(w http.ResponseWriter, status int, message string) {
	JSON(w, status, apiResponse{Success: false, Error: message})
}

// JSONNoContent writes a 204 No Content response (for DELETE).
func JSONNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
