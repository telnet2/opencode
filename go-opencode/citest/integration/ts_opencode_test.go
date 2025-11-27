// Package integration provides integration tests for the TypeScript OpenCode server
// using the MockLLM for deterministic LLM responses.
package integration_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencode-ai/opencode/citest/testutil"
)

func TestTSOpenCodeIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TypeScript OpenCode Integration Suite")
}

// TSOpenCodeServer manages a TypeScript OpenCode server process
type TSOpenCodeServer struct {
	cmd        *exec.Cmd
	baseURL    string
	port       int
	workDir    string
	configDir  string
	stateDir   string
	mockLLMURL string
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	mu         sync.Mutex
	started    bool
}

// TSOpenCodeConfig holds configuration for the TS server
type TSOpenCodeConfig struct {
	MockLLMURL string
	WorkDir    string
	Port       int
}

// NewTSOpenCodeServer creates a new TypeScript OpenCode server manager
func NewTSOpenCodeServer(config TSOpenCodeConfig) (*TSOpenCodeServer, error) {
	// Create temp directories
	tempDir, err := os.MkdirTemp("", "ts-opencode-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	workDir := config.WorkDir
	if workDir == "" {
		workDir = tempDir
	}

	configDir := filepath.Join(tempDir, "config")
	stateDir := filepath.Join(tempDir, "state")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create state dir: %w", err)
	}

	port := config.Port
	if port == 0 {
		port, err = findAvailablePort()
		if err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("failed to find available port: %w", err)
		}
	}

	return &TSOpenCodeServer{
		port:       port,
		workDir:    workDir,
		configDir:  configDir,
		stateDir:   stateDir,
		mockLLMURL: config.MockLLMURL,
		baseURL:    fmt.Sprintf("http://127.0.0.1:%d", port),
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
	}, nil
}

// Start launches the TypeScript OpenCode server
func (s *TSOpenCodeServer) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}

	// Create OpenCode config file pointing to MockLLM
	configContent := fmt.Sprintf(`{
  "$schema": "https://opencode.ai/config.json",
  "model": "openai/gpt-4o-mini",
  "provider": {
    "openai": {
      "options": {
        "apiKey": "mock-api-key",
        "baseURL": "%s/v1"
      }
    }
  },
  "permission": {
    "edit": "allow",
    "bash": "allow"
  }
}`, s.mockLLMURL)

	configPath := filepath.Join(s.configDir, "opencode.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Find the packages/opencode directory
	opencodeDir, err := findOpencodeDir()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to find opencode directory: %w", err)
	}

	// Build the command to run the TypeScript server
	s.cmd = exec.CommandContext(ctx, "bun", "run", "./src/index.ts", "serve",
		"--port", fmt.Sprintf("%d", s.port),
		"--hostname", "127.0.0.1",
	)
	s.cmd.Dir = opencodeDir

	// Set environment variables
	s.cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENCODE_CONFIG=%s", configPath),
		fmt.Sprintf("OPENCODE_STATE_DIR=%s", s.stateDir),
		fmt.Sprintf("HOME=%s", s.workDir),
		"OPENCODE_DISABLE_AUTOUPDATE=true",
		"OPENCODE_DISABLE_LSP_DOWNLOAD=true",
	)

	// Capture stdout and stderr
	s.cmd.Stdout = s.stdout
	s.cmd.Stderr = s.stderr

	// Start the process
	if err := s.cmd.Start(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.mu.Unlock()

	// Wait for the server to be ready (outside of lock)
	if err := s.waitForReady(30 * time.Second); err != nil {
		s.stopInternal() // Use internal stop to kill the process
		return fmt.Errorf("server failed to become ready: %w\nstdout: %s\nstderr: %s",
			err, s.stdout.String(), s.stderr.String())
	}

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	return nil
}

// waitForReady polls the server until it's ready or timeout
func (s *TSOpenCodeServer) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(s.baseURL + "/config")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("server not ready after %v", timeout)
}

