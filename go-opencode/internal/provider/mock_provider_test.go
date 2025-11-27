// Package provider_test provides MockLLM server for testing providers.
// The MockLLM server mimics OpenAI and Anthropic APIs with deterministic responses.
package provider_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// MockLLMConfig represents the configuration for MockLLM responses.
type MockLLMConfig struct {
	Responses map[string]MockResponse
	Defaults  MockDefaults
	Settings  MockSettings
}

// MockResponse represents a predefined response for a specific prompt.
type MockResponse struct {
	Content   string
	ToolCalls []MockToolCall
}

// MockToolCall represents a tool call in a mock response.
type MockToolCall struct {
	ID       string
	Type     string
	Function MockFunctionCall
}

// MockFunctionCall represents a function call in a mock response.
type MockFunctionCall struct {
	Name      string
	Arguments string
}

// MockDefaults provides fallback responses when no match is found.
type MockDefaults struct {
	Fallback string
}

// MockSettings configures mock behavior.
type MockSettings struct {
	LagMS           int
	EnableStreaming bool
}

// MockRequest records incoming requests for verification.
type MockRequest struct {
	Timestamp time.Time
	Method    string
	Path      string
	Body      map[string]interface{}
	Headers   http.Header
}

// MockLLMServer provides an HTTP server that mimics OpenAI/Anthropic APIs.
type MockLLMServer struct {
	server    *httptest.Server
	config    *MockLLMConfig
	requests  []MockRequest
	streaming bool
}

// NewMockLLMServer creates a new mock LLM server.
func NewMockLLMServer(config *MockLLMConfig) *MockLLMServer {
	m := &MockLLMServer{
		config:    config,
		requests:  make([]MockRequest, 0),
		streaming: config.Settings.EnableStreaming,
	}

	mux := http.NewServeMux()

	// OpenAI-compatible endpoint (also used by ARK)
	mux.HandleFunc("/v1/chat/completions", m.handleOpenAIChatCompletions)
	mux.HandleFunc("/chat/completions", m.handleOpenAIChatCompletions)

	// Anthropic-compatible endpoint
	mux.HandleFunc("/v1/messages", m.handleAnthropicMessages)

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

// ClearRequests clears the recorded requests.
func (m *MockLLMServer) ClearRequests() {
	m.requests = make([]MockRequest, 0)
}

// handleOpenAIChatCompletions handles OpenAI-compatible chat completions.
func (m *MockLLMServer) handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request) {
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
		Headers:   r.Header,
	})

	// Extract last message content for matching
	lastPrompt := m.extractLastPrompt(req)
	response := m.findResponse(lastPrompt)

	// Check if streaming is requested
	stream, _ := req["stream"].(bool)

	// Apply lag if configured
	if m.config.Settings.LagMS > 0 {
		time.Sleep(time.Duration(m.config.Settings.LagMS) * time.Millisecond)
	}

	if stream && m.streaming {
		m.writeOpenAIStreamingResponse(w, response)
	} else {
		m.writeOpenAIResponse(w, response)
	}
}

