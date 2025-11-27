package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// MockLLMServer provides an HTTP server that mimics OpenAI/Anthropic APIs for testing.
type MockLLMServer struct {
	server   *httptest.Server
	config   *MockLLMConfig
	requests []MockRequest
	mu       sync.Mutex
}

// MockRequest records incoming requests for verification.
type MockRequest struct {
	Timestamp time.Time
	Method    string
	Path      string
	Body      map[string]interface{}
}

// NewMockLLMServer creates a new mock LLM server with default configuration.
func NewMockLLMServer() *MockLLMServer {
	return NewMockLLMServerWithConfig(DefaultMockLLMConfig())
}

// NewMockLLMServerWithConfig creates a new mock LLM server with custom configuration.
func NewMockLLMServerWithConfig(config *MockLLMConfig) *MockLLMServer {
	m := &MockLLMServer{
		config:   config,
		requests: make([]MockRequest, 0),
	}

	mux := http.NewServeMux()

	// OpenAI-compatible endpoint
	mux.HandleFunc("/v1/chat/completions", m.handleChatCompletions)
	mux.HandleFunc("/chat/completions", m.handleChatCompletions)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	m.server = httptest.NewServer(mux)
	return m
}

// NewMockLLMServerFromFile creates a mock LLM server from a YAML config file.
func NewMockLLMServerFromFile(configPath string) (*MockLLMServer, error) {
	config, err := LoadMockLLMConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return NewMockLLMServerWithConfig(config), nil
}

// URL returns the mock server's URL.
func (m *MockLLMServer) URL() string {
	return m.server.URL
}

// Close shuts down the mock server.
func (m *MockLLMServer) Close() {
	m.server.Close()
}

// GetRequests returns all recorded requests.
func (m *MockLLMServer) GetRequests() []MockRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockRequest{}, m.requests...)
}

// ClearRequests clears all recorded requests.
func (m *MockLLMServer) ClearRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = make([]MockRequest, 0)
}

// GetConfig returns the current configuration.
func (m *MockLLMServer) GetConfig() *MockLLMConfig {
	return m.config
}

// SetConfig updates the configuration at runtime.
func (m *MockLLMServer) SetConfig(config *MockLLMConfig) {
	m.config = config
}

// AddResponse adds a response rule at runtime.
func (m *MockLLMServer) AddResponse(rule ResponseRule) {
	m.config.Responses = append(m.config.Responses, rule)
}

// AddToolRule adds a tool rule at runtime.
func (m *MockLLMServer) AddToolRule(rule ToolRule) {
	m.config.ToolRules = append(m.config.ToolRules, rule)
}

