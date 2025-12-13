package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Client manages connections to language servers.
type Client struct {
	mu       sync.RWMutex
	clients  map[string]*languageClient
	servers  map[string]*ServerConfig
	workDir  string
	disabled bool
}

// languageClient wraps a connection to a language server.
type languageClient struct {
	mu        sync.Mutex
	conn      *jsonrpcConn
	cmd       *exec.Cmd
	root      string
	serverID  string
	openFiles map[string]int // URI -> version
}

// jsonrpcConn manages JSON-RPC communication.
type jsonrpcConn struct {
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	nextID   int64
	mu       sync.Mutex
	pending  map[int64]chan *JSONRPCResponse
	closed   bool
}

// NewClient creates a new LSP client manager.
func NewClient(workDir string, disabled bool) *Client {
	return &Client{
		clients:  make(map[string]*languageClient),
		servers:  builtInServers(),
		workDir:  workDir,
		disabled: disabled,
	}
}

// builtInServers returns default language server configurations.
func builtInServers() map[string]*ServerConfig {
	return map[string]*ServerConfig{
		"typescript": {
			ID:         "typescript",
			Extensions: []string{".ts", ".tsx", ".js", ".jsx"},
			Command:    []string{"typescript-language-server", "--stdio"},
		},
		"go": {
			ID:         "go",
			Extensions: []string{".go"},
			Command:    []string{"gopls"},
		},
		"python": {
			ID:         "python",
			Extensions: []string{".py"},
			Command:    []string{"pyright-langserver", "--stdio"},
		},
		"rust": {
			ID:         "rust",
			Extensions: []string{".rs"},
			Command:    []string{"rust-analyzer"},
		},
	}
}

// AddServer adds a custom server configuration.
func (c *Client) AddServer(config *ServerConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.servers[config.ID] = config
}

// GetClient returns or creates a client for the given file.
func (c *Client) GetClient(ctx context.Context, filePath string) (*languageClient, error) {
	if c.disabled {
		return nil, fmt.Errorf("LSP disabled")
	}

	ext := filepath.Ext(filePath)
	if ext == "" {
		return nil, fmt.Errorf("no extension for file: %s", filePath)
	}

	// Find server for this extension
	var serverConfig *ServerConfig
	c.mu.RLock()
	for _, cfg := range c.servers {
		for _, e := range cfg.Extensions {
			if e == ext {
				serverConfig = cfg
				break
			}
		}
		if serverConfig != nil {
			break
		}
	}
	c.mu.RUnlock()

	if serverConfig == nil {
		return nil, fmt.Errorf("no server for extension: %s", ext)
	}

	// Find project root
	root := c.findProjectRoot(filePath, serverConfig.ID)

	// Check for existing client
	clientKey := fmt.Sprintf("%s:%s", serverConfig.ID, root)

	c.mu.RLock()
	if client, ok := c.clients[clientKey]; ok {
		c.mu.RUnlock()
		return client, nil
	}
	c.mu.RUnlock()

	// Create new client
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := c.clients[clientKey]; ok {
		return client, nil
	}

	client, err := c.spawnServer(ctx, serverConfig, root)
	if err != nil {
		return nil, err
	}

	c.clients[clientKey] = client
	return client, nil
}

// spawnServer starts a language server process.
func (c *Client) spawnServer(ctx context.Context, config *ServerConfig, root string) (*languageClient, error) {
	if len(config.Command) == 0 {
		return nil, fmt.Errorf("empty command for server: %s", config.ID)
	}

	cmd := exec.CommandContext(ctx, config.Command[0], config.Command[1:]...)
	cmd.Dir = root

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	conn := &jsonrpcConn{
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		pending: make(map[int64]chan *JSONRPCResponse),
	}

	// Start reading responses
	go conn.readLoop()

	client := &languageClient{
		conn:      conn,
		cmd:       cmd,
		root:      root,
		serverID:  config.ID,
		openFiles: make(map[string]int),
	}

	// Initialize server
	if err := client.initialize(ctx, root); err != nil {
		cmd.Process.Kill()
		return nil, err
	}

	return client, nil
}

