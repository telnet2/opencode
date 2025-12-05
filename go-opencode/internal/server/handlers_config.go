package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/opencode-ai/opencode/internal/command"
	"github.com/opencode-ai/opencode/internal/mcp"
	"github.com/opencode-ai/opencode/pkg/types"
)

// getConfig handles GET /config
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	if s.appConfig != nil {
		s.appConfig.Keybinds = types.MergeKeybinds(types.DefaultKeybinds(), s.appConfig.Keybinds)
	}
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
// SDK compatible: uses "capabilities" with nested boolean structure to match TypeScript.
type ProviderModel struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	ReleaseDate  string             `json:"release_date"`
	Capabilities *ModelCapabilities `json:"capabilities"`
	Cost         ModelCost          `json:"cost"`
	Limit        ModelLimit         `json:"limit"`
	Options      map[string]any     `json:"options"`
	Status       string             `json:"status,omitempty"`
}

// ModelCapabilities represents model capabilities and modalities.
// SDK compatible: matches TypeScript Model.capabilities structure.
type ModelCapabilities struct {
	Temperature bool                 `json:"temperature"`
	Reasoning   bool                 `json:"reasoning"`
	Attachment  bool                 `json:"attachment"`
	ToolCall    bool                 `json:"toolcall"`
	Input       ModalityCapabilities `json:"input"`
	Output      ModalityCapabilities `json:"output"`
}

// ModalityCapabilities represents input/output modality capabilities.
// SDK compatible: matches TypeScript input/output capability structure.
type ModalityCapabilities struct {
	Text  bool `json:"text"`
	Audio bool `json:"audio"`
	Image bool `json:"image"`
	Video bool `json:"video"`
	PDF   bool `json:"pdf"`
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
					Capabilities: &ModelCapabilities{
						Temperature: true,
						Reasoning:   false,
						Attachment:  true,
						ToolCall:    true,
						Input:       ModalityCapabilities{Text: true, Audio: false, Image: true, Video: false, PDF: true},
						Output:      ModalityCapabilities{Text: true, Audio: false, Image: false, Video: false, PDF: false},
					},
					Cost:    ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
					Limit:   ModelLimit{Context: 200000, Output: 64000},
					Options: map[string]any{},
				},
				"claude-opus-4-20250514": {
					ID:          "claude-opus-4-20250514",
					Name:        "Claude Opus 4",
					ReleaseDate: "2025-05-14",
					Capabilities: &ModelCapabilities{
						Temperature: true,
						Reasoning:   false,
						Attachment:  true,
						ToolCall:    true,
						Input:       ModalityCapabilities{Text: true, Audio: false, Image: true, Video: false, PDF: true},
						Output:      ModalityCapabilities{Text: true, Audio: false, Image: false, Video: false, PDF: false},
					},
					Cost:    ModelCost{Input: 15.0, Output: 75.0, CacheRead: 1.5, CacheWrite: 18.75},
					Limit:   ModelLimit{Context: 200000, Output: 32000},
					Options: map[string]any{},
				},
				"claude-3-5-haiku-20241022": {
					ID:          "claude-3-5-haiku-20241022",
					Name:        "Claude 3.5 Haiku",
					ReleaseDate: "2024-10-22",
					Capabilities: &ModelCapabilities{
						Temperature: true,
						Reasoning:   false,
						Attachment:  true,
						ToolCall:    true,
						Input:       ModalityCapabilities{Text: true, Audio: false, Image: true, Video: false, PDF: true},
						Output:      ModalityCapabilities{Text: true, Audio: false, Image: false, Video: false, PDF: false},
					},
					Cost:    ModelCost{Input: 0.8, Output: 4.0, CacheRead: 0.08, CacheWrite: 1.0},
					Limit:   ModelLimit{Context: 200000, Output: 8192},
					Options: map[string]any{},
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
					Capabilities: &ModelCapabilities{
						Temperature: true,
						Reasoning:   false,
						Attachment:  true,
						ToolCall:    true,
						Input:       ModalityCapabilities{Text: true, Audio: false, Image: true, Video: false, PDF: false},
						Output:      ModalityCapabilities{Text: true, Audio: false, Image: false, Video: false, PDF: false},
					},
					Cost:    ModelCost{Input: 2.5, Output: 10.0},
					Limit:   ModelLimit{Context: 128000, Output: 16384},
					Options: map[string]any{},
				},
				"gpt-4o-mini": {
					ID:          "gpt-4o-mini",
					Name:        "GPT-4o Mini",
					ReleaseDate: "2024-07-18",
					Capabilities: &ModelCapabilities{
						Temperature: true,
						Reasoning:   false,
						Attachment:  true,
						ToolCall:    true,
						Input:       ModalityCapabilities{Text: true, Audio: false, Image: true, Video: false, PDF: false},
						Output:      ModalityCapabilities{Text: true, Audio: false, Image: false, Video: false, PDF: false},
					},
					Cost:    ModelCost{Input: 0.15, Output: 0.6},
					Limit:   ModelLimit{Context: 128000, Output: 16384},
					Options: map[string]any{},
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

// AgentInfo represents agent information returned by the /agent endpoint.
// SDK compatible: matches TypeScript Agent.Info structure.
type AgentInfo struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Mode        string              `json:"mode"`
	BuiltIn     bool                `json:"builtIn"`
	Prompt      string              `json:"prompt,omitempty"`
	Tools       map[string]bool     `json:"tools"`
	Options     map[string]any      `json:"options"`
	Permission  AgentPermissionInfo `json:"permission"`
	Temperature float64             `json:"temperature,omitempty"`
	TopP        float64             `json:"topP,omitempty"`
	Model       *AgentModelRef      `json:"model,omitempty"`
	Color       string              `json:"color,omitempty"`
}