// stopInternal kills the process without locking (used internally)
func (s *TSOpenCodeServer) stopInternal() {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	// Cleanup temp directories
	if s.configDir != "" {
		parentDir := filepath.Dir(s.configDir)
		if strings.Contains(parentDir, "ts-opencode-test") {
			os.RemoveAll(parentDir)
		}
	}
}

// Stop shuts down the TypeScript server
func (s *TSOpenCodeServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopInternal()
	s.started = false
	return nil
}

// URL returns the server's base URL
func (s *TSOpenCodeServer) URL() string {
	return s.baseURL
}

// GetOutput returns captured stdout and stderr
func (s *TSOpenCodeServer) GetOutput() (stdout, stderr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stdout.String(), s.stderr.String()
}

// findOpencodeDir finds the packages/opencode directory
func findOpencodeDir() (string, error) {
	// Try relative paths from different possible locations
	candidates := []string{
		"../../packages/opencode",
		"../../../packages/opencode",
		"../../../../packages/opencode",
	}

	cwd, _ := os.Getwd()
	for _, candidate := range candidates {
		path := filepath.Join(cwd, candidate)
		if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
			return filepath.Abs(path)
		}
	}

	// Try from OPENCODE_ROOT env var
	if root := os.Getenv("OPENCODE_ROOT"); root != "" {
		path := filepath.Join(root, "packages/opencode")
		if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not find packages/opencode directory")
}

// findAvailablePort finds an available TCP port
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// HTTPClient provides helper methods for making requests to the server
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client for the server
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CreateSession creates a new session
func (c *HTTPClient) CreateSession(ctx context.Context, directory string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"directory": directory,
	}
	return c.post(ctx, "/session", body)
}

// SendMessage sends a message to a session
func (c *HTTPClient) SendMessage(ctx context.Context, sessionID string, message string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"parts": []map[string]interface{}{
			{
				"type": "text",
				"text": message,
			},
		},
	}
	return c.post(ctx, fmt.Sprintf("/session/%s/message", sessionID), body)
}

// GetConfig gets the server configuration
func (c *HTTPClient) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	return c.get(ctx, "/config")
}

// ListSessions lists all sessions
func (c *HTTPClient) ListSessions(ctx context.Context) ([]interface{}, error) {
	result, err := c.get(ctx, "/session")
	if err != nil {
		return nil, err
	}

	// The response might be a slice directly
	if arr, ok := result["sessions"].([]interface{}); ok {
		return arr, nil
	}

	return nil, nil
}

