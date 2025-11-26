package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

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

	// Build config
	appConfig := buildTestConfig()

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	ctx := context.Background()

	// Initialize storage
	storagePath := filepath.Join(tempDir, "storage")
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}
	store := storage.New(storagePath)

	// Initialize providers
	providerReg, err := provider.InitializeProviders(ctx, appConfig)
	if err != nil {
		os.RemoveAll(tempDir)
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
		os.RemoveAll(tempDir)
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

// buildTestConfig creates a test configuration with ARK provider
func buildTestConfig() *types.Config {
	apiKey := os.Getenv("ARK_API_KEY")
	baseURL := os.Getenv("ARK_BASE_URL")
	modelID := os.Getenv("ARK_MODEL_ID")

	return &types.Config{
		Model: fmt.Sprintf("ark/%s", modelID),
		Provider: map[string]types.ProviderConfig{
			"ark": {
				APIKey:  apiKey,
				BaseURL: baseURL,
				Model:   modelID,
			},
		},
		Permission: &types.PermissionConfig{
			Edit: "allow",
			Bash: "allow",
		},
	}
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
		resp, err := client.Get(context.Background(), "/config", nil)
		if err == nil && resp.IsSuccess() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("server not ready after %v", timeout)
}