// AgentPermissionInfo represents agent permission settings.
type AgentPermissionInfo struct {
	Edit        string            `json:"edit,omitempty"`
	Bash        map[string]string `json:"bash,omitempty"`
	WebFetch    string            `json:"webfetch,omitempty"`
	ExternalDir string            `json:"external_directory,omitempty"`
	DoomLoop    string            `json:"doom_loop,omitempty"`
}

// AgentModelRef references a model for an agent.
type AgentModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

// listAgents handles GET /agent
// Returns full agent objects matching TypeScript Agent.Info structure.
func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	// Start with built-in agents
	agents := getBuiltInAgents()

	// Merge with config agents
	if s.appConfig != nil && s.appConfig.Agent != nil {
		for name, cfg := range s.appConfig.Agent {
			// Find existing or create new
			var agent *AgentInfo
			for i := range agents {
				if agents[i].Name == name {
					agent = &agents[i]
					break
				}
			}

			if agent == nil {
				// New custom agent
				newAgent := AgentInfo{
					Name:    name,
					Mode:    "all",
					BuiltIn: false,
					Tools:   make(map[string]bool),
					Options: make(map[string]any),
					Permission: AgentPermissionInfo{
						Edit:        "allow",
						Bash:        map[string]string{"*": "allow"},
						WebFetch:    "allow",
						DoomLoop:    "ask",
						ExternalDir: "ask",
					},
				}
				agents = append(agents, newAgent)
				agent = &agents[len(agents)-1]
			}

			// Apply config overrides
			if cfg.Description != "" {
				agent.Description = cfg.Description
			}
			if cfg.Prompt != "" {
				agent.Prompt = cfg.Prompt
			}
			if cfg.Mode != "" {
				agent.Mode = cfg.Mode
			}
			if cfg.Temperature != nil {
				agent.Temperature = *cfg.Temperature
			}
			if cfg.TopP != nil {
				agent.TopP = *cfg.TopP
			}
			if cfg.Color != "" {
				agent.Color = cfg.Color
			}
			if cfg.Model != "" {
				// Parse model string "provider/model"
				parts := strings.SplitN(cfg.Model, "/", 2)
				if len(parts) == 2 {
					agent.Model = &AgentModelRef{
						ProviderID: parts[0],
						ModelID:    parts[1],
					}
				}
			}
			if cfg.Tools != nil {
				for k, v := range cfg.Tools {
					agent.Tools[k] = v
				}
			}
			agent.BuiltIn = false // Mark as customized
		}
	}

	writeJSON(w, http.StatusOK, agents)
}

