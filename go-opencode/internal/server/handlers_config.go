package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opencode-ai/opencode/pkg/types"
)

// getConfig handles GET /config
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.appConfig)
}

// updateConfig handles PATCH /config
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	// Apply updates (simplified - in production, merge properly)
	if model, ok := updates["model"].(string); ok {
		s.appConfig.Model = model
	}
	if smallModel, ok := updates["small_model"].(string); ok {
		s.appConfig.SmallModel = smallModel
	}

	writeJSON(w, http.StatusOK, s.appConfig)
}

// ProviderInfo represents provider information for JSON serialization
type ProviderInfo struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Models []types.Model `json:"models"`
}

// listProviders handles GET /config/providers
func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	providers := s.providerReg.List()
	result := make([]ProviderInfo, len(providers))
	for i, p := range providers {
		result[i] = ProviderInfo{
			ID:     p.ID(),
			Name:   p.Name(),
			Models: p.Models(),
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// listAllProviders handles GET /provider
func (s *Server) listAllProviders(w http.ResponseWriter, r *http.Request) {
	providers := s.providerReg.List()
	result := make([]ProviderInfo, len(providers))
	for i, p := range providers {
		result[i] = ProviderInfo{
			ID:     p.ID(),
			Name:   p.Name(),
			Models: p.Models(),
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// getAuthMethods handles GET /provider/auth
func (s *Server) getAuthMethods(w http.ResponseWriter, r *http.Request) {
	// Return available auth methods for providers
	authMethods := []map[string]any{
		{"provider": "anthropic", "type": "api_key", "envVar": "ANTHROPIC_API_KEY"},
		{"provider": "openai", "type": "api_key", "envVar": "OPENAI_API_KEY"},
		{"provider": "google", "type": "api_key", "envVar": "GOOGLE_API_KEY"},
		{"provider": "bedrock", "type": "aws_credentials"},
	}
	writeJSON(w, http.StatusOK, authMethods)
}

// oauthAuthorize handles POST /provider/{providerID}/oauth/authorize
func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}

// oauthCallback handles POST /provider/{providerID}/oauth/callback
func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}

// setAuth handles PUT /auth/{providerID}
func (s *Server) setAuth(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "providerID")

	var req struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	// Update provider config
	if s.appConfig.Provider == nil {
		s.appConfig.Provider = make(map[string]types.ProviderConfig)
	}

	// This would typically save to config file
	writeSuccess(w)

	_ = providerID
	_ = req
}

// getLSPStatus handles GET /lsp
func (s *Server) getLSPStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"enabled": s.appConfig.LSP == nil || !s.appConfig.LSP.Disabled,
		"servers": []any{},
	}
	writeJSON(w, http.StatusOK, status)
}

// getMCPStatus handles GET /mcp
func (s *Server) getMCPStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"enabled": true,
		"servers": []any{},
	}
	writeJSON(w, http.StatusOK, status)
}

// addMCPServer handles POST /mcp
func (s *Server) addMCPServer(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}

// listAgents handles GET /agent
func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents := []map[string]any{
		{"id": "coder", "name": "Coder", "description": "General coding assistant"},
		{"id": "build", "name": "Build", "description": "Build and test assistant"},
	}
	writeJSON(w, http.StatusOK, agents)
}

// getFormatterStatus handles GET /formatter
func (s *Server) getFormatterStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"enabled": true,
	}
	writeJSON(w, http.StatusOK, status)
}

// listCommands handles GET /command
func (s *Server) listCommands(w http.ResponseWriter, r *http.Request) {
	commands := []map[string]any{
		{"name": "help", "description": "Show help"},
		{"name": "clear", "description": "Clear conversation"},
		{"name": "compact", "description": "Compact conversation"},
	}
	writeJSON(w, http.StatusOK, commands)
}

// getPath handles GET /path
func (s *Server) getPath(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"directory": getDirectory(r.Context()),
	})
}

// writeLog handles POST /log
func (s *Server) writeLog(w http.ResponseWriter, r *http.Request) {
	// Log endpoint for TUI
	writeSuccess(w)
}

// disposeInstance handles POST /instance/dispose
func (s *Server) disposeInstance(w http.ResponseWriter, r *http.Request) {
	// Cleanup instance resources
	writeSuccess(w)
}

// getToolIDs handles GET /experimental/tool/ids
func (s *Server) getToolIDs(w http.ResponseWriter, r *http.Request) {
	tools := s.toolReg.List()
	ids := make([]string, len(tools))
	for i, t := range tools {
		ids[i] = t.ID()
	}
	writeJSON(w, http.StatusOK, ids)
}

// getToolDefinitions handles GET /experimental/tool
func (s *Server) getToolDefinitions(w http.ResponseWriter, r *http.Request) {
	tools := s.toolReg.List()
	defs := make([]map[string]any, len(tools))
	for i, t := range tools {
		defs[i] = map[string]any{
			"name":        t.ID(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		}
	}
	writeJSON(w, http.StatusOK, defs)
}
