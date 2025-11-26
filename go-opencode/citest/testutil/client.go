package testutil

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TestClient provides HTTP client utilities for testing
type TestClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewTestClient creates a new test HTTP client
func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// RequestOption configures HTTP requests
type RequestOption func(*http.Request)

// WithHeader adds a header to the request
func WithHeader(key, value string) RequestOption {
	return func(r *http.Request) {
		r.Header.Set(key, value)
	}
}

// WithQuery adds query parameters
func WithQuery(params map[string]string) RequestOption {
	return func(r *http.Request) {
		q := r.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		r.URL.RawQuery = q.Encode()
	}
}

// Response wraps HTTP response with helpers
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// JSON unmarshals response body into v
func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// String returns response body as string
func (r *Response) String() string {
	return string(r.Body)
}

// IsSuccess returns true if status code is 2xx
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// Get performs HTTP GET request
func (c *TestClient) Get(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, opts...)
}

// Post performs HTTP POST request with JSON body
func (c *TestClient) Post(ctx context.Context, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, body, opts...)
}

// Patch performs HTTP PATCH request with JSON body
func (c *TestClient) Patch(ctx context.Context, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPatch, path, body, opts...)
}

// Delete performs HTTP DELETE request
func (c *TestClient) Delete(ctx context.Context, path string, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, opts...)
}

// do performs the actual HTTP request
func (c *TestClient) do(ctx context.Context, method, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	fullURL := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for _, opt := range opts {
		opt(req)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}, nil
}

// StreamingResponse represents a chunked streaming response
type StreamingResponse struct {
	StatusCode int
	Headers    http.Header
	reader     *bufio.Reader
	body       io.ReadCloser
}

// PostStreaming performs HTTP POST and returns streaming response
func (c *TestClient) PostStreaming(ctx context.Context, path string, body interface{}, opts ...RequestOption) (*StreamingResponse, error) {
	fullURL := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for _, opt := range opts {
		opt(req)
	}

	// Use client without timeout for streaming
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return &StreamingResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		reader:     bufio.NewReader(resp.Body),
		body:       resp.Body,
	}, nil
}

// ReadChunk reads the next JSON chunk from streaming response
func (sr *StreamingResponse) ReadChunk(v interface{}) error {
	line, err := sr.reader.ReadBytes('\n')
	if err != nil {
		return err
	}

	// Skip empty lines
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return sr.ReadChunk(v)
	}

	return json.Unmarshal(line, v)
}

