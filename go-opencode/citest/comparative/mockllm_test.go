// Package comparative provides comparative testing between TypeScript and Go implementations
// using MockLLM for deterministic LLM responses.
package comparative_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockLLMConfig represents the configuration for MockLLM responses.
type MockLLMConfig struct {
	Responses map[string]MockResponse `yaml:"responses"`
	Defaults  MockDefaults            `yaml:"defaults"`
	Settings  MockSettings            `yaml:"settings"`
}

// MockResponse represents a predefined response for a specific prompt.
type MockResponse struct {
	Content   string   `yaml:"content"`
	ToolCalls []MockToolCall `yaml:"tool_calls,omitempty"`
}

// MockToolCall represents a tool call in a mock response.
type MockToolCall struct {
	ID       string          `yaml:"id"`
	Type     string          `yaml:"type"`
	Function MockFunctionCall `yaml:"function"`
}

// MockFunctionCall represents a function call in a mock response.
type MockFunctionCall struct {
	Name      string `yaml:"name"`
	Arguments string `yaml:"arguments"`
}

// MockDefaults provides fallback responses when no match is found.
type MockDefaults struct {
	Fallback string `yaml:"fallback"`
}

// MockSettings configures mock behavior.
type MockSettings struct {
	LagMS           int  `yaml:"lag_ms"`
	EnableStreaming bool `yaml:"enable_streaming"`
}

// MockLLMServer provides an HTTP server that mimics OpenAI/Anthropic APIs.
type MockLLMServer struct {
	server    *httptest.Server
	config    *MockLLMConfig
	requests  []MockRequest
	streaming bool
}

// MockRequest records incoming requests for verification.
type MockRequest struct {
	Timestamp time.Time
	Method    string
	Path      string
	Body      map[string]interface{}
}

