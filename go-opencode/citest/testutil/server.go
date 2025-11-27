package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/server"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// TestServer wraps a server instance for testing
type TestServer struct {
	Server      *server.Server
	BaseURL     string
	Config      *types.Config
	Storage     *storage.Storage
	ProviderReg *provider.Registry
	ToolReg     *tool.Registry
	TempDir     string
	WorkDir     string
	port        int
	mockLLM     *MockLLMServer // MockLLM server if using mockllm provider
}

// TestServerOption configures TestServer
type TestServerOption func(*testServerConfig)

type testServerConfig struct {
	workDir string
	envFile string
}

// WithWorkDir sets the working directory
func WithWorkDir(dir string) TestServerOption {
	return func(c *testServerConfig) {
		c.workDir = dir
	}
}

// WithEnvFile sets the .env file to load
func WithEnvFile(path string) TestServerOption {
	return func(c *testServerConfig) {
		c.envFile = path
	}
}

// StartTestServer creates and starts a test server
func StartTestServer(opts ...TestServerOption) (*TestServer, error) {
	cfg := &testServerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Load environment variables
	if cfg.envFile != "" {
		_ = godotenv.Load(cfg.envFile)
	} else {
		// Try default locations
		_ = godotenv.Load("../../.env")
		_ = godotenv.Load("../.env")
		_ = godotenv.Load(".env")
	}

	// Create temp directory for test data
	tempDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	workDir := cfg.workDir
	if workDir == "" {
		workDir = tempDir
	}

	// Check which provider to use
	testProvider := os.Getenv("TEST_PROVIDER")
	if testProvider == "" {
		testProvider = "openai" // Default to OpenAI
	}

	var mockLLM *MockLLMServer

	switch testProvider {
	case "mockllm":
		// Start MockLLM server with config from mockllm.yaml
		configDir := getMockLLMConfigDir()
		var err error
		if configDir != "" {
			mockLLM, err = NewMockLLMServerFromDir(configDir)
			if err != nil {
				// Fall back to default config if loading fails
				mockLLM = NewMockLLMServer()
			}
		} else {
			mockLLM = NewMockLLMServer()
		}

		// Set env vars for config interpolation
		os.Setenv("OPENAI_BASE_URL", mockLLM.URL())
		os.Setenv("OPENAI_API_KEY", "mock-api-key")
		os.Setenv("OPENAI_MODEL_ID", "gpt-4o-mini")
		if os.Getenv("OPENCODE_MODEL") == "" {
			os.Setenv("OPENCODE_MODEL", "openai/gpt-4o-mini")
		}

	case "ark":
		// Set OPENCODE_MODEL for ARK provider (only if not already set)
		if os.Getenv("OPENCODE_MODEL") == "" {
			arkModelID := os.Getenv("ARK_MODEL_ID")
			if arkModelID != "" {
				os.Setenv("OPENCODE_MODEL", "ark/"+arkModelID)
			}
		}

	case "openai":
		// Set OPENCODE_MODEL for OpenAI provider (only if not already set)
		if os.Getenv("OPENCODE_MODEL") == "" {
			openaiModelID := os.Getenv("OPENAI_MODEL_ID")
			if openaiModelID == "" {
				openaiModelID = "gpt-4o-mini"
				os.Setenv("OPENAI_MODEL_ID", openaiModelID)
			}
			os.Setenv("OPENCODE_MODEL", "openai/"+openaiModelID)
		}

	case "anthropic":
		// Set OPENCODE_MODEL for Anthropic provider (only if not already set)
		if os.Getenv("OPENCODE_MODEL") == "" {
			anthropicModelID := os.Getenv("ANTHROPIC_MODEL_ID")
			if anthropicModelID == "" {
				anthropicModelID = "claude-sonnet-4-20250514"
				os.Setenv("ANTHROPIC_MODEL_ID", anthropicModelID)
			}
			os.Setenv("OPENCODE_MODEL", "anthropic/"+anthropicModelID)
		}
	}

	// Set OPENCODE_CONFIG to point to the test config file
	// This ensures config.Load() finds our test configuration
	configPath := getTestConfigPath()
	if configPath != "" {
		os.Setenv("OPENCODE_CONFIG", configPath)
	}

	// Load config from opencode.json using config.Load()
	// The config file uses {env:VAR} interpolation for provider settings
	appConfig, err := config.Load(workDir)
	if err != nil {
		if mockLLM != nil {
			mockLLM.Close()
		}
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Helper for cleanup on error
	cleanup := func() {
		if mockLLM != nil {
			mockLLM.Close()
		}
		os.RemoveAll(tempDir)
	}

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	ctx := context.Background()

	// Initialize storage
	storagePath := filepath.Join(tempDir, "storage")
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}
	store := storage.New(storagePath)

	// Initialize providers
	providerReg, err := provider.InitializeProviders(ctx, appConfig)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Initialize tools
	toolReg := tool.DefaultRegistry(workDir)

	// Configure server
	serverConfig := server.DefaultConfig()
	serverConfig.Port = port
	serverConfig.Directory = workDir

	// Create server
	srv := server.New(serverConfig, appConfig, store, providerReg, toolReg)

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Wait for server to be ready
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if err := waitForServer(baseURL, 10*time.Second); err != nil {
		srv.Shutdown(ctx)
		cleanup()
		return nil, fmt.Errorf("server failed to start: %w", err)
	}

	return &TestServer{
		Server:      srv,
		BaseURL:     baseURL,
		Config:      appConfig,
		Storage:     store,
		ProviderReg: providerReg,
		ToolReg:     toolReg,
		TempDir:     tempDir,
		WorkDir:     workDir,
		port:        port,
		mockLLM:     mockLLM,
	}, nil
}

