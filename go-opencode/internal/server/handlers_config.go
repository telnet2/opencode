package server

import (
	"encoding/json"
	"net/http"
	"os"

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

// ProviderModel represents a model in models.dev format for TUI compatibility.
type ProviderModel struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ReleaseDate string            `json:"release_date"`
	Attachment  bool              `json:"attachment"`
	Reasoning   bool              `json:"reasoning"`
	Temperature bool              `json:"temperature"`
	ToolCall    bool              `json:"tool_call"`
	Cost        ModelCost         `json:"cost"`
	Limit       ModelLimit        `json:"limit"`
	Options     map[string]any    `json:"options"`
	Modalities  *ModelModalities  `json:"modalities,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// ModelCost represents model pricing.
type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty"`
}

// ModelLimit represents model limits.
type ModelLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

// ModelModalities represents model input/output modalities.
type ModelModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// ProviderInfo represents provider information in models.dev format for TUI compatibility.
type ProviderInfo struct {
	ID     string                   `json:"id"`
	Name   string                   `json:"name"`
	API    string                   `json:"api,omitempty"`
	Env    []string                 `json:"env"`
	Npm    string                   `json:"npm,omitempty"`
	Models map[string]ProviderModel `json:"models"` // Map, not array!
}

// ProvidersResponse is the response format for /config/providers.
type ProvidersResponse struct {
	Providers []ProviderInfo    `json:"providers"`
	Default   map[string]string `json:"default"`
}

// getDefaultProviders returns mock providers for TUI compatibility.
// TODO: Replace with actual provider registration from models.dev.
func getDefaultProviders() []ProviderInfo {
	return []ProviderInfo{
		{
			ID:   "anthropic",
			Name: "Anthropic",
			Env:  []string{"ANTHROPIC_API_KEY"},
			Npm:  "@ai-sdk/anthropic",
			Models: map[string]ProviderModel{
				"claude-sonnet-4-20250514": {
					ID:          "claude-sonnet-4-20250514",
					Name:        "Claude Sonnet 4",
					ReleaseDate: "2025-05-14",
					Attachment:  true,
					Reasoning:   false,
					Temperature: true,
					ToolCall:    true,
					Cost:        ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
					Limit:       ModelLimit{Context: 200000, Output: 64000},
					Options:     map[string]any{},
					Modalities:  &ModelModalities{Input: []string{"text", "image", "pdf"}, Output: []string{"text"}},
				},
				"claude-opus-4-20250514": {
					ID:          "claude-opus-4-20250514",
					Name:        "Claude Opus 4",
					ReleaseDate: "2025-05-14",
					Attachment:  true,
					Reasoning:   false,
					Temperature: true,
					ToolCall:    true,
					Cost:        ModelCost{Input: 15.0, Output: 75.0, CacheRead: 1.5, CacheWrite: 18.75},
					Limit:       ModelLimit{Context: 200000, Output: 32000},
					Options:     map[string]any{},
					Modalities:  &ModelModalities{Input: []string{"text", "image", "pdf"}, Output: []string{"text"}},
				},
				"claude-3-5-haiku-20241022": {
					ID:          "claude-3-5-haiku-20241022",
					Name:        "Claude 3.5 Haiku",
					ReleaseDate: "2024-10-22",
					Attachment:  true,
					Reasoning:   false,
					Temperature: true,
					ToolCall:    true,
					Cost:        ModelCost{Input: 0.8, Output: 4.0, CacheRead: 0.08, CacheWrite: 1.0},
					Limit:       ModelLimit{Context: 200000, Output: 8192},
					Options:     map[string]any{},
					Modalities:  &ModelModalities{Input: []string{"text", "image", "pdf"}, Output: []string{"text"}},
				},
			},
		},
		{
			ID:   "openai",
			Name: "OpenAI",
			Env:  []string{"OPENAI_API_KEY"},
			Npm:  "@ai-sdk/openai",
			Models: map[string]ProviderModel{
				"gpt-4o": {
					ID:          "gpt-4o",
					Name:        "GPT-4o",
					ReleaseDate: "2024-05-13",
					Attachment:  true,
					Reasoning:   false,
					Temperature: true,
					ToolCall:    true,
					Cost:        ModelCost{Input: 2.5, Output: 10.0},
					Limit:       ModelLimit{Context: 128000, Output: 16384},
					Options:     map[string]any{},
					Modalities:  &ModelModalities{Input: []string{"text", "image"}, Output: []string{"text"}},
				},
				"gpt-4o-mini": {
					ID:          "gpt-4o-mini",
					Name:        "GPT-4o Mini",
					ReleaseDate: "2024-07-18",
					Attachment:  true,
					Reasoning:   false,
					Temperature: true,
					ToolCall:    true,
					Cost:        ModelCost{Input: 0.15, Output: 0.6},
					Limit:       ModelLimit{Context: 128000, Output: 16384},
					Options:     map[string]any{},
					Modalities:  &ModelModalities{Input: []string{"text", "image"}, Output: []string{"text"}},
				},
			},
		},
	}
}

