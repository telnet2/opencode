package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// MockLLMServer provides an HTTP server that mimics OpenAI/Anthropic APIs for testing.
type MockLLMServer struct {
	server   *httptest.Server
	requests []MockRequest
}

// MockRequest records incoming requests for verification.
type MockRequest struct {
	Timestamp time.Time
	Method    string
	Path      string
	Body      map[string]interface{}
}

// NewMockLLMServer creates a new mock LLM server with predefined responses.
func NewMockLLMServer() *MockLLMServer {
	m := &MockLLMServer{
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
	return m.requests
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
	m.requests = append(m.requests, MockRequest{
		Timestamp: time.Now(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Body:      req,
	})

	// Extract last message content for matching
	lastPrompt := m.extractLastPrompt(req)
	tools := m.extractTools(req)

	// Check if streaming is requested
	stream, _ := req["stream"].(bool)

	// Generate appropriate response based on prompt and available tools
	response := m.generateResponse(lastPrompt, tools)

	if stream {
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

// generateResponse generates a response based on the prompt and available tools.
func (m *MockLLMServer) generateResponse(prompt string, tools []string) *mockResponse {
	promptLower := strings.ToLower(prompt)

	// Check for tool-related prompts
	hasBashTool := containsTool(tools, "bash")
	hasReadTool := containsTool(tools, "read")

	// Handle bash commands
	if hasBashTool && (strings.Contains(promptLower, "run") || strings.Contains(promptLower, "bash") || strings.Contains(promptLower, "execute")) {
		// Extract command to run
		if strings.Contains(promptLower, "echo hello world") {
			return &mockResponse{
				content: "I'll run that bash command for you.",
				toolCalls: []toolCall{
					{
						id:        "call_bash_001",
						name:      "bash",
						arguments: `{"command": "echo hello world"}`,
					},
				},
			}
		}
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
		// Extract file path - look for paths in the prompt
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

	// Handle simple prompts (no tools needed)
	switch {
	case strings.Contains(promptLower, "hello, world"):
		return &mockResponse{content: "Hello, World!"}

	case strings.Contains(promptLower, "2+2") || strings.Contains(promptLower, "2 + 2"):
		return &mockResponse{content: "4"}

	case strings.Contains(promptLower, "remember") && strings.Contains(promptLower, "42"):
		return &mockResponse{content: "OK"}

	case strings.Contains(promptLower, "what number") && strings.Contains(promptLower, "remember"):
		return &mockResponse{content: "42"}

	case strings.Contains(promptLower, "alice") && strings.Contains(promptLower, "name"):
		return &mockResponse{content: "Nice to meet you, Alice"}

	case strings.Contains(promptLower, "what") && strings.Contains(promptLower, "name"):
		return &mockResponse{content: "Alice"}

	case strings.Contains(promptLower, "hello"):
		return &mockResponse{content: "Hello! How can I help you today?"}

	default:
		return &mockResponse{content: "I understand your request. Let me help you with that."}
	}
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
		"model":   "mock-gpt-4",
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
		"model":   "mock-gpt-4",
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

	// Handle tool calls
	if len(resp.toolCalls) > 0 {
		for _, tc := range resp.toolCalls {
			// Tool call chunk
			toolChunk := map[string]interface{}{
				"id":      "chatcmpl-mockllm-" + generateMockID(),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "mock-gpt-4",
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
		// Stream content word by word
		words := strings.Fields(resp.content)
		for i, word := range words {
			content := word
			if i < len(words)-1 {
				content += " "
			}

			chunk := map[string]interface{}{
				"id":      "chatcmpl-mockllm-" + generateMockID(),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "mock-gpt-4",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": content,
						},
					},
				},
			}

			data, _ = json.Marshal(chunk)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()

			// Small delay between chunks
			time.Sleep(5 * time.Millisecond)
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
		"model":   "mock-gpt-4",
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

// generateMockID generates a simple mock ID.
func generateMockID() string {
	return "mock123456"
}