// getBuiltInAgents returns the default built-in agents.
func getBuiltInAgents() []AgentInfo {
	defaultPermission := AgentPermissionInfo{
		Edit:        "allow",
		Bash:        map[string]string{"*": "allow"},
		WebFetch:    "allow",
		DoomLoop:    "ask",
		ExternalDir: "ask",
	}

	planPermission := AgentPermissionInfo{
		Edit: "deny",
		Bash: map[string]string{
			"cut*":             "allow",
			"diff*":            "allow",
			"du*":              "allow",
			"file *":           "allow",
			"find * -delete*":  "ask",
			"find * -exec*":    "ask",
			"find * -fprint*":  "ask",
			"find * -fls*":     "ask",
			"find * -fprintf*": "ask",
			"find * -ok*":      "ask",
			"find *":           "allow",
			"git diff*":        "allow",
			"git log*":         "allow",
			"git show*":        "allow",
			"git status*":      "allow",
			"git branch":       "allow",
			"git branch -v":    "allow",
			"grep*":            "allow",
			"head*":            "allow",
			"less*":            "allow",
			"ls*":              "allow",
			"more*":            "allow",
			"pwd*":             "allow",
			"rg*":              "allow",
			"sort --output=*":  "ask",
			"sort -o *":        "ask",
			"sort*":            "allow",
			"stat*":            "allow",
			"tail*":            "allow",
			"tree -o *":        "ask",
			"tree*":            "allow",
			"uniq*":            "allow",
			"wc*":              "allow",
			"whereis*":         "allow",
			"which*":           "allow",
			"*":                "ask",
		},
		WebFetch: "allow",
	}

	return []AgentInfo{
		{
			Name:        "general",
			Description: "General-purpose agent for researching complex questions and executing multi-step tasks. Use this agent to execute multiple units of work in parallel.",
			Mode:        "subagent",
			BuiltIn:     true,
			Tools: map[string]bool{
				"todoread":  false,
				"todowrite": false,
			},
			Options:    map[string]any{},
			Permission: defaultPermission,
		},
		{
			Name:        "explore",
			Description: `Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (eg. "src/components/**/*.tsx"), search code for keywords (eg. "API endpoints"), or answer questions about the codebase (eg. "how do API endpoints work?"). When calling this agent, specify the desired thoroughness level: "quick" for basic searches, "medium" for moderate exploration, or "very thorough" for comprehensive analysis across multiple locations and naming conventions.`,
			Mode:        "subagent",
			BuiltIn:     true,
			Tools: map[string]bool{
				"todoread":  false,
				"todowrite": false,
				"edit":      false,
				"write":     false,
			},
			Options:    map[string]any{},
			Permission: defaultPermission,
			Prompt: `You are a file search specialist. You excel at thoroughly navigating and exploring codebases.

Your strengths:
- Rapidly finding files using glob patterns
- Searching code and text with powerful regex patterns
- Reading and analyzing file contents

Guidelines:
- Use Glob for broad file pattern matching
- Use Grep for searching file contents with regex
- Use Read when you know the specific file path you need to read
- Use Bash for file operations like copying, moving, or listing directory contents
- Adapt your search approach based on the thoroughness level specified by the caller
- Return file paths as absolute paths in your final response
- For clear communication, avoid using emojis
- Do not create any files, or run bash commands that modify the user's system state in any way

Complete the user's search request efficiently and report your findings clearly.`,
		},
		{
			Name:       "build",
			Mode:       "primary",
			BuiltIn:    true,
			Tools:      map[string]bool{},
			Options:    map[string]any{},
			Permission: defaultPermission,
		},
		{
			Name:       "plan",
			Mode:       "primary",
			BuiltIn:    true,
			Tools:      map[string]bool{},
			Options:    map[string]any{},
			Permission: planPermission,
		},
	}
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

// CommandInfo represents command information returned by the /command endpoint.
// SDK compatible: matches TypeScript Command.Info structure.
type CommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Template    string `json:"template"`
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
}

// listCommands handles GET /command
// Returns full command objects matching TypeScript Command.Info structure.
func (s *Server) listCommands(w http.ResponseWriter, r *http.Request) {
	commands := make([]CommandInfo, 0)

	// Add built-in commands with templates
	builtinCommands := getBuiltInCommands(s.config.Directory)
	commands = append(commands, builtinCommands...)

	// Add custom commands from executor (config and file-based)
	if s.commandExecutor != nil {
		for _, cmd := range s.commandExecutor.List() {
			commands = append(commands, CommandInfo{
				Name:        cmd.Name,
				Description: cmd.Description,
				Template:    cmd.Template,
				Agent:       cmd.Agent,
				Model:       cmd.Model,
				Subtask:     cmd.Subtask,
			})
		}
	}

	writeJSON(w, http.StatusOK, commands)
}