// handleChatCompletions handles OpenAI-compatible chat completions.
func (m *MockLLMServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Record request
	m.mu.Lock()
	m.requests = append(m.requests, MockRequest{
		Timestamp: time.Now(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Body:      req,
	})
	m.mu.Unlock()

	// Apply artificial lag if configured
	if m.config.Settings.LagMS > 0 {
		time.Sleep(time.Duration(m.config.Settings.LagMS) * time.Millisecond)
	}

	// Extract last message content for matching
	lastPrompt := m.extractLastPrompt(req)
	tools := m.extractTools(req)

	// Check if streaming is requested
	stream, _ := req["stream"].(bool)

	// Generate appropriate response based on prompt and available tools
	response := m.generateResponse(lastPrompt, tools)

	if stream && m.config.Settings.EnableStreaming {
		m.writeStreamingResponse(w, response)
	} else {
		m.writeResponse(w, response)
	}
}

// extractLastPrompt extracts the last user message from OpenAI format.
func (m *MockLLMServer) extractLastPrompt(req map[string]interface{}) string {
	messages, ok := req["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return ""
	}

	// Find last user message
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(map[string]interface{})
		if !ok {
			continue
		}
		if role, ok := msg["role"].(string); ok && role == "user" {
			if content, ok := msg["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}

// extractTools extracts tool definitions from the request
func (m *MockLLMServer) extractTools(req map[string]interface{}) []string {
	var toolNames []string
	tools, ok := req["tools"].([]interface{})
	if !ok {
		return toolNames
	}
	for _, t := range tools {
		tool, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		fn, ok := tool["function"].(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := fn["name"].(string); ok {
			toolNames = append(toolNames, name)
		}
	}
	return toolNames
}

// mockResponse represents a response with optional tool calls
type mockResponse struct {
	content   string
	toolCalls []toolCall
}

// toolCall represents a tool call in the response
type toolCall struct {
	id        string
	name      string
	arguments string
}

// generateResponse generates a response based on the config rules.
func (m *MockLLMServer) generateResponse(prompt string, tools []string) *mockResponse {
	// First, check for tool rules
	if len(tools) > 0 {
		if toolRule := m.config.FindMatchingToolRule(prompt, tools); toolRule != nil {
			// Build tool call arguments as JSON
			argsJSON, _ := json.Marshal(toolRule.ToolCall.Arguments)

			// Generate tool call ID if not specified
			callID := toolRule.ToolCall.ID
			if callID == "" {
				callID = fmt.Sprintf("call_%s_%s", toolRule.Tool, generateMockID())
			}

			return &mockResponse{
				content: toolRule.Response,
				toolCalls: []toolCall{
					{
						id:        callID,
						name:      toolRule.Tool,
						arguments: string(argsJSON),
					},
				},
			}
		}

		// Fallback to legacy tool handling for prompts not in config
		if resp := m.legacyToolHandling(prompt, tools); resp != nil {
			return resp
		}
	}

	// Check for matching response rule
	response, _ := m.config.FindMatchingResponse(prompt)
	return &mockResponse{content: response}
}

// legacyToolHandling provides backward-compatible tool handling for prompts
// not covered by the config. This ensures existing tests continue to work.
func (m *MockLLMServer) legacyToolHandling(prompt string, tools []string) *mockResponse {
	promptLower := strings.ToLower(prompt)

	hasBashTool := containsTool(tools, "bash")
	hasReadTool := containsTool(tools, "read")

	// Handle bash commands
	if hasBashTool && (strings.Contains(promptLower, "run") || strings.Contains(promptLower, "bash") || strings.Contains(promptLower, "execute")) {
		if strings.Contains(promptLower, "ls ") {
			// Extract the path from the prompt
			parts := strings.Split(prompt, "'")
			if len(parts) >= 2 {
				cmd := parts[1]
				return &mockResponse{
					content: "I'll list the files in that directory.",
					toolCalls: []toolCall{
						{
							id:        "call_bash_002",
							name:      "bash",
							arguments: `{"command": "` + cmd + `"}`,
						},
					},
				}
			}
		}
	}

	// Handle file reading
	if hasReadTool && (strings.Contains(promptLower, "read") || strings.Contains(promptLower, "file")) {
		words := strings.Fields(prompt)
		for _, word := range words {
			if strings.HasPrefix(word, "/") || strings.Contains(word, ".txt") || strings.Contains(word, ".go") {
				path := strings.Trim(word, ".,")
				return &mockResponse{
					content: "I'll read that file for you.",
					toolCalls: []toolCall{
						{
							id:        "call_read_001",
							name:      "read",
							arguments: `{"file_path": "` + path + `"}`,
						},
					},
				}
			}
		}
	}

	return nil
}

// containsTool checks if a tool name is in the list
func containsTool(tools []string, name string) bool {
	for _, t := range tools {
		if t == name {
			return true
		}
	}
	return false
}

// writeResponse writes a non-streaming OpenAI response.
func (m *MockLLMServer) writeResponse(w http.ResponseWriter, resp *mockResponse) {
	response := map[string]interface{}{
		"id":      "chatcmpl-mockllm-" + generateMockID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-4o-mini",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": resp.content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     100,
			"completion_tokens": 50,
			"total_tokens":      150,
		},
	}

	// Add tool calls if present
	if len(resp.toolCalls) > 0 {
		toolCalls := make([]map[string]interface{}, len(resp.toolCalls))
		for i, tc := range resp.toolCalls {
			toolCalls[i] = map[string]interface{}{
				"id":   tc.id,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.name,
					"arguments": tc.arguments,
				},
			}
		}
		response["choices"].([]map[string]interface{})[0]["message"].(map[string]interface{})["tool_calls"] = toolCalls
		response["choices"].([]map[string]interface{})[0]["finish_reason"] = "tool_calls"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeStreamingResponse writes a streaming OpenAI response.
func (m *MockLLMServer) writeStreamingResponse(w http.ResponseWriter, resp *mockResponse) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// First chunk with role
	firstChunk := map[string]interface{}{
		"id":      "chatcmpl-mockllm-" + generateMockID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "gpt-4o-mini",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"role": "assistant",
				},
			},
		},
	}
	data, _ := json.Marshal(firstChunk)
	w.Write([]byte("data: " + string(data) + "\n\n"))
	flusher.Flush()

	// Get chunk delay from config
	chunkDelay := time.Duration(m.config.Settings.ChunkDelayMS) * time.Millisecond
	if chunkDelay <= 0 {
		chunkDelay = 5 * time.Millisecond
	}

	// Handle tool calls
	if len(resp.toolCalls) > 0 {
		for _, tc := range resp.toolCalls {
			// Tool call chunk
			toolChunk := map[string]interface{}{
				"id":      "chatcmpl-mockllm-" + generateMockID(),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "gpt-4o-mini",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"tool_calls": []map[string]interface{}{
								{
									"index": 0,
									"id":    tc.id,
									"type":  "function",
									"function": map[string]interface{}{
										"name":      tc.name,
										"arguments": tc.arguments,
									},
								},
							},
						},
					},
				},
			}
			data, _ = json.Marshal(toolChunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
		}
	} else {
		// Split content into chunks based on settings
		chunks := m.splitIntoChunks(resp.content)
		for _, chunkContent := range chunks {
			chunk := map[string]interface{}{
				"id":      "chatcmpl-mockllm-" + generateMockID(),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "gpt-4o-mini",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": chunkContent,
						},
					},
				},
			}

			data, _ = json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()

			time.Sleep(chunkDelay)
		}
	}

	// Send finish chunk
	finishReason := "stop"
	if len(resp.toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	finishChunk := map[string]interface{}{
		"id":      "chatcmpl-mockllm-" + generateMockID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "gpt-4o-mini",
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": finishReason,
			},
		},
	}
	data, _ = json.Marshal(finishChunk)
	w.Write([]byte("data: " + string(data) + "\n\n"))
	w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