// readLoop reads responses from the server.
func (c *jsonrpcConn) readLoop() {
	for {
		resp, err := c.readMessage()
		if err != nil {
			c.mu.Lock()
			c.closed = true
			// Close all pending channels
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[int64]chan *JSONRPCResponse)
			c.mu.Unlock()
			return
		}

		if resp.ID != 0 {
			c.mu.Lock()
			if ch, ok := c.pending[resp.ID]; ok {
				ch <- resp
				delete(c.pending, resp.ID)
			}
			c.mu.Unlock()
		}
	}
}

// readMessage reads a single JSON-RPC message.
func (c *jsonrpcConn) readMessage() (*JSONRPCResponse, error) {
	// Read headers
	var contentLength int
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			lenStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(lenStr)
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("no content-length header")
	}

	// Read body
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, body); err != nil {
		return nil, err
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// call sends a request and waits for a response.
func (c *jsonrpcConn) call(ctx context.Context, method string, params any, result any) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("connection closed")
	}

	id := atomic.AddInt64(&c.nextID, 1)
	ch := make(chan *JSONRPCResponse, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	// Send request
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.writeMessage(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return err
	}

	// Wait for response
	select {
	case resp := <-ch:
		if resp == nil {
			return fmt.Errorf("connection closed")
		}
		if resp.Error != nil {
			return fmt.Errorf("LSP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		if result != nil && resp.Result != nil {
			return json.Unmarshal(resp.Result, result)
		}
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return ctx.Err()
	}
}

// notify sends a notification (no response expected).
func (c *jsonrpcConn) notify(ctx context.Context, method string, params any) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.writeMessage(req)
}

// writeMessage writes a JSON-RPC message.
func (c *jsonrpcConn) writeMessage(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := c.stdin.Write(body); err != nil {
		return err
	}
	return nil
}

// initialize sends the initialize request to the server.
func (lc *languageClient) initialize(ctx context.Context, root string) error {
	params := InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   "file://" + root,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Hover: &HoverCapability{
					ContentFormat: []string{"plaintext", "markdown"},
				},
				DocumentSymbol: &DocumentSymbolCapability{
					SymbolKind: &SymbolKindCapability{
						ValueSet: AllSymbolKinds(),
					},
				},
			},
			Workspace: WorkspaceClientCapabilities{
				Symbol: &WorkspaceSymbolCapability{
					SymbolKind: &SymbolKindCapability{
						ValueSet: AllSymbolKinds(),
					},
				},
			},
		},
	}

	var result json.RawMessage
	if err := lc.conn.call(ctx, "initialize", params, &result); err != nil {
		return err
	}

	// Send initialized notification
	return lc.conn.notify(ctx, "initialized", struct{}{})
}

// findProjectRoot finds the project root for a file.
func (c *Client) findProjectRoot(filePath, serverID string) string {
	dir := filepath.Dir(filePath)

	// Look for project markers based on server type
	markers := map[string][]string{
		"typescript": {"package.json", "tsconfig.json"},
		"go":         {"go.mod"},
		"python":     {"pyproject.toml", "setup.py", "requirements.txt"},
		"rust":       {"Cargo.toml"},
	}

	fileMarkers := markers[serverID]
	if fileMarkers == nil {
		fileMarkers = []string{".git"}
	}

	for {
		for _, marker := range fileMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return c.workDir
}

// Status returns the status of all LSP servers.
func (c *Client) Status() []ServerStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var status []ServerStatus
	for key, client := range c.clients {
		status = append(status, ServerStatus{
			ID:     client.serverID,
			Root:   client.root,
			Key:    key,
			Active: true,
		})
	}
	return status
}

// Close shuts down all language servers.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := context.Background()
	for _, client := range c.clients {
		client.conn.notify(ctx, "shutdown", nil)
		client.conn.notify(ctx, "exit", nil)
		if client.cmd.Process != nil {
			client.cmd.Process.Kill()
		}
	}

	c.clients = make(map[string]*languageClient)
	return nil
}

// IsDisabled returns whether LSP is disabled.
func (c *Client) IsDisabled() bool {
	return c.disabled
}

// SetDisabled sets the disabled state.
func (c *Client) SetDisabled(disabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disabled = disabled
}

// GetServers returns the configured servers.
func (c *Client) GetServers() map[string]*ServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	servers := make(map[string]*ServerConfig)
	for k, v := range c.servers {
		servers[k] = v
	}
	return servers
}
