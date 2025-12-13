package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockOpenCodeServer provides a mock OpenCode server for testing the CLI
type MockOpenCodeServer struct {
	server      *httptest.Server
	sessions    map[string]*MockSession
	sessionsMu  sync.RWMutex
	eventSubs   []chan string
	eventSubsMu sync.Mutex
}

type MockSession struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Directory string            `json:"directory"`
	Messages  []MockMessage     `json:"messages"`
	CreatedAt time.Time         `json:"createdAt"`
	Metadata  map[string]string `json:"metadata"`
}

type MockMessage struct {
	ID        string     `json:"id"`
	SessionID string     `json:"sessionID"`
	Role      string     `json:"role"`
	Parts     []MockPart `json:"parts"`
	CreatedAt time.Time  `json:"createdAt"`
}

type MockPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type MockMessageInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Cost      int    `json:"cost"`
	Mode      string `json:"mode"`
	ModelID   string `json:"modelID"`
	ParentID  string `json:"parentID"`
	Path      struct {
		Turn int `json:"turn"`
	} `json:"path"`
	ProviderID string `json:"providerID"`
	System     []string `json:"system"`
	Time       struct {
		Start int64 `json:"start"`
		End   int64 `json:"end"`
	} `json:"time"`
	Tokens struct {
		Input  int `json:"input"`
		Output int `json:"output"`
	} `json:"tokens"`
}

func NewMockOpenCodeServer() *MockOpenCodeServer {
	m := &MockOpenCodeServer{
		sessions:  make(map[string]*MockSession),
		eventSubs: make([]chan string, 0),
	}

	mux := http.NewServeMux()

	// Session endpoints
	mux.HandleFunc("/session", m.handleSessions)
	mux.HandleFunc("/session/", m.handleSession)

	// Event endpoint
	mux.HandleFunc("/event", m.handleEvents)

	// Config endpoint
	mux.HandleFunc("/config", m.handleConfig)
	mux.HandleFunc("/config/providers", m.handleProviders)

	m.server = httptest.NewServer(mux)
	return m
}

func (m *MockOpenCodeServer) URL() string {
	return m.server.URL
}

func (m *MockOpenCodeServer) Close() {
	m.server.Close()
}