// NewMockLLMServer creates a new mock LLM server.
func NewMockLLMServer(config *MockLLMConfig) *MockLLMServer {
	m := &MockLLMServer{
		config:    config,
		requests:  make([]MockRequest, 0),
		streaming: config.Settings.EnableStreaming,
	}

	mux := http.NewServeMux()

	// OpenAI-compatible endpoint
	mux.HandleFunc("/v1/chat/completions", m.handleOpenAIChatCompletions)

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
		"id":      "chatcmpl-mock-" + generateID(),
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

	// Stream content character by character
	for i, char := range resp.Content {
		chunk := map[string]interface{}{
			"id":      "chatcmpl-mock-" + generateID(),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "mock-gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": string(char),
					},
				},
			},
		}

		// Add role on first chunk
		if i == 0 {
			chunk["choices"].([]map[string]interface{})[0]["delta"].(map[string]interface{})["role"] = "assistant"
		}

		data, _ := json.Marshal(chunk)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		// Small delay between chunks for realistic streaming
		time.Sleep(10 * time.Millisecond)
	}

	// Send finish chunk
	finishChunk := map[string]interface{}{
		"id":      "chatcmpl-mock-" + generateID(),
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
	data, _ := json.Marshal(finishChunk)
	w.Write([]byte("data: " + string(data) + "\n\n"))
	w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

// writeAnthropicResponse writes a non-streaming Anthropic response.
func (m *MockLLMServer) writeAnthropicResponse(w http.ResponseWriter, resp *MockResponse) {
	response := map[string]interface{}{
		"id":           "msg_mock_" + generateID(),
		"type":         "message",
		"role":         "assistant",
		"model":        "mock-claude-3",
		"stop_reason":  "end_turn",
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
			"id":    "msg_mock_" + generateID(),
			"type":  "message",
			"role":  "assistant",
			"model": "mock-claude-3",
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
		"type":         "content_block_start",
		"index":        0,
		"content_block": map[string]interface{}{
			"type": "text",
			"text": "",
		},
	}
	data, _ = json.Marshal(blockStart)
	w.Write([]byte("event: content_block_start\ndata: " + string(data) + "\n\n"))
	flusher.Flush()

	// Stream content
	for _, char := range resp.Content {
		delta := map[string]interface{}{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": string(char),
			},
		}
		data, _ := json.Marshal(delta)
		w.Write([]byte("event: content_block_delta\ndata: " + string(data) + "\n\n"))
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)
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

// generateID generates a simple mock ID.
func generateID() string {
	return "mock123456"
}

// ===== Tests =====

var _ = Describe("MockLLM Server", func() {
	var mockServer *MockLLMServer

	BeforeEach(func() {
		config := &MockLLMConfig{
			Responses: map[string]MockResponse{
				"hello": {
					Content: "Hello! How can I help you today?",
				},
				"what is 2+2": {
					Content: "4",
				},
				"read file": {
					Content: "I'll read that file for you.",
					ToolCalls: []MockToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: MockFunctionCall{
								Name:      "read_file",
								Arguments: `{"path": "/test.txt"}`,
							},
						},
					},
				},
			},
			Defaults: MockDefaults{
				Fallback: "I understand your request.",
			},
			Settings: MockSettings{
				LagMS:           0,
				EnableStreaming: true,
			},
		}
		mockServer = NewMockLLMServer(config)
	})

	AfterEach(func() {
		mockServer.Close()
	})

	Describe("OpenAI Compatibility", func() {
		It("should handle basic chat completion", func() {
			body := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "hello"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			Expect(result["object"]).To(Equal("chat.completion"))
			choices := result["choices"].([]interface{})
			Expect(len(choices)).To(Equal(1))

			msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
			Expect(msg["role"]).To(Equal("assistant"))
			Expect(msg["content"]).To(ContainSubstring("Hello"))
		})

		It("should record requests", func() {
			body := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "test"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))

			requests := mockServer.GetRequests()
			Expect(len(requests)).To(Equal(1))
			Expect(requests[0].Path).To(Equal("/v1/chat/completions"))
		})

		It("should handle streaming responses", func() {
			body := map[string]interface{}{
				"model":  "gpt-4",
				"stream": true,
				"messages": []map[string]interface{}{
					{"role": "user", "content": "hello"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal("text/event-stream"))

			// Read all events
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			bodyStr := string(bodyBytes)
			Expect(bodyStr).To(ContainSubstring("data:"))
			Expect(bodyStr).To(ContainSubstring("[DONE]"))
		})

		It("should include tool calls in response", func() {
			body := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "please read file /test.txt"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
			Expect(err).NotTo(HaveOccurred())

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			choices := result["choices"].([]interface{})
			msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
			Expect(msg["tool_calls"]).NotTo(BeNil())

			toolCalls := msg["tool_calls"].([]interface{})
			Expect(len(toolCalls)).To(Equal(1))
			tc := toolCalls[0].(map[string]interface{})
			fn := tc["function"].(map[string]interface{})
			Expect(fn["name"]).To(Equal("read_file"))
		})
	})

	Describe("Anthropic Compatibility", func() {
		It("should handle basic messages request", func() {
			body := map[string]interface{}{
				"model":      "claude-3-opus",
				"max_tokens": 1024,
				"messages": []map[string]interface{}{
					{"role": "user", "content": "hello"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", mockServer.URL()+"/v1/messages", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", "test-key")
			req.Header.Set("anthropic-version", "2023-06-01")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			Expect(result["type"]).To(Equal("message"))
			Expect(result["role"]).To(Equal("assistant"))

			content := result["content"].([]interface{})
			Expect(len(content)).To(BeNumerically(">", 0))
			textBlock := content[0].(map[string]interface{})
			Expect(textBlock["type"]).To(Equal("text"))
			Expect(textBlock["text"]).To(ContainSubstring("Hello"))
		})

		It("should handle streaming responses", func() {
			body := map[string]interface{}{
				"model":      "claude-3-opus",
				"max_tokens": 1024,
				"stream":     true,
				"messages": []map[string]interface{}{
					{"role": "user", "content": "hello"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(mockServer.URL()+"/v1/messages", "application/json", bytes.NewReader(jsonBody))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal("text/event-stream"))

			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			bodyStr := string(bodyBytes)
			Expect(bodyStr).To(ContainSubstring("event: message_start"))
			Expect(bodyStr).To(ContainSubstring("event: content_block_delta"))
			Expect(bodyStr).To(ContainSubstring("event: message_stop"))
		})
	})

	Describe("Fallback Behavior", func() {
		It("should return default response for unknown prompts", func() {
			body := map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "something completely unknown"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(mockServer.URL()+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
			Expect(err).NotTo(HaveOccurred())

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			choices := result["choices"].([]interface{})
			msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
			Expect(msg["content"]).To(Equal("I understand your request."))
		})
	})
})

func TestComparative(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Comparative Test Suite")
}
