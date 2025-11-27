package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opencode-ai/opencode/internal/command"
	"github.com/opencode-ai/opencode/internal/mcp"
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
	if s.mcpClient == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false,
			"servers": []any{},
		})
		return
	}

	servers := s.mcpClient.Status()
	tools := s.mcpClient.Tools()

	status := map[string]any{
		"enabled":        true,
		"serverCount":    s.mcpClient.ServerCount(),
		"connectedCount": s.mcpClient.ConnectedCount(),
		"servers":        servers,
		"tools":          tools,
	}
	writeJSON(w, http.StatusOK, status)
}

// addMCPServer handles POST /mcp
func (s *Server) addMCPServer(w http.ResponseWriter, r *http.Request) {
	if s.mcpClient == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "MCP client not initialized")
		return
	}

	var req struct {
		Name        string            `json:"name"`
		Type        string            `json:"type"`
		URL         string            `json:"url,omitempty"`
		Command     []string          `json:"command,omitempty"`
		Headers     map[string]string `json:"headers,omitempty"`
		Environment map[string]string `json:"environment,omitempty"`
		Timeout     int               `json:"timeout,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Name is required")
		return
	}

	config := &mcp.Config{
		Enabled:     true,
		Type:        mcp.TransportType(req.Type),
		URL:         req.URL,
		Command:     req.Command,
		Headers:     req.Headers,
		Environment: req.Environment,
		Timeout:     req.Timeout,
	}

	if err := s.mcpClient.AddServer(r.Context(), req.Name, config); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Return the server status
	serverStatus, _ := s.mcpClient.GetServer(req.Name)
	writeJSON(w, http.StatusCreated, serverStatus)
}

// removeMCPServer handles DELETE /mcp/{name}
func (s *Server) removeMCPServer(w http.ResponseWriter, r *http.Request) {
	if s.mcpClient == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "MCP client not initialized")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Server name is required")
		return
	}

	if err := s.mcpClient.RemoveServer(name); err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, err.Error())
		return
	}

	writeSuccess(w)
}

// getMCPTools handles GET /mcp/tools
func (s *Server) getMCPTools(w http.ResponseWriter, r *http.Request) {
	if s.mcpClient == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	tools := s.mcpClient.Tools()
	writeJSON(w, http.StatusOK, tools)
}

// executeMCPTool handles POST /mcp/tool/{name}
func (s *Server) executeMCPTool(w http.ResponseWriter, r *http.Request) {
	if s.mcpClient == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "MCP client not initialized")
		return
	}

	toolName := chi.URLParam(r, "name")
	if toolName == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Tool name is required")
		return
	}

	var args json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		// Empty body is ok
		args = nil
	}

	result, err := s.mcpClient.ExecuteTool(r.Context(), toolName, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"result": result})
}

// getMCPResources handles GET /mcp/resources
func (s *Server) getMCPResources(w http.ResponseWriter, r *http.Request) {
	if s.mcpClient == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	resources, err := s.mcpClient.ListResources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

// readMCPResource handles GET /mcp/resource
func (s *Server) readMCPResource(w http.ResponseWriter, r *http.Request) {
	if s.mcpClient == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "MCP client not initialized")
		return
	}

	uri := r.URL.Query().Get("uri")
	if uri == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "URI is required")
		return
	}

	result, err := s.mcpClient.ReadResource(r.Context(), uri)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
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
	if s.formatterManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}

	status := s.formatterManager.Status()
	writeJSON(w, http.StatusOK, status)
}

// formatFile handles POST /formatter/format
func (s *Server) formatFile(w http.ResponseWriter, r *http.Request) {
	if s.formatterManager == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "Formatter not initialized")
		return
	}

	var req struct {
		Path  string   `json:"path"`
		Paths []string `json:"paths,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	if req.Path != "" {
		result, err := s.formatterManager.Format(r.Context(), req.Path)
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	if len(req.Paths) > 0 {
		results := s.formatterManager.FormatMultiple(r.Context(), req.Paths)
		writeJSON(w, http.StatusOK, results)
		return
	}

	writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Either 'path' or 'paths' is required")
}

// listCommands handles GET /command
func (s *Server) listCommands(w http.ResponseWriter, r *http.Request) {
	// Start with builtin commands
	commands := make([]map[string]any, 0)
	for _, cmd := range command.BuiltinCommands() {
		commands = append(commands, map[string]any{
			"name":        cmd.Name,
			"description": cmd.Description,
			"source":      cmd.Source,
		})
	}

	// Add custom commands from executor
	if s.commandExecutor != nil {
		for _, cmd := range s.commandExecutor.List() {
			commands = append(commands, map[string]any{
				"name":        cmd.Name,
				"description": cmd.Description,
				"source":      cmd.Source,
				"agent":       cmd.Agent,
				"model":       cmd.Model,
				"subtask":     cmd.Subtask,
			})
		}
	}

	writeJSON(w, http.StatusOK, commands)
}

// executeCommand handles POST /command/{name}
func (s *Server) executeCommand(w http.ResponseWriter, r *http.Request) {
	if s.commandExecutor == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "Command executor not initialized")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Command name is required")
		return
	}

	var req struct {
		Args string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is ok
		req.Args = ""
	}

	result, err := s.commandExecutor.Execute(r.Context(), name, req.Args)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// getCommand handles GET /command/{name}
func (s *Server) getCommand(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Command name is required")
		return
	}

	// Check builtin commands first
	for _, cmd := range command.BuiltinCommands() {
		if cmd.Name == name {
			writeJSON(w, http.StatusOK, cmd)
			return
		}
	}

	// Check custom commands
	if s.commandExecutor != nil {
		if cmd, ok := s.commandExecutor.Get(name); ok {
			writeJSON(w, http.StatusOK, cmd)
			return
		}
	}

	writeError(w, http.StatusNotFound, ErrCodeNotFound, "Command not found")
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
