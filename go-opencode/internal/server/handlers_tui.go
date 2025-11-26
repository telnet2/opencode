package server

import (
	"encoding/json"
	"net/http"
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

// Client tool handlers

// registerClientTool handles POST /client-tools/register
func (s *Server) registerClientTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	writeSuccess(w)
}

// unregisterClientTool handles DELETE /client-tools/unregister
func (s *Server) unregisterClientTool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}
	writeSuccess(w)
}

// executeClientTool handles POST /client-tools/execute
func (s *Server) executeClientTool(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}

// submitClientToolResult handles POST /client-tools/result
func (s *Server) submitClientToolResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CallID string `json:"callID"`
		Result string `json:"result"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
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