// splitIntoChunks splits content based on the configured chunk mode.
func (m *MockLLMServer) splitIntoChunks(content string) []string {
	mode := m.config.Settings.ChunkMode
	size := m.config.Settings.ChunkSize
	maxChunks := m.config.Settings.MaxChunks

	var chunks []string

	switch mode {
	case "char":
		// Split by character count
		if size <= 0 {
			size = 1
		}
		for i := 0; i < len(content); i += size {
			end := i + size
			if end > len(content) {
				end = len(content)
			}
			chunks = append(chunks, content[i:end])
		}

	case "fixed":
		// Fixed number of chunks (use maxChunks as the target count)
		numChunks := maxChunks
		if numChunks <= 0 {
			numChunks = 1
		}
		chunkLen := (len(content) + numChunks - 1) / numChunks
		if chunkLen < 1 {
			chunkLen = 1
		}
		for i := 0; i < len(content); i += chunkLen {
			end := i + chunkLen
			if end > len(content) {
				end = len(content)
			}
			chunks = append(chunks, content[i:end])
		}
		// For fixed mode, maxChunks is the target, not a limit
		return chunks

	default: // "word" mode (default)
		// Split by words
		words := strings.Fields(content)
		for i, word := range words {
			chunk := word
			if i < len(words)-1 {
				chunk += " "
			}
			chunks = append(chunks, chunk)
		}
	}

	// Apply maxChunks limit (except for "fixed" mode which returns early)
	if maxChunks > 0 && len(chunks) > maxChunks {
		// Combine excess chunks into the last one
		combined := strings.Join(chunks[maxChunks-1:], "")
		chunks = append(chunks[:maxChunks-1], combined)
	}

	return chunks
}

// generateMockID generates a simple mock ID.
func generateMockID() string {
	return fmt.Sprintf("mock%d", time.Now().UnixNano()%1000000)
}