// handleAnthropicMessages handles Anthropic-compatible messages API.
func (m *MockLLMServer) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
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
		Headers:   r.Header,
	})

	// Extract last message content
	lastPrompt := m.extractLastPromptAnthropic(req)
	response := m.findResponse(lastPrompt)

	// Check if streaming is requested
	stream, _ := req["stream"].(bool)

	if stream && m.streaming {
		m.writeAnthropicStreamingResponse(w, response)
	} else {
		m.writeAnthropicResponse(w, response)
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

// extractLastPromptAnthropic extracts the last user message from Anthropic format.
func (m *MockLLMServer) extractLastPromptAnthropic(req map[string]interface{}) string {
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
			// Anthropic content can be string or array
			if content, ok := msg["content"].(string); ok {
				return content
			}
			if contentArr, ok := msg["content"].([]interface{}); ok {
				for _, item := range contentArr {
					if block, ok := item.(map[string]interface{}); ok {
						if blockType, _ := block["type"].(string); blockType == "text" {
							if text, ok := block["text"].(string); ok {
								return text
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// findResponse finds the best matching response for a prompt.
func (m *MockLLMServer) findResponse(prompt string) *MockResponse {
	prompt = strings.ToLower(strings.TrimSpace(prompt))

	// Check exact matches first
	for key, resp := range m.config.Responses {
		if strings.Contains(prompt, strings.ToLower(key)) {
			return &resp
		}
	}

	// Return default fallback
	return &MockResponse{
		Content: m.config.Defaults.Fallback,
	}
}

// writeOpenAIResponse writes a non-streaming OpenAI response.
func (m *MockLLMServer) writeOpenAIResponse(w http.ResponseWriter, resp *MockResponse) {
	response := map[string]interface{}{
		"id":      "chatcmpl-mock-" + generateMockID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "mock-gpt-4",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": resp.Content,
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
	if len(resp.ToolCalls) > 0 {
		toolCalls := make([]map[string]interface{}, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			toolCalls[i] = map[string]interface{}{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			}
		}
		response["choices"].([]map[string]interface{})[0]["message"].(map[string]interface{})["tool_calls"] = toolCalls
		response["choices"].([]map[string]interface{})[0]["finish_reason"] = "tool_calls"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeOpenAIStreamingResponse writes a streaming OpenAI response.
func (m *MockLLMServer) writeOpenAIStreamingResponse(w http.ResponseWriter, resp *MockResponse) {
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
		"id":      "chatcmpl-mock-" + generateMockID(),
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

	// Stream content in chunks (word by word for more realistic streaming)
	words := strings.Fields(resp.Content)
	for i, word := range words {
		content := word
		if i < len(words)-1 {
			content += " "
		}

		chunk := map[string]interface{}{
			"id":      "chatcmpl-mock-" + generateMockID(),
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

		data, _ := json.Marshal(chunk)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Small delay between chunks
		time.Sleep(5 * time.Millisecond)
	}

	// Send finish chunk
	finishChunk := map[string]interface{}{
		"id":      "chatcmpl-mock-" + generateMockID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "mock-gpt-4",
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": "stop",
			},
		},
	}
	data, _ = json.Marshal(finishChunk)
	w.Write([]byte("data: " + string(data) + "\n\n"))
	w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

// writeAnthropicResponse writes a non-streaming Anthropic response.
func (m *MockLLMServer) writeAnthropicResponse(w http.ResponseWriter, resp *MockResponse) {
	response := map[string]interface{}{
		"id":            "msg_mock_" + generateMockID(),
		"type":          "message",
		"role":          "assistant",
		"model":         "mock-claude-3",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": resp.Content,
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  100,
			"output_tokens": 50,
		},
	}

	// Add tool use if present
	if len(resp.ToolCalls) > 0 {
		content := response["content"].([]map[string]interface{})
		for _, tc := range resp.ToolCalls {
			var input map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
			content = append(content, map[string]interface{}{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Function.Name,
				"input": input,
			})
		}
		response["content"] = content
		response["stop_reason"] = "tool_use"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeAnthropicStreamingResponse writes a streaming Anthropic response.
func (m *MockLLMServer) writeAnthropicStreamingResponse(w http.ResponseWriter, resp *MockResponse) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Message start
	msgStart := map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":      "msg_mock_" + generateMockID(),
			"type":    "message",
			"role":    "assistant",
			"model":   "mock-claude-3",
			"content": []interface{}{},
			"usage": map[string]interface{}{
				"input_tokens":  100,
				"output_tokens": 0,
			},
		},
	}
	data, _ := json.Marshal(msgStart)
	w.Write([]byte("event: message_start\ndata: " + string(data) + "\n\n"))
	flusher.Flush()

	// Content block start
	blockStart := map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]interface{}{
			"type": "text",
			"text": "",
		},
	}
	data, _ = json.Marshal(blockStart)
	w.Write([]byte("event: content_block_start\ndata: " + string(data) + "\n\n"))
	flusher.Flush()

	// Stream content word by word
	words := strings.Fields(resp.Content)
	for i, word := range words {
		content := word
		if i < len(words)-1 {
			content += " "
		}

		delta := map[string]interface{}{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": content,
			},
		}
		data, _ := json.Marshal(delta)
		w.Write([]byte("event: content_block_delta\ndata: " + string(data) + "\n\n"))
		flusher.Flush()
		time.Sleep(5 * time.Millisecond)
	}

	// Content block stop
	blockStop := map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	}
	data, _ = json.Marshal(blockStop)
	w.Write([]byte("event: content_block_stop\ndata: " + string(data) + "\n\n"))
	flusher.Flush()

	// Message delta with stop reason
	msgDelta := map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   "end_turn",
			"stop_sequence": nil,
		},
		"usage": map[string]interface{}{
			"output_tokens": 50,
		},
	}
	data, _ = json.Marshal(msgDelta)
	w.Write([]byte("event: message_delta\ndata: " + string(data) + "\n\n"))
	flusher.Flush()

	// Message stop
	w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	flusher.Flush()
}

// generateMockID generates a simple mock ID.
func generateMockID() string {
	return "mock123456"
}