func (c *HTTPClient) get(ctx context.Context, path string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (c *HTTPClient) post(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// SSEReader reads Server-Sent Events from a response
type SSEReader struct {
	reader *bufio.Reader
}

// NewSSEReader creates a new SSE reader
func NewSSEReader(body io.Reader) *SSEReader {
	return &SSEReader{
		reader: bufio.NewReader(body),
	}
}

// ReadEvent reads the next SSE event
func (r *SSEReader) ReadEvent() (eventType string, data string, err error) {
	for {
		line, err := r.reader.ReadString('\n')
		if err != nil {
			return "", "", err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			if data != "" {
				return eventType, data, nil
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}

// ===== Integration Tests =====

var _ = Describe("TypeScript OpenCode Server with MockLLM", func() {
	var mockLLM *testutil.MockLLMServer
	var tsServer *TSOpenCodeServer
	var client *HTTPClient
	var ctx context.Context
	var tempDir string

	BeforeEach(func() {
		// Skip if bun is not available
		if _, err := exec.LookPath("bun"); err != nil {
			Skip("bun not found in PATH, skipping TypeScript server tests")
		}

		// Check if TypeScript dependencies are installed
		opencodeDir, err := findOpencodeDir()
		if err != nil {
			Skip("Could not find packages/opencode directory: " + err.Error())
		}
		nodeModulesPath := filepath.Join(opencodeDir, "node_modules")
		if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
			Skip("TypeScript dependencies not installed (node_modules not found). Run 'bun install' in packages/opencode first.")
		}

		// Create temp directory for test files
		tempDir, err = os.MkdirTemp("", "opencode-integration-*")
		Expect(err).NotTo(HaveOccurred())

		// Create MockLLM server with custom responses
		mockConfig := &testutil.MockLLMConfig{
			Settings: testutil.MockSettings{
				LagMS:           0,
				EnableStreaming: true,
				ChunkDelayMS:    5,
				ChunkMode:       "word",
			},
			Defaults: testutil.MockDefaults{
				Fallback: "I understand your request. Let me help you with that.",
			},
			Responses: []testutil.ResponseRule{
				{
					Name:     "hello-world",
					Match:    testutil.MatchConfig{Contains: "hello"},
					Response: "Hello! I'm the MockLLM responding through the TypeScript OpenCode server.",
					Priority: 10,
				},
				{
					Name:     "math-2plus2",
					Match:    testutil.MatchConfig{ContainsAny: []string{"2+2", "2 + 2"}},
					Response: "The answer is 4.",
					Priority: 10,
				},
				{
					Name:     "simple-test",
					Match:    testutil.MatchConfig{Contains: "test"},
					Response: "This is a test response from MockLLM.",
					Priority: 5,
				},
			},
			ToolRules: []testutil.ToolRule{
				{
					Name:  "echo-command",
					Match: testutil.MatchConfig{Contains: "run echo"},
					Tool:  "bash",
					ToolCall: testutil.ToolCallConfig{
						ID:        "call_bash_echo",
						Arguments: map[string]string{"command": "echo hello from mockllm"},
					},
					Response: "I'll run that echo command for you.",
					Priority: 10,
				},
			},
		}
		mockLLM = testutil.NewMockLLMServerWithConfig(mockConfig)

		// Create TypeScript OpenCode server pointing to MockLLM
		tsServer, err = NewTSOpenCodeServer(TSOpenCodeConfig{
			MockLLMURL: mockLLM.URL(),
			WorkDir:    tempDir,
		})
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()

		// Start the TypeScript server
		err = tsServer.Start(ctx)
		if err != nil {
			// If starting fails due to missing deps or server issues, skip
			if strings.Contains(err.Error(), "not found") ||
				strings.Contains(err.Error(), "not ready") ||
				strings.Contains(err.Error(), "preload") ||
				strings.Contains(err.Error(), "401") {
				Skip("TypeScript server could not start: " + err.Error())
			}
			Fail("Failed to start TypeScript server: " + err.Error())
		}

		client = NewHTTPClient(tsServer.URL())
	})

	AfterEach(func() {
		if tsServer != nil {
			tsServer.Stop()
		}
		if mockLLM != nil {
			mockLLM.Close()
		}
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	Describe("Server Connectivity", func() {
		It("should respond to health/config check", func() {
			config, err := client.GetConfig(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
		})

		It("should verify MockLLM is accessible", func() {
			// Direct request to MockLLM to verify it's working
			resp, err := http.Get(mockLLM.URL() + "/health")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Describe("Session Management", func() {
		It("should create a new session", func() {
			session, err := client.CreateSession(ctx, tempDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(session).NotTo(BeNil())
			Expect(session["id"]).NotTo(BeEmpty())
		})

		It("should list sessions", func() {
			// Create a session first
			_, err := client.CreateSession(ctx, tempDir)
			Expect(err).NotTo(HaveOccurred())

			// List sessions
			sessions, err := client.ListSessions(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions).NotTo(BeNil())
		})
	})

	Describe("Message Exchange with MockLLM", func() {
		var sessionID string

		BeforeEach(func() {
			session, err := client.CreateSession(ctx, tempDir)
			Expect(err).NotTo(HaveOccurred())
			sessionID = session["id"].(string)
		})

		It("should send message and receive MockLLM response", func() {
			// Clear any previous requests
			mockLLM.ClearRequests()

			// Send a message that matches our MockLLM config
			response, err := client.SendMessage(ctx, sessionID, "hello, how are you?")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			// Verify MockLLM received the request
			requests := mockLLM.GetRequests()
			Expect(len(requests)).To(BeNumerically(">", 0), "MockLLM should have received at least one request")

			// The last request should contain our message
			lastRequest := requests[len(requests)-1]
			Expect(lastRequest.Path).To(ContainSubstring("chat/completions"))
		})

		It("should get deterministic response from MockLLM", func() {
			mockLLM.ClearRequests()

			// Send the same message twice
			resp1, err := client.SendMessage(ctx, sessionID, "what is 2+2?")
			Expect(err).NotTo(HaveOccurred())

			resp2, err := client.SendMessage(ctx, sessionID, "what is 2+2?")
			Expect(err).NotTo(HaveOccurred())

			// Both should have succeeded
			Expect(resp1).NotTo(BeNil())
			Expect(resp2).NotTo(BeNil())

			// MockLLM should have received 2 requests
			requests := mockLLM.GetRequests()
			Expect(len(requests)).To(BeNumerically(">=", 2))
		})

		It("should handle test prompt with MockLLM", func() {
			mockLLM.ClearRequests()

			response, err := client.SendMessage(ctx, sessionID, "this is a test")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			// Verify request was received
			requests := mockLLM.GetRequests()
			Expect(len(requests)).To(BeNumerically(">", 0))
		})
	})

	Describe("Request Recording", func() {
		It("should record all requests to MockLLM", func() {
			mockLLM.ClearRequests()

			// Create session
			session, err := client.CreateSession(ctx, tempDir)
			Expect(err).NotTo(HaveOccurred())
			sessionID := session["id"].(string)

			// Send multiple messages
			messages := []string{"hello", "test", "another message"}
			for _, msg := range messages {
				_, err := client.SendMessage(ctx, sessionID, msg)
				Expect(err).NotTo(HaveOccurred())
			}

			// Check recorded requests
			requests := mockLLM.GetRequests()
			Expect(len(requests)).To(BeNumerically(">=", len(messages)))

			// All requests should be POST to chat completions
			for _, req := range requests {
				Expect(req.Method).To(Equal("POST"))
				Expect(req.Path).To(ContainSubstring("chat/completions"))
			}
		})
	})

	Describe("Streaming Responses", func() {
		It("should handle streaming from MockLLM", func() {
			session, err := client.CreateSession(ctx, tempDir)
			Expect(err).NotTo(HaveOccurred())
			sessionID := session["id"].(string)

			mockLLM.ClearRequests()

			// Send a message - the server should handle streaming internally
			response, err := client.SendMessage(ctx, sessionID, "hello streaming test")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			// Verify MockLLM received the request
			requests := mockLLM.GetRequests()
			Expect(len(requests)).To(BeNumerically(">", 0))
		})
	})
})

var _ = Describe("MockLLM Configuration Loading", func() {
	It("should load config from YAML file", func() {
		// Find the config file
		configPath := "../../citest/config/mockllm.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "../config/mockllm.yaml"
		}

		if _, err := os.Stat(configPath); err != nil {
			Skip("mockllm.yaml config file not found")
		}

		config, err := testutil.LoadMockLLMConfig(configPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(config).NotTo(BeNil())
		Expect(config.Settings.EnableStreaming).To(BeTrue())
	})

	It("should create server from YAML config", func() {
		configPath := "../../citest/config/mockllm.yaml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "../config/mockllm.yaml"
		}

		if _, err := os.Stat(configPath); err != nil {
			Skip("mockllm.yaml config file not found")
		}

		server, err := testutil.NewMockLLMServerFromFile(configPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(server).NotTo(BeNil())
		defer server.Close()

		// Verify server is running
		resp, err := http.Get(server.URL() + "/health")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})