// listProviders handles GET /config/providers
func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	providers := getDefaultProviders()

	// Build default model map (first model for each provider)
	defaultModels := make(map[string]string)
	for _, p := range providers {
		for modelID := range p.Models {
			defaultModels[p.ID] = modelID
			break // Just get the first one
		}
	}

	response := ProvidersResponse{
		Providers: providers,
		Default:   defaultModels,
	}
	writeJSON(w, http.StatusOK, response)
}

// ProviderListResponse is the response format for /provider.
type ProviderListResponse struct {
	All       []ProviderInfo    `json:"all"`
	Default   map[string]string `json:"default"`
	Connected []string          `json:"connected"`
}

// listAllProviders handles GET /provider
func (s *Server) listAllProviders(w http.ResponseWriter, r *http.Request) {
	providers := getDefaultProviders()

	// Build default model map
	defaultModels := make(map[string]string)
	for _, p := range providers {
		for modelID := range p.Models {
			defaultModels[p.ID] = modelID
			break
		}
	}

	// Get connected providers (those with API keys configured)
	connected := []string{}
	for _, p := range providers {
		// Check if provider has API key in environment
		for _, envVar := range p.Env {
			if val := getEnvValue(envVar); val != "" {
				connected = append(connected, p.ID)
				break
			}
		}
	}

	response := ProviderListResponse{
		All:       providers,
		Default:   defaultModels,
		Connected: connected,
	}
	writeJSON(w, http.StatusOK, response)
}

// getEnvValue gets an environment variable value.
func getEnvValue(key string) string {
	return os.Getenv(key)
}

// AuthMethod represents an authentication method for a provider.
type AuthMethod struct {
	Type  string `json:"type"`  // "oauth" or "api"
	Label string `json:"label"` // Display label
}

// getAuthMethods handles GET /provider/auth
// Returns Record<string, AuthMethod[]> - map from provider ID to auth methods.
func (s *Server) getAuthMethods(w http.ResponseWriter, r *http.Request) {
	// Return available auth methods for providers
	// Format: { "providerId": [{"type": "api", "label": "..."}], ... }
	authMethods := map[string][]AuthMethod{
		"anthropic": {
			{Type: "api", Label: "Manually enter API Key"},
		},
		"openai": {
			{Type: "api", Label: "Manually enter API Key"},
		},
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

// MCPServerStatus represents the status of an MCP server for TUI.
// Status can be "connected", "disabled", or "failed".
type MCPServerStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"` // Only for failed status
}

// getMCPStatus handles GET /mcp
// Returns Record<string, MCPServerStatus> - a map from server name to status.
func (s *Server) getMCPStatus(w http.ResponseWriter, r *http.Request) {
	// Return map of serverName -> status
	statuses := make(map[string]MCPServerStatus)

	if s.mcpClient != nil {
		// Get status from actual MCP client (returns []ServerStatus)
		for _, server := range s.mcpClient.Status() {
			status := MCPServerStatus{
				Status: string(server.Status),
			}
			if server.Error != nil {
				status.Error = *server.Error
			}
			statuses[server.Name] = status
		}
	}

	writeJSON(w, http.StatusOK, statuses)
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