// Stop shuts down the test server and cleans up
func (ts *TestServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if ts.Server != nil {
		if err := ts.Server.Shutdown(ctx); err != nil {
			return err
		}
	}

	// Close MockLLM server if running
	if ts.mockLLM != nil {
		ts.mockLLM.Close()
	}

	if ts.TempDir != "" {
		os.RemoveAll(ts.TempDir)
	}

	return nil
}

// Client returns a new test client for this server
func (ts *TestServer) Client() *TestClient {
	return NewTestClient(ts.BaseURL)
}

// SSEClient returns a new SSE client for this server
func (ts *TestServer) SSEClient() *SSEClient {
	return NewSSEClient(ts.BaseURL)
}

// getTestConfigPath returns the path to the test configuration file.
// It searches for citest/config/opencode.json relative to the current directory.
func getTestConfigPath() string {
	// Try to find the config file relative to current directory
	// This handles running tests from different directories
	candidates := []string{
		"citest/config/opencode.json",
		"../citest/config/opencode.json",
		"../../citest/config/opencode.json",
		"../../../citest/config/opencode.json",
	}

	for _, candidate := range candidates {
		if absPath, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Also check if OPENCODE_CONFIG is already set
	if existing := os.Getenv("OPENCODE_CONFIG"); existing != "" {
		return existing
	}

	return ""
}

// getMockLLMConfigDir returns the directory containing mockllm.yaml config.
func getMockLLMConfigDir() string {
	candidates := []string{
		"citest/config",
		"../citest/config",
		"../../citest/config",
		"../../../citest/config",
	}

	for _, candidate := range candidates {
		if absPath, err := filepath.Abs(candidate); err == nil {
			mockllmPath := filepath.Join(absPath, "mockllm.yaml")
			if _, err := os.Stat(mockllmPath); err == nil {
				return absPath
			}
		}
	}

	return ""
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

// waitForServer waits for the server to be ready
func waitForServer(baseURL string, timeout time.Duration) error {
	client := NewTestClient(baseURL)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(context.Background(), "/config")
		if err == nil && resp.IsSuccess() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("server not ready after %v", timeout)
}
