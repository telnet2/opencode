package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/opencode-ai/opencode/internal/clienttool"
)

// TUI control handlers for the TUI client.

// tuiAppendPrompt handles POST /tui/append-prompt
func (s *Server) tuiAppendPrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	// TUI would receive this via SSE
	writeSuccess(w)
}

// tuiExecuteCommand handles POST /tui/execute-command
func (s *Server) tuiExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	writeSuccess(w)
}

// tuiShowToast handles POST /tui/show-toast
func (s *Server) tuiShowToast(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
		Type    string `json:"type"` // "info" | "error" | "success"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	writeSuccess(w)
}

// tuiPublish handles POST /tui/publish
func (s *Server) tuiPublish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Event string `json:"event"`
		Data  any    `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	writeSuccess(w)
}

// tuiOpenHelp handles POST /tui/open-help
func (s *Server) tuiOpenHelp(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w)
}

// tuiOpenSessions handles POST /tui/open-sessions
func (s *Server) tuiOpenSessions(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w)
}

// tuiOpenThemes handles POST /tui/open-themes
func (s *Server) tuiOpenThemes(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w)
}

// tuiOpenModels handles POST /tui/open-models
func (s *Server) tuiOpenModels(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w)
}

// tuiSubmitPrompt handles POST /tui/submit-prompt
func (s *Server) tuiSubmitPrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	writeSuccess(w)
}

// tuiClearPrompt handles POST /tui/clear-prompt
func (s *Server) tuiClearPrompt(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w)
}

// tuiControlNext handles GET /tui/control/next
// Returns the next pending TUI request from the queue.
func (s *Server) tuiControlNext(w http.ResponseWriter, r *http.Request) {
	// Return empty request if nothing pending
	// In a real implementation, this would pull from a queue
	response := map[string]any{
		"path": "",
		"body": nil,
	}
	writeJSON(w, http.StatusOK, response)
}

// tuiControlResponse handles POST /tui/control/response
// Submits a response to a TUI control request.
func (s *Server) tuiControlResponse(w http.ResponseWriter, r *http.Request) {
	var req any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = nil
	}
	// In a real implementation, this would process the response
	_ = req
	writeSuccess(w)
}

// Client tool handlers

// registerClientTool handles POST /client-tools/register
func (s *Server) registerClientTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID string `json:"clientID"`
		Tools    []struct {
			ID          string         `json:"id"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.ClientID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "clientID required")
		return
	}

	// Convert to ToolDefinition slice
	tools := make([]clienttool.ToolDefinition, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = clienttool.ToolDefinition{
			ID:          t.ID,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
	}

	registered := clienttool.Register(req.ClientID, tools)
	writeJSON(w, http.StatusOK, map[string]any{
		"registered": registered,
	})
}

// unregisterClientTool handles DELETE /client-tools/unregister
func (s *Server) unregisterClientTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID string   `json:"clientID"`
		ToolIDs  []string `json:"toolIDs,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.ClientID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "clientID required")
		return
	}

	unregistered := clienttool.Unregister(req.ClientID, req.ToolIDs)
	writeJSON(w, http.StatusOK, map[string]any{
		"unregistered": unregistered,
	})
}

// executeClientTool handles POST /client-tools/execute
func (s *Server) executeClientTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolID    string         `json:"toolID"`
		RequestID string         `json:"requestID"`
		SessionID string         `json:"sessionID"`
		MessageID string         `json:"messageID"`
		CallID    string         `json:"callID"`
		Input     map[string]any `json:"input"`
		Timeout   int            `json:"timeout,omitempty"` // milliseconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	// Find the client that owns this tool
	clientID := clienttool.FindClientForTool(req.ToolID)
	if clientID == "" {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Tool not found")
		return
	}

	// Default timeout: 30 seconds
	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Millisecond
	}

	execReq := clienttool.ExecutionRequest{
		RequestID: req.RequestID,
		SessionID: req.SessionID,
		MessageID: req.MessageID,
		CallID:    req.CallID,
		Tool:      req.ToolID,
		Input:     req.Input,
	}

	result, err := clienttool.Execute(r.Context(), clientID, execReq, timeout)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// submitClientToolResult handles POST /client-tools/result
func (s *Server) submitClientToolResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID string         `json:"requestID"`
		Status    string         `json:"status"` // "success" or "error"
		Title     string         `json:"title,omitempty"`
		Output    string         `json:"output,omitempty"`
		Metadata  map[string]any `json:"metadata,omitempty"`
		Error     string         `json:"error,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.RequestID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "requestID required")
		return
	}

	resp := clienttool.ToolResponse{
		Status:   req.Status,
		Title:    req.Title,
		Output:   req.Output,
		Metadata: req.Metadata,
		Error:    req.Error,
	}

	if !clienttool.SubmitResult(req.RequestID, resp) {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Request not found or already completed")
		return
	}

	writeSuccess(w)
}

// openAPISpec handles GET /doc
func (s *Server) openAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       "OpenCode Server API",
			"version":     "1.0.0",
			"description": "REST API for OpenCode AI coding assistant",
		},
		"servers": []map[string]any{
			{"url": "http://localhost:8080", "description": "Local server"},
		},
	}
	writeJSON(w, http.StatusOK, spec)
}
