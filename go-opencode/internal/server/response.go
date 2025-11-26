package server

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Error codes
const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeProviderError    = "PROVIDER_ERROR"
	ErrCodeRateLimited      = "RATE_LIMITED"
	ErrCodeInternalError    = "INTERNAL_ERROR"
)

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// writeErrorWithDetails writes an error response with details.
func writeErrorWithDetails(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// writeSuccess writes a success response.
func writeSuccess(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// notImplemented writes a not implemented response.
func notImplemented(w http.ResponseWriter) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "This endpoint is not yet implemented")
}