// getBuiltInCommands returns the built-in commands with their templates.
func getBuiltInCommands(workDir string) []CommandInfo {
	return []CommandInfo{
		{
			Name:        "init",
			Description: "create/update AGENTS.md",
			Template: `Please analyze this codebase and create an AGENTS.md file containing:
1. Build/lint/test commands - especially for running a single test
2. Code style guidelines including imports, formatting, types, naming conventions, error handling, etc.

The file you create will be given to agentic coding agents (such as yourself) that operate in this repository. Make it about 20 lines long.
If there are Cursor rules (in .cursor/rules/ or .cursorrules) or Copilot rules (in .github/copilot-instructions.md), make sure to include them.

If there's already an AGENTS.md, improve it if it's located in ` + workDir + `

$ARGUMENTS
`,
		},
		{
			Name:        "review",
			Description: "review changes [commit|branch|pr], defaults to uncommitted",
			Template: `You are a code reviewer. Your job is to review code changes and provide actionable feedback.

---

Input: $ARGUMENTS

---

## Determining What to Review

Based on the input provided, determine which type of review to perform:

1. **No arguments (default)**: Review all uncommitted changes
   - Run: ` + "`git diff`" + ` for unstaged changes
   - Run: ` + "`git diff --cached`" + ` for staged changes

2. **Commit hash** (40-char SHA or short hash): Review that specific commit
   - Run: ` + "`git show $ARGUMENTS`" + `

3. **Branch name**: Compare current branch to the specified branch
   - Run: ` + "`git diff $ARGUMENTS...HEAD`" + `

4. **PR URL or number** (contains "github.com" or "pull" or looks like a PR number): Review the pull request
   - Run: ` + "`gh pr view $ARGUMENTS`" + ` to get PR context
   - Run: ` + "`gh pr diff $ARGUMENTS`" + ` to get the diff

Use best judgement when processing input.

---

## What to Look For

**Bugs** - Your primary focus.
- Logic errors, off-by-one mistakes, incorrect conditionals
- Edge cases: null/empty inputs, error conditions, race conditions
- Security issues: injection, auth bypass, data exposure
- Broken error handling that swallows failures

**Structure** - Does the code fit the codebase?
- Does it follow existing patterns and conventions?
- Are there established abstractions it should use but doesn't?

**Performance** - Only flag if obviously problematic.
- O(n²) on unbounded data, N+1 queries, blocking I/O on hot paths

## Before You Flag Something

Be certain. If you're going to call something a bug, you need to be confident it actually is one.

- Only review the changes - do not review pre-existing code that wasn't modified
- Don't flag something as a bug if you're unsure - investigate first
- Don't flag style preferences as issues
- Don't invent hypothetical problems - if an edge case matters, explain the realistic scenario where it breaks
- If you need more context to be sure, use the tools below to get it

## Tools

Use these to inform your review:

- **Explore agent** - Find how existing code handles similar problems. Check patterns, conventions, and prior art before claiming something doesn't fit.
- **Exa Code Context** - Verify correct usage of libraries/APIs before flagging something as wrong.
- **Exa Web Search** - Research best practices if you're unsure about a pattern.

If you're uncertain about something and can't verify it with these tools, say "I'm not sure about X" rather than flagging it as a definite issue.

## Tone and Approach

1. If there is a bug, be direct and clear about why it is a bug.
2. You should clearly communicate severity of issues, do not claim issues are more severe than they actually are.
3. Critiques should clearly and explicitly communicate the scenarios, environments, or inputs that are necessary for the bug to arise. The comment should immediately indicate that the issue's severity depends on these factors.
4. Your tone should be matter-of-fact and not accusatory or overly positive. It should read as a helpful AI assistant suggestion without sounding too much like a human reviewer.
5. Write in a manner that allows reader to quickly understand issue without reading too closely.
6. AVOID flattery, do not give any comments that are not helpful to the reader. Avoid phrasing like "Great job ...", "Thanks for ...".
`,
			Subtask: true,
		},
	}
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