func (m *MockOpenCodeServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.listSessions(w, r)
	case http.MethodPost:
		m.createSession(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockOpenCodeServer) listSessions(w http.ResponseWriter, r *http.Request) {
	m.sessionsMu.RLock()
	defer m.sessionsMu.RUnlock()

	sessions := make([]*MockSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (m *MockOpenCodeServer) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Directory string `json:"directory"`
		Title     string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	session := &MockSession{
		ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		Title:     req.Title,
		Directory: req.Directory,
		CreatedAt: time.Now(),
		Messages:  make([]MockMessage, 0),
	}

	m.sessionsMu.Lock()
	m.sessions[session.ID] = session
	m.sessionsMu.Unlock()

	m.publishEvent("session.created", session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (m *MockOpenCodeServer) handleSession(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/session/")
	parts := strings.Split(path, "/")
	sessionID := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			m.getSession(w, r, sessionID)
		case http.MethodDelete:
			m.deleteSession(w, r, sessionID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle sub-resources
	if len(parts) >= 2 {
		switch parts[1] {
		case "message":
			if r.Method == http.MethodPost {
				m.sendMessage(w, r, sessionID)
			} else if r.Method == http.MethodGet {
				m.getMessages(w, r, sessionID)
			}
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}
}

func (m *MockOpenCodeServer) getSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	m.sessionsMu.RLock()
	session, ok := m.sessions[sessionID]
	m.sessionsMu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (m *MockOpenCodeServer) deleteSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	m.sessionsMu.Lock()
	delete(m.sessions, sessionID)
	m.sessionsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(true)
}

func (m *MockOpenCodeServer) sendMessage(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req struct {
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
		Directory string `json:"directory"`
		Agent     string `json:"agent,omitempty"`
		Model     struct {
			ModelID    string `json:"modelID"`
			ProviderID string `json:"providerID"`
		} `json:"model,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	m.sessionsMu.Lock()
	session, ok := m.sessions[sessionID]
	if !ok {
		m.sessionsMu.Unlock()
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Create user message
	userMsg := MockMessage{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		SessionID: sessionID,
		Role:      "user",
		Parts:     make([]MockPart, 0),
		CreatedAt: time.Now(),
	}
	for _, p := range req.Parts {
		userMsg.Parts = append(userMsg.Parts, MockPart{Type: p.Type, Text: p.Text})
	}
	session.Messages = append(session.Messages, userMsg)

	// Create assistant response
	assistantMsg := MockMessage{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()+1),
		SessionID: sessionID,
		Role:      "assistant",
		Parts: []MockPart{
			{Type: "text", Text: "This is a mock response from the test server."},
		},
		CreatedAt: time.Now(),
	}
	session.Messages = append(session.Messages, assistantMsg)
	m.sessionsMu.Unlock()

	// Publish events
	m.publishEvent("message.updated", map[string]interface{}{
		"info": MockMessageInfo{
			ID:        assistantMsg.ID,
			SessionID: sessionID,
			Role:      "assistant",
			ModelID:   "test-model",
		},
	})

	// Respond with streaming-like format
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"info": MockMessageInfo{
			ID:         assistantMsg.ID,
			SessionID:  sessionID,
			Role:       "assistant",
			ModelID:    "test-model",
			ProviderID: "test-provider",
		},
		"parts": assistantMsg.Parts,
	}
	json.NewEncoder(w).Encode(response)
}

func (m *MockOpenCodeServer) getMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	m.sessionsMu.RLock()
	session, ok := m.sessions[sessionID]
	m.sessionsMu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session.Messages)
}

func (m *MockOpenCodeServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	eventCh := make(chan string, 10)
	m.eventSubsMu.Lock()
	m.eventSubs = append(m.eventSubs, eventCh)
	m.eventSubsMu.Unlock()

	defer func() {
		m.eventSubsMu.Lock()
		for i, ch := range m.eventSubs {
			if ch == eventCh {
				m.eventSubs = append(m.eventSubs[:i], m.eventSubs[i+1:]...)
				break
			}
		}
		m.eventSubsMu.Unlock()
		close(eventCh)
	}()

	// Send initial heartbeat
	fmt.Fprintf(w, ": heartbeat\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventCh:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", event)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (m *MockOpenCodeServer) publishEvent(eventType string, data interface{}) {
	eventData, _ := json.Marshal(map[string]interface{}{
		"type":       eventType,
		"properties": data,
	})

	m.eventSubsMu.Lock()
	for _, ch := range m.eventSubs {
		select {
		case ch <- string(eventData):
		default:
		}
	}
	m.eventSubsMu.Unlock()
}

func (m *MockOpenCodeServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model":    "test-provider/test-model",
		"provider": "test-provider",
	})
}

func (m *MockOpenCodeServer) handleProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": []map[string]interface{}{
			{
				"id":   "test-provider",
				"name": "Test Provider",
				"models": []map[string]interface{}{
					{"id": "test-model", "name": "Test Model"},
				},
			},
		},
	})
}

// ==================== Tests ====================

func TestResolveConfig(t *testing.T) {
	t.Run("should require URL", func(t *testing.T) {
		os.Unsetenv("OPENCODE_SERVER_URL")
		_, err := resolveConfig([]string{})
		if err == nil {
			t.Error("Expected error for missing URL")
		}
	})

	t.Run("should use URL from flag", func(t *testing.T) {
		cfg, err := resolveConfig([]string{"--url", "http://localhost:8080"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.URL != "http://localhost:8080" {
			t.Errorf("Expected URL http://localhost:8080, got %s", cfg.URL)
		}
	})

	t.Run("should use URL from environment", func(t *testing.T) {
		os.Setenv("OPENCODE_SERVER_URL", "http://env-server:9000")
		defer os.Unsetenv("OPENCODE_SERVER_URL")

		cfg, err := resolveConfig([]string{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.URL != "http://env-server:9000" {
			t.Errorf("Expected URL from env, got %s", cfg.URL)
		}
	})

	t.Run("should parse all flags", func(t *testing.T) {
		cfg, err := resolveConfig([]string{
			"--url", "http://localhost:8080",
			"--model", "gpt-4",
			"--provider", "openai",
			"--agent", "CodeAgent",
			"--quiet",
			"--verbose",
			"--json",
			"--no-color",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.Model != "gpt-4" {
			t.Errorf("Expected model gpt-4, got %s", cfg.Model)
		}
		if cfg.Provider != "openai" {
			t.Errorf("Expected provider openai, got %s", cfg.Provider)
		}
		if cfg.Agent != "CodeAgent" {
			t.Errorf("Expected agent CodeAgent, got %s", cfg.Agent)
		}
		if !cfg.Quiet {
			t.Error("Expected quiet to be true")
		}
		if !cfg.Verbose {
			t.Error("Expected verbose to be true")
		}
		if !cfg.JSON {
			t.Error("Expected json to be true")
		}
		if !cfg.NoColor {
			t.Error("Expected no-color to be true")
		}
	})
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected commandResult
	}{
		{"/exit", commandResult{Type: "exit"}},
		{"/quit", commandResult{Type: "exit"}},
		{"/help", commandResult{Type: "help"}},
		{"/model gpt-4", commandResult{Type: "set", Key: "model", Val: "gpt-4"}},
		{"/provider openai", commandResult{Type: "set", Key: "provider", Val: "openai"}},
		{"/agent CodeAgent", commandResult{Type: "set", Key: "agent", Val: "CodeAgent"}},
		{"/unknown", commandResult{Type: "unknown", Val: "/unknown"}},
		{"", commandResult{Type: "unknown"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseCommand(tc.input)
			if result.Type != tc.expected.Type {
				t.Errorf("Expected type %s, got %s", tc.expected.Type, result.Type)
			}
			if result.Key != tc.expected.Key {
				t.Errorf("Expected key %s, got %s", tc.expected.Key, result.Key)
			}
			if result.Val != tc.expected.Val {
				t.Errorf("Expected val %s, got %s", tc.expected.Val, result.Val)
			}
		})
	}
}

func TestApplyCommand(t *testing.T) {
	cfg := &ResolvedConfig{
		CliOptions: CliOptions{
			Model:    "initial-model",
			Provider: "initial-provider",
			Agent:    "initial-agent",
		},
	}

	applyCommand(cfg, commandResult{Type: "set", Key: "model", Val: "new-model"})
	if cfg.Model != "new-model" {
		t.Errorf("Expected model new-model, got %s", cfg.Model)
	}

	applyCommand(cfg, commandResult{Type: "set", Key: "provider", Val: "new-provider"})
	if cfg.Provider != "new-provider" {
		t.Errorf("Expected provider new-provider, got %s", cfg.Provider)
	}

	applyCommand(cfg, commandResult{Type: "set", Key: "agent", Val: "new-agent"})
	if cfg.Agent != "new-agent" {
		t.Errorf("Expected agent new-agent, got %s", cfg.Agent)
	}
}

func TestRenderer(t *testing.T) {
	t.Run("should create renderer with config", func(t *testing.T) {
		cfg := ResolvedConfig{
			CliOptions: CliOptions{
				NoColor: true,
				Quiet:   true,
				JSON:    true,
				Verbose: true,
			},
		}
		renderer := NewRenderer(cfg)
		if renderer == nil {
			t.Error("Expected renderer to be created")
		}
	})

	t.Run("should track rendered messages", func(t *testing.T) {
		cfg := ResolvedConfig{}
		renderer := NewRenderer(cfg)

		if renderer.WasRendered("msg-1") {
			t.Error("Message should not be rendered yet")
		}

		renderer.MarkRendered("msg-1")

		if !renderer.WasRendered("msg-1") {
			t.Error("Message should be marked as rendered")
		}
	})

	t.Run("should output JSON format", func(t *testing.T) {
		cfg := ResolvedConfig{
			CliOptions: CliOptions{
				JSON: true,
			},
		}
		renderer := NewRenderer(cfg)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		renderer.User("test input")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)

		var output map[string]string
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("Failed to parse JSON output: %v", err)
		}
		if output["type"] != "user" {
			t.Errorf("Expected type user, got %s", output["type"])
		}
		if output["text"] != "test input" {
			t.Errorf("Expected text 'test input', got %s", output["text"])
		}
	})
}

func TestSessionState(t *testing.T) {
	t.Run("should persist and load session state", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := filepath.Join(tmpDir, "state.json")

		cfg := ResolvedConfig{
			SessionFile: stateFile,
		}

		entry := SessionStateEntry{
			SessionID: "test-session-123",
			Model:     "gpt-4",
			Provider:  "openai",
			Agent:     "CodeAgent",
			UpdatedAt: time.Now().UnixMilli(),
		}

		err := persistSessionState(cfg, entry)
		if err != nil {
			t.Fatalf("Failed to persist state: %v", err)
		}

		loaded := loadSessionState(cfg)
		if loaded == nil {
			t.Fatal("Expected to load session state")
		}
		if loaded.SessionID != entry.SessionID {
			t.Errorf("Expected session ID %s, got %s", entry.SessionID, loaded.SessionID)
		}
		if loaded.Model != entry.Model {
			t.Errorf("Expected model %s, got %s", entry.Model, loaded.Model)
		}
	})

	t.Run("should return nil for non-existent state file", func(t *testing.T) {
		cfg := ResolvedConfig{
			SessionFile: "/non/existent/path/state.json",
		}

		loaded := loadSessionState(cfg)
		if loaded != nil {
			t.Error("Expected nil for non-existent state file")
		}
	})
}

func TestSimpleClientIntegration(t *testing.T) {
	// Start mock server
	mockServer := NewMockOpenCodeServer()
	defer mockServer.Close()

	cfg := ResolvedConfig{
		CliOptions: CliOptions{
			URL:       mockServer.URL(),
			Directory: t.TempDir(),
		},
		SessionFile: filepath.Join(t.TempDir(), "state.json"),
	}

	renderer := NewRenderer(cfg)

	t.Run("should create client and session", func(t *testing.T) {
		client, err := newSimpleClient(cfg, renderer)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		if client.sessionID == "" {
			t.Error("Expected session ID to be set")
		}
	})

	t.Run("should send prompt and receive response", func(t *testing.T) {
		client, err := newSimpleClient(cfg, renderer)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()
		resp, err := client.SendPrompt(ctx, "Hello, world!", cfg)
		if err != nil {
			t.Fatalf("Failed to send prompt: %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response")
		}
		if resp.Info.Role != "assistant" {
			t.Errorf("Expected assistant role, got %s", resp.Info.Role)
		}
	})

	t.Run("should reuse cached session", func(t *testing.T) {
		// Create first client
		client1, err := newSimpleClient(cfg, renderer)
		if err != nil {
			t.Fatalf("Failed to create first client: %v", err)
		}
		sessionID1 := client1.sessionID
		client1.Close()

		// Create second client - should reuse session from state
		client2, err := newSimpleClient(cfg, renderer)
		if err != nil {
			t.Fatalf("Failed to create second client: %v", err)
		}
		defer client2.Close()

		if client2.sessionID != sessionID1 {
			t.Errorf("Expected session ID %s, got %s", sessionID1, client2.sessionID)
		}
	})
}

func TestConfigFileLoading(t *testing.T) {
	t.Run("should load config from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".opencode")
		os.MkdirAll(configDir, 0755)

		configContent := `{
			"url": "http://config-file-url:8080",
			"model": "config-model",
			"provider": "config-provider"
		}`
		configPath := filepath.Join(configDir, "simple-cli.json")
		os.WriteFile(configPath, []byte(configContent), 0644)

		// Change to temp dir
		oldDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(oldDir)

		fileOpts, err := loadConfigFile(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load config file: %v", err)
		}

		if fileOpts.Model != "config-model" {
			t.Errorf("Expected model config-model, got %s", fileOpts.Model)
		}
		if fileOpts.Provider != "config-provider" {
			t.Errorf("Expected provider config-provider, got %s", fileOpts.Provider)
		}
	})
}

func TestBuildPrompt(t *testing.T) {
	prompt := buildPrompt()
	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}
	if !strings.HasSuffix(prompt, "> ") {
		t.Errorf("Expected prompt to end with '> ', got %s", prompt)
	}
}

// Integration test with full workflow
func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := NewMockOpenCodeServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	cfg := ResolvedConfig{
		CliOptions: CliOptions{
			URL:       mockServer.URL(),
			Directory: tmpDir,
			Verbose:   true,
		},
		SessionFile: filepath.Join(tmpDir, "state.json"),
	}

	renderer := NewRenderer(cfg)

	// Create client
	client, err := newSimpleClient(cfg, renderer)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send multiple messages
	for i := 0; i < 3; i++ {
		ctx := context.Background()
		resp, err := client.SendPrompt(ctx, fmt.Sprintf("Message %d", i+1), cfg)
		if err != nil {
			t.Fatalf("Failed to send message %d: %v", i+1, err)
		}
		if resp == nil {
			t.Fatalf("Expected response for message %d", i+1)
		}
	}
}

// Benchmark tests
func BenchmarkParseCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseCommand("/model gpt-4")
	}
}

func BenchmarkRenderer(b *testing.B) {
	cfg := ResolvedConfig{
		CliOptions: CliOptions{
			NoColor: true,
			Quiet:   true,
		},
	}
	renderer := NewRenderer(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.MarkRendered(fmt.Sprintf("msg-%d", i))
	}
}

// Test concurrent access
func TestConcurrentAccess(t *testing.T) {
	mockServer := NewMockOpenCodeServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	cfg := ResolvedConfig{
		CliOptions: CliOptions{
			URL:       mockServer.URL(),
			Directory: tmpDir,
		},
		SessionFile: filepath.Join(tmpDir, "state.json"),
	}

	renderer := NewRenderer(cfg)
	client, err := newSimpleClient(cfg, renderer)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send concurrent messages
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ctx := context.Background()
			_, err := client.SendPrompt(ctx, fmt.Sprintf("Concurrent message %d", n), cfg)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent send error: %v", err)
	}
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	t.Run("should handle connection refused", func(t *testing.T) {
		// Find an unused port
		listener, _ := net.Listen("tcp", "127.0.0.1:0")
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		cfg := ResolvedConfig{
			CliOptions: CliOptions{
				URL:       fmt.Sprintf("http://127.0.0.1:%d", port),
				Directory: t.TempDir(),
			},
			SessionFile: filepath.Join(t.TempDir(), "state.json"),
		}

		renderer := NewRenderer(cfg)
		_, err := newSimpleClient(cfg, renderer)
		if err == nil {
			t.Error("Expected error for connection refused")
		}
	})
}

// Test help text
func TestHelpText(t *testing.T) {
	if helpText == "" {
		t.Error("Expected non-empty help text")
	}
	if !strings.Contains(helpText, "/help") {
		t.Error("Help text should contain /help command")
	}
	if !strings.Contains(helpText, "/exit") {
		t.Error("Help text should contain /exit command")
	}
}