// ReadAllChunks reads all chunks into a slice
func (sr *StreamingResponse) ReadAllChunks(factory func() interface{}) ([]interface{}, error) {
	var chunks []interface{}
	for {
		chunk := factory()
		err := sr.ReadChunk(chunk)
		if err == io.EOF {
			break
		}
		if err != nil {
			return chunks, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// Close closes the streaming response
func (sr *StreamingResponse) Close() error {
	if sr.body != nil {
		return sr.body.Close()
	}
	return nil
}

// ---- Session Helpers ----

// Session represents a session response
type Session struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Directory string `json:"directory"`
}

// CreateSession creates a new session
func (c *TestClient) CreateSession(ctx context.Context, directory string) (*Session, error) {
	resp, err := c.Post(ctx, "/session", map[string]string{
		"directory": directory,
	})
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to create session: %d - %s", resp.StatusCode, resp.String())
	}

	var session Session
	if err := resp.JSON(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSession retrieves a session by ID
func (c *TestClient) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	resp, err := c.Get(ctx, "/session/"+sessionID)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get session: %d - %s", resp.StatusCode, resp.String())
	}

	var session Session
	if err := resp.JSON(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession deletes a session
func (c *TestClient) DeleteSession(ctx context.Context, sessionID string) error {
	resp, err := c.Delete(ctx, "/session/"+sessionID)
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("failed to delete session: %d - %s", resp.StatusCode, resp.String())
	}
	return nil
}

// ListSessions lists all sessions
func (c *TestClient) ListSessions(ctx context.Context) ([]Session, error) {
	resp, err := c.Get(ctx, "/session")
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to list sessions: %d - %s", resp.StatusCode, resp.String())
	}

	var sessions []Session
	if err := resp.JSON(&sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// ---- Message Helpers ----

// MessagePart represents a message part
type MessagePart struct {
	Type    string          `json:"type"`
	Content string          `json:"content,omitempty"`
	Tool    json.RawMessage `json:"tool,omitempty"`
}

// Message represents a message
type Message struct {
	ID        string        `json:"id"`
	SessionID string        `json:"sessionID"`
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	Parts     []MessagePart `json:"parts,omitempty"`
}

// MessageResponse represents the streaming message response
type MessageResponse struct {
	Info  *Message `json:"info,omitempty"`
	Parts []MessagePart `json:"parts,omitempty"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse represents an error
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendMessageRequest represents a send message request
type SendMessageRequest struct {
	Content string `json:"content"`
	Agent   string `json:"agent,omitempty"`
}

// SendMessage sends a message and waits for complete response
func (c *TestClient) SendMessage(ctx context.Context, sessionID, content string) (*MessageResponse, error) {
	stream, err := c.PostStreaming(ctx, "/session/"+sessionID+"/message", SendMessageRequest{
		Content: content,
	})
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	if stream.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to send message: %d", stream.StatusCode)
	}

	// Read all chunks and return the last one (final response)
	var lastResponse *MessageResponse
	for {
		var resp MessageResponse
		err := stream.ReadChunk(&resp)
		if err == io.EOF {
			break
		}
		if err != nil {
			if lastResponse != nil {
				return lastResponse, nil
			}
			return nil, err
		}
		lastResponse = &resp
	}

	return lastResponse, nil
}

// SendMessageStreaming sends a message and returns the stream
func (c *TestClient) SendMessageStreaming(ctx context.Context, sessionID, content string) (*StreamingResponse, error) {
	return c.PostStreaming(ctx, "/session/"+sessionID+"/message", SendMessageRequest{
		Content: content,
	})
}

// GetMessages retrieves all messages in a session
func (c *TestClient) GetMessages(ctx context.Context, sessionID string) ([]Message, error) {
	resp, err := c.Get(ctx, "/session/"+sessionID+"/message")
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get messages: %d - %s", resp.StatusCode, resp.String())
	}

	var messages []Message
	if err := resp.JSON(&messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// ---- File Helpers ----

// FileEntry represents a file/directory entry
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

// FileContent represents file content response
type FileContent struct {
	Content   string `json:"content"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
}

// ListFiles lists directory contents
func (c *TestClient) ListFiles(ctx context.Context, path string) ([]FileEntry, error) {
	resp, err := c.Get(ctx, "/file", WithQuery(map[string]string{"path": path}))
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to list files: %d - %s", resp.StatusCode, resp.String())
	}

	var entries []FileEntry
	if err := resp.JSON(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ReadFile reads file content
func (c *TestClient) ReadFile(ctx context.Context, path string) (*FileContent, error) {
	resp, err := c.Get(ctx, "/file/content", WithQuery(map[string]string{"path": path}))
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to read file: %d - %s", resp.StatusCode, resp.String())
	}

	var content FileContent
	if err := resp.JSON(&content); err != nil {
		return nil, err
	}
	return &content, nil
}

// ---- Config Helpers ----

// Provider represents a provider
type Provider struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Models []Model `json:"models"`
}

// Model represents a model
type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"contextLength"`
}

// GetProviders lists available providers
func (c *TestClient) GetProviders(ctx context.Context) ([]Provider, error) {
	resp, err := c.Get(ctx, "/config/providers")
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get providers: %d - %s", resp.StatusCode, resp.String())
	}

	var providers []Provider
	if err := resp.JSON(&providers); err != nil {
		return nil, err
	}
	return providers, nil
}

// ---- Search Helpers ----

// SearchMatch represents a search match
type SearchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// SearchResult represents search results
type SearchResult struct {
	Matches   []SearchMatch `json:"matches"`
	Count     int           `json:"count"`
	Truncated bool          `json:"truncated"`
}

// SearchText searches for text in files
func (c *TestClient) SearchText(ctx context.Context, query string) (*SearchResult, error) {
	resp, err := c.Get(ctx, "/find", WithQuery(map[string]string{"query": url.QueryEscape(query)}))
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to search: %d - %s", resp.StatusCode, resp.String())
	}

	var result SearchResult
	if err := resp.JSON(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SearchFiles searches for files by pattern
func (c *TestClient) SearchFiles(ctx context.Context, pattern string) ([]string, error) {
	resp, err := c.Get(ctx, "/find/file", WithQuery(map[string]string{"pattern": pattern}))
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to search files: %d - %s", resp.StatusCode, resp.String())
	}

	var files []string
	if err := resp.JSON(&files); err != nil {
		return nil, err
	}
	return files, nil
}

// ---- Assertion Helpers ----

// ContainsString checks if a string slice contains a value
func ContainsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// ContainsSubstring checks if any string in slice contains substring
func ContainsSubstring(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
