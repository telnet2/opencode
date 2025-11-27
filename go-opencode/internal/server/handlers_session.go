package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/pkg/types"
)

// CreateSessionRequest represents the request body for creating a session.
type CreateSessionRequest struct {
	Directory string `json:"directory"`
	Title     string `json:"title,omitempty"`
}

// listSessions handles GET /session
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	// Only use explicitly provided directory query parameter
	// If not provided, list all sessions (directory = "")
	directory := r.URL.Query().Get("directory")

	sessions, err := s.sessionService.List(r.Context(), directory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Ensure we return an empty array [] instead of null
	if sessions == nil {
		sessions = []*types.Session{}
	}

	writeJSON(w, http.StatusOK, sessions)
}

// createSession handles POST /session
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	directory := req.Directory
	if directory == "" {
		directory = getDirectory(r.Context())
	}

	if directory == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "directory is required")
		return
	}

	session, err := s.sessionService.Create(r.Context(), directory, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Publish event (SDK compatible: uses "info" field)
	event.Publish(event.Event{
		Type: event.SessionCreated,
		Data: event.SessionCreatedData{Info: session},
	})

	writeJSON(w, http.StatusOK, session)
}

// getSession handles GET /session/{sessionID}
func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	session, err := s.sessionService.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// updateSession handles PATCH /session/{sessionID}
func (s *Server) updateSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	session, err := s.sessionService.Update(r.Context(), sessionID, updates)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Publish event (SDK compatible: uses "info" field)
	event.Publish(event.Event{
		Type: event.SessionUpdated,
		Data: event.SessionUpdatedData{Info: session},
	})

	writeJSON(w, http.StatusOK, session)
}

// deleteSession handles DELETE /session/{sessionID}
func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	// Get session before deletion for the event (SDK expects full session info)
	session, _ := s.sessionService.Get(r.Context(), sessionID)

	if err := s.sessionService.Delete(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Publish event (SDK compatible: uses "info" field with full session)
	event.Publish(event.Event{
		Type: event.SessionDeleted,
		Data: event.SessionDeletedData{Info: session},
	})

	writeSuccess(w)
}

// getSessionStatus handles GET /session/status
func (s *Server) getSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "sessionID required")
		return
	}

	session, err := s.sessionService.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Session not found")
		return
	}

	status := map[string]any{
		"sessionID": session.ID,
		"title":     session.Title,
		"status":    "idle", // TODO: track actual status
	}

	writeJSON(w, http.StatusOK, status)
}

// getChildren handles GET /session/{sessionID}/children
func (s *Server) getChildren(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	children, err := s.sessionService.GetChildren(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, children)
}

// ForkSessionRequest represents the request body for forking a session.
type ForkSessionRequest struct {
	MessageID string `json:"messageID"`
}

// forkSession handles POST /session/{sessionID}/fork
func (s *Server) forkSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var req ForkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	newSession, err := s.sessionService.Fork(r.Context(), sessionID, req.MessageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Publish event (SDK compatible: uses "info" field)
	event.Publish(event.Event{
		Type: event.SessionCreated,
		Data: event.SessionCreatedData{Info: newSession},
	})

	writeJSON(w, http.StatusOK, newSession)
}

// abortSession handles POST /session/{sessionID}/abort
func (s *Server) abortSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if err := s.sessionService.Abort(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeSuccess(w)
}

// shareSession handles POST /session/{sessionID}/share
func (s *Server) shareSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	shareURL, err := s.sessionService.Share(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": shareURL})
}

// unshareSession handles DELETE /session/{sessionID}/share
func (s *Server) unshareSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if err := s.sessionService.Unshare(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeSuccess(w)
}

// summarizeSession handles POST /session/{sessionID}/summarize
func (s *Server) summarizeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	summary, err := s.sessionService.Summarize(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// initSession handles POST /session/{sessionID}/init
func (s *Server) initSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	session, err := s.sessionService.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Session not found")
		return
	}

	// Return session info for initialization
	writeJSON(w, http.StatusOK, session)
}

// getDiff handles GET /session/{sessionID}/diff
func (s *Server) getDiff(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	diffs, err := s.sessionService.GetDiffs(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, diffs)
}

// getTodo handles GET /session/{sessionID}/todo
func (s *Server) getTodo(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	todos, err := s.sessionService.GetTodos(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, todos)
}

// RevertSessionRequest represents the request body for reverting a session.
type RevertSessionRequest struct {
	MessageID string  `json:"messageID"`
	PartID    *string `json:"partID,omitempty"`
}

// revertSession handles POST /session/{sessionID}/revert
func (s *Server) revertSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var req RevertSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if err := s.sessionService.Revert(r.Context(), sessionID, req.MessageID, req.PartID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeSuccess(w)
}

// unrevertSession handles POST /session/{sessionID}/unrevert
func (s *Server) unrevertSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	if err := s.sessionService.Unrevert(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeSuccess(w)
}

// SendCommandRequest represents the request body for sending a command.
type SendCommandRequest struct {
	Command string `json:"command"`
}

// sendCommand handles POST /session/{sessionID}/command
func (s *Server) sendCommand(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var req SendCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	result, err := s.sessionService.ExecuteCommand(r.Context(), sessionID, req.Command)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// RunShellRequest represents the request body for running a shell command.
type RunShellRequest struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// runShell handles POST /session/{sessionID}/shell
func (s *Server) runShell(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var req RunShellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	result, err := s.sessionService.RunShell(r.Context(), sessionID, req.Command, req.Timeout)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// PermissionResponse represents the response body for permission.
type PermissionResponse struct {
	Granted bool `json:"granted"`
}

// respondPermission handles POST /session/{sessionID}/permissions/{permissionID}
func (s *Server) respondPermission(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	permissionID := chi.URLParam(r, "permissionID")

	var req PermissionResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if err := s.sessionService.RespondPermission(r.Context(), sessionID, permissionID, req.Granted); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Convert granted bool to SDK response format
	response := "reject"
	if req.Granted {
		response = "once"
	}

	// Publish event (SDK compatible: uses PermissionReplied)
	event.Publish(event.Event{
		Type: event.PermissionReplied,
		Data: event.PermissionRepliedData{
			PermissionID: permissionID,
			SessionID:    sessionID,
			Response:     response,
		},
	})

	writeSuccess(w)
}

// generateID generates a new ULID.
func generateID() string {
	return ulid.Make().String()
}

// nowMillis returns current time in milliseconds.
func nowMillis() int64 {
	return time.Now().UnixMilli()
}
