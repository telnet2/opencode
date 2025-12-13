package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "hello"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("Expected message 'hello', got '%s'", result["message"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var result ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Error.Code != ErrCodeInvalidRequest {
		t.Errorf("Expected code %s, got %s", ErrCodeInvalidRequest, result.Error.Code)
	}
	if result.Error.Message != "Invalid input" {
		t.Errorf("Expected message 'Invalid input', got '%s'", result.Error.Message)
	}
}

func TestWriteErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]any{
		"field": "email",
		"reason": "invalid format",
	}

	writeErrorWithDetails(w, http.StatusUnprocessableEntity, ErrCodeInvalidRequest, "Validation failed", details)

	var result ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Error.Details["field"] != "email" {
		t.Errorf("Expected details.field 'email', got '%v'", result.Error.Details["field"])
	}
}

func TestWriteSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	writeSuccess(w)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// writeSuccess returns `true` (boolean) to match TypeScript SDK contract
	var result bool
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !result {
		t.Error("Expected true")
	}
}

func TestNotImplemented(t *testing.T) {
	w := httptest.NewRecorder()

	notImplemented(w)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", w.Code)
	}

	var result ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Error.Code != "NOT_IMPLEMENTED" {
		t.Errorf("Expected code NOT_IMPLEMENTED, got %s", result.Error.Code)
	}
}
