# Phase 7: Advanced Features (Weeks 11-12)

## Overview

Implement advanced features including Language Server Protocol (LSP) integration, Model Context Protocol (MCP) support, multi-agent system, and plugin architecture. These features extend OpenCode's capabilities beyond basic conversation.

---

## 7.1 Language Server Protocol (LSP) Integration

### LSP Client

```go
// internal/lsp/client.go
package lsp

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"

    "github.com/sourcegraph/go-lsp"
    "github.com/sourcegraph/jsonrpc2"
)

// Client manages connections to language servers
type Client struct {
    mu       sync.RWMutex
    clients  map[string]*languageClient
    servers  map[string]*ServerConfig
    workDir  string
    disabled bool
}

// ServerConfig defines a language server configuration
type ServerConfig struct {
    ID         string   `json:"id"`
    Extensions []string `json:"extensions"` // File extensions handled
    Command    []string `json:"command"`    // Command to spawn server
}

// languageClient wraps a connection to a language server
type languageClient struct {
    conn     *jsonrpc2.Conn
    cmd      *exec.Cmd
    root     string
    serverID string
}

// NewClient creates a new LSP client manager
func NewClient(workDir string, disabled bool) *Client {
    return &Client{
        clients:  make(map[string]*languageClient),
        servers:  builtInServers(),
        workDir:  workDir,
        disabled: disabled,
    }
}

// builtInServers returns default language server configurations
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

// getClient returns or creates a client for the given file
func (c *Client) getClient(ctx context.Context, filePath string) (*languageClient, error) {
    if c.disabled {
        return nil, fmt.Errorf("LSP disabled")
    }

    ext := filepath.Ext(filePath)
    if ext == "" {
        return nil, fmt.Errorf("no extension for file: %s", filePath)
    }

    // Find server for this extension
    var serverConfig *ServerConfig
    for _, cfg := range c.servers {
        for _, e := range cfg.Extensions {
            if e == ext {
                serverConfig = cfg
                break
            }
        }
    }

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

// spawnServer starts a language server process
func (c *Client) spawnServer(ctx context.Context, config *ServerConfig, root string) (*languageClient, error) {
    cmd := exec.CommandContext(ctx, config.Command[0], config.Command[1:]...)
    cmd.Dir = root

    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    // Create JSON-RPC connection
    stream := &readWriteCloser{
        Reader: stdout,
        Writer: stdin,
        Closer: stdin,
    }

    conn := jsonrpc2.NewConn(
        ctx,
        jsonrpc2.NewBufferedStream(stream, jsonrpc2.VSCodeObjectCodec{}),
        &handler{},
    )

    client := &languageClient{
        conn:     conn,
        cmd:      cmd,
        root:     root,
        serverID: config.ID,
    }

    // Initialize server
    if err := client.initialize(ctx, root); err != nil {
        cmd.Process.Kill()
        return nil, err
    }

    return client, nil
}

// initialize sends the initialize request to the server
func (lc *languageClient) initialize(ctx context.Context, root string) error {
    params := lsp.InitializeParams{
        RootURI: lsp.DocumentURI("file://" + root),
        Capabilities: lsp.ClientCapabilities{
            TextDocument: lsp.TextDocumentClientCapabilities{
                Hover: &lsp.HoverCapability{
                    ContentFormat: []lsp.MarkupKind{lsp.PlainText, lsp.Markdown},
                },
                DocumentSymbol: &lsp.DocumentSymbolCapability{
                    SymbolKind: &lsp.SymbolKindCapability{
                        ValueSet: allSymbolKinds(),
                    },
                },
            },
            Workspace: lsp.WorkspaceClientCapabilities{
                Symbol: &lsp.WorkspaceSymbolCapability{
                    SymbolKind: &lsp.SymbolKindCapability{
                        ValueSet: allSymbolKinds(),
                    },
                },
            },
        },
    }

    var result lsp.InitializeResult
    if err := lc.conn.Call(ctx, "initialize", params, &result); err != nil {
        return err
    }

    // Send initialized notification
    return lc.conn.Notify(ctx, "initialized", struct{}{})
}

// findProjectRoot finds the project root for a file
func (c *Client) findProjectRoot(filePath, serverID string) string {
    dir := filepath.Dir(filePath)

    // Look for project markers based on server type
    markers := map[string][]string{
        "typescript": {"package.json", "tsconfig.json"},
        "go":         {"go.mod"},
        "python":     {"pyproject.toml", "setup.py"},
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
```

### LSP Operations

```go
// internal/lsp/operations.go
package lsp

import (
    "context"
    "fmt"

    "github.com/sourcegraph/go-lsp"
)

// Symbol kinds
const (
    SymbolKindFile          = 1
    SymbolKindModule        = 2
    SymbolKindNamespace     = 3
    SymbolKindPackage       = 4
    SymbolKindClass         = 5
    SymbolKindMethod        = 6
    SymbolKindProperty      = 7
    SymbolKindField         = 8
    SymbolKindConstructor   = 9
    SymbolKindEnum          = 10
    SymbolKindInterface     = 11
    SymbolKindFunction      = 12
    SymbolKindVariable      = 13
    SymbolKindConstant      = 14
    SymbolKindString        = 15
    SymbolKindNumber        = 16
    SymbolKindBoolean       = 17
    SymbolKindArray         = 18
    SymbolKindObject        = 19
    SymbolKindStruct        = 23
)

// Symbol represents a code symbol
type Symbol struct {
    Name     string `json:"name"`
    Kind     int    `json:"kind"`
    Location struct {
        URI   string `json:"uri"`
        Range struct {
            Start struct {
                Line      int `json:"line"`
                Character int `json:"character"`
            } `json:"start"`
            End struct {
                Line      int `json:"line"`
                Character int `json:"character"`
            } `json:"end"`
        } `json:"range"`
    } `json:"location"`
}

// Diagnostic represents a code diagnostic
type Diagnostic struct {
    Range struct {
        Start struct {
            Line      int `json:"line"`
            Character int `json:"character"`
        } `json:"start"`
        End struct {
            Line      int `json:"line"`
            Character int `json:"character"`
        } `json:"end"`
    } `json:"range"`
    Severity int    `json:"severity"`
    Message  string `json:"message"`
    Source   string `json:"source"`
}

// WorkspaceSymbol searches for symbols in the workspace
func (c *Client) WorkspaceSymbol(ctx context.Context, query string) ([]Symbol, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var allSymbols []Symbol

    for _, client := range c.clients {
        symbols, err := client.workspaceSymbol(ctx, query)
        if err != nil {
            continue // Skip failed clients
        }
        allSymbols = append(allSymbols, symbols...)
    }

    return allSymbols, nil
}

func (lc *languageClient) workspaceSymbol(ctx context.Context, query string) ([]Symbol, error) {
    params := lsp.WorkspaceSymbolParams{
        Query: query,
    }

    var result []lsp.SymbolInformation
    if err := lc.conn.Call(ctx, "workspace/symbol", params, &result); err != nil {
        return nil, err
    }

    symbols := make([]Symbol, len(result))
    for i, s := range result {
        symbols[i] = Symbol{
            Name: s.Name,
            Kind: int(s.Kind),
        }
        symbols[i].Location.URI = string(s.Location.URI)
        symbols[i].Location.Range.Start.Line = s.Location.Range.Start.Line
        symbols[i].Location.Range.Start.Character = s.Location.Range.Start.Character
        symbols[i].Location.Range.End.Line = s.Location.Range.End.Line
        symbols[i].Location.Range.End.Character = s.Location.Range.End.Character
    }

    return symbols, nil
}

// Hover returns hover information for a position
func (c *Client) Hover(ctx context.Context, file string, line, character int) (string, error) {
    client, err := c.getClient(ctx, file)
    if err != nil {
        return "", err
    }

    params := lsp.TextDocumentPositionParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: lsp.DocumentURI("file://" + file),
        },
        Position: lsp.Position{
            Line:      line,
            Character: character,
        },
    }

    var result *lsp.Hover
    if err := client.conn.Call(ctx, "textDocument/hover", params, &result); err != nil {
        return "", err
    }

    if result == nil {
        return "", nil
    }

    // Extract text from hover contents
    switch v := result.Contents.(type) {
    case string:
        return v, nil
    case lsp.MarkupContent:
        return v.Value, nil
    case []interface{}:
        var parts []string
        for _, p := range v {
            if s, ok := p.(string); ok {
                parts = append(parts, s)
            }
        }
        return strings.Join(parts, "\n"), nil
    }

    return "", nil
}

// DocumentSymbol returns symbols in a document
func (c *Client) DocumentSymbol(ctx context.Context, file string) ([]Symbol, error) {
    client, err := c.getClient(ctx, file)
    if err != nil {
        return nil, err
    }

    params := lsp.DocumentSymbolParams{
        TextDocument: lsp.TextDocumentIdentifier{
            URI: lsp.DocumentURI("file://" + file),
        },
    }

    var result []lsp.SymbolInformation
    if err := client.conn.Call(ctx, "textDocument/documentSymbol", params, &result); err != nil {
        return nil, err
    }

    symbols := make([]Symbol, len(result))
    for i, s := range result {
        symbols[i] = Symbol{
            Name: s.Name,
            Kind: int(s.Kind),
        }
        symbols[i].Location.URI = string(s.Location.URI)
        symbols[i].Location.Range.Start.Line = s.Location.Range.Start.Line
        symbols[i].Location.Range.Start.Character = s.Location.Range.Start.Character
    }

    return symbols, nil
}

// Diagnostics returns all diagnostics across open files
func (c *Client) Diagnostics(ctx context.Context) map[string][]Diagnostic {
    c.mu.RLock()
    defer c.mu.RUnlock()

    result := make(map[string][]Diagnostic)

    for _, client := range c.clients {
        // Diagnostics are pushed via notifications, stored in client
        // This is a simplified version
    }

    return result
}

// TouchFile notifies the server of file changes
func (c *Client) TouchFile(ctx context.Context, file string) error {
    client, err := c.getClient(ctx, file)
    if err != nil {
        return err
    }

    content, err := os.ReadFile(file)
    if err != nil {
        return err
    }

    params := lsp.DidOpenTextDocumentParams{
        TextDocument: lsp.TextDocumentItem{
            URI:        lsp.DocumentURI("file://" + file),
            LanguageID: detectLanguageID(file),
            Version:    1,
            Text:       string(content),
        },
    }

    return client.conn.Notify(ctx, "textDocument/didOpen", params)
}

// Close shuts down all language servers
func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, client := range c.clients {
        client.conn.Notify(context.Background(), "shutdown", nil)
        client.conn.Notify(context.Background(), "exit", nil)
        client.cmd.Process.Kill()
    }

    c.clients = make(map[string]*languageClient)
    return nil
}

// Status returns the status of all LSP servers
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

type ServerStatus struct {
    ID     string `json:"id"`
    Root   string `json:"root"`
    Key    string `json:"key"`
    Active bool   `json:"active"`
}
```

---

## 7.2 Model Context Protocol (MCP) Support

### MCP Client

```go
// internal/mcp/client.go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "sync"
    "time"
)

// Config defines MCP server configuration
type Config struct {
    Enabled     bool              `json:"enabled"`
    Type        string            `json:"type"` // "remote" | "local"
    URL         string            `json:"url,omitempty"`
    Headers     map[string]string `json:"headers,omitempty"`
    Command     []string          `json:"command,omitempty"`
    Environment map[string]string `json:"environment,omitempty"`
    Timeout     int               `json:"timeout,omitempty"` // milliseconds
}

// Client manages MCP server connections
type Client struct {
    mu       sync.RWMutex
    servers  map[string]*mcpServer
    configs  map[string]*Config
}

// mcpServer represents a connected MCP server
type mcpServer struct {
    name      string
    config    *Config
    transport Transport
    tools     []Tool
    status    string // "connected" | "disabled" | "failed"
    error     string
}

// Transport interface for MCP communication
type Transport interface {
    Send(ctx context.Context, method string, params any) (json.RawMessage, error)
    Close() error
}

// Tool represents an MCP tool
type Tool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    InputSchema json.RawMessage `json:"inputSchema"`
}

// NewClient creates a new MCP client
func NewClient() *Client {
    return &Client{
        servers: make(map[string]*mcpServer),
        configs: make(map[string]*Config),
    }
}

// AddServer adds and connects to an MCP server
func (c *Client) AddServer(ctx context.Context, name string, config *Config) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Check if already exists
    if _, ok := c.servers[name]; ok {
        return fmt.Errorf("server already exists: %s", name)
    }

    if !config.Enabled {
        c.servers[name] = &mcpServer{
            name:   name,
            config: config,
            status: "disabled",
        }
        return nil
    }

    server, err := c.connectServer(ctx, name, config)
    if err != nil {
        c.servers[name] = &mcpServer{
            name:   name,
            config: config,
            status: "failed",
            error:  err.Error(),
        }
        return err
    }

    c.servers[name] = server
    return nil
}

// connectServer establishes connection to an MCP server
func (c *Client) connectServer(ctx context.Context, name string, config *Config) (*mcpServer, error) {
    var transport Transport
    var err error

    timeout := time.Duration(config.Timeout) * time.Millisecond
    if timeout == 0 {
        timeout = 5 * time.Second
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    switch config.Type {
    case "remote":
        transport, err = NewHTTPTransport(config.URL, config.Headers)
    case "local":
        transport, err = NewStdioTransport(ctx, config.Command, config.Environment)
    default:
        return nil, fmt.Errorf("unknown transport type: %s", config.Type)
    }

    if err != nil {
        return nil, err
    }

    server := &mcpServer{
        name:      name,
        config:    config,
        transport: transport,
        status:    "connected",
    }

    // Initialize and get tools
    if err := server.initialize(ctx); err != nil {
        transport.Close()
        return nil, err
    }

    return server, nil
}

// initialize sends the initialize request and lists tools
func (s *mcpServer) initialize(ctx context.Context) error {
    // Initialize
    _, err := s.transport.Send(ctx, "initialize", map[string]any{
        "protocolVersion": "2024-11-05",
        "capabilities": map[string]any{
            "tools": map[string]any{},
        },
        "clientInfo": map[string]any{
            "name":    "opencode",
            "version": "1.0.0",
        },
    })
    if err != nil {
        return fmt.Errorf("initialize failed: %w", err)
    }

    // List tools
    result, err := s.transport.Send(ctx, "tools/list", nil)
    if err != nil {
        return fmt.Errorf("tools/list failed: %w", err)
    }

    var toolsResp struct {
        Tools []Tool `json:"tools"`
    }
    if err := json.Unmarshal(result, &toolsResp); err != nil {
        return err
    }

    s.tools = toolsResp.Tools
    return nil
}

// Tools returns all tools from all connected servers
func (c *Client) Tools() []Tool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var allTools []Tool
    for name, server := range c.servers {
        if server.status != "connected" {
            continue
        }

        for _, tool := range server.tools {
            // Prefix tool name with server name
            prefixedTool := tool
            prefixedTool.Name = sanitizeToolName(name) + "_" + sanitizeToolName(tool.Name)
            allTools = append(allTools, prefixedTool)
        }
    }

    return allTools
}

// ExecuteTool executes a tool on the appropriate server
func (c *Client) ExecuteTool(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
    c.mu.RLock()

    // Find server and tool
    var targetServer *mcpServer
    var originalToolName string

    for name, server := range c.servers {
        if server.status != "connected" {
            continue
        }

        prefix := sanitizeToolName(name) + "_"
        if strings.HasPrefix(toolName, prefix) {
            targetServer = server
            originalToolName = strings.TrimPrefix(toolName, prefix)
            break
        }
    }
    c.mu.RUnlock()

    if targetServer == nil {
        return "", fmt.Errorf("no server found for tool: %s", toolName)
    }

    // Execute tool
    result, err := targetServer.transport.Send(ctx, "tools/call", map[string]any{
        "name":      originalToolName,
        "arguments": json.RawMessage(args),
    })
    if err != nil {
        return "", err
    }

    var callResult struct {
        Content []struct {
            Type string `json:"type"`
            Text string `json:"text"`
        } `json:"content"`
    }
    if err := json.Unmarshal(result, &callResult); err != nil {
        return string(result), nil
    }

    // Extract text content
    var output strings.Builder
    for _, c := range callResult.Content {
        if c.Type == "text" {
            output.WriteString(c.Text)
        }
    }

    return output.String(), nil
}

// Status returns status of all MCP servers
func (c *Client) Status() []MCPStatus {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var status []MCPStatus
    for name, server := range c.servers {
        s := MCPStatus{
            Name:      name,
            Status:    server.status,
            ToolCount: len(server.tools),
        }
        if server.error != "" {
            s.Error = &server.error
        }
        status = append(status, s)
    }
    return status
}

type MCPStatus struct {
    Name      string  `json:"name"`
    Status    string  `json:"status"`
    ToolCount int     `json:"toolCount"`
    Error     *string `json:"error,omitempty"`
}

// Close disconnects all servers
func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, server := range c.servers {
        if server.transport != nil {
            server.transport.Close()
        }
    }

    c.servers = make(map[string]*mcpServer)
    return nil
}

// sanitizeToolName replaces non-alphanumeric chars with underscore
func sanitizeToolName(name string) string {
    var result strings.Builder
    for _, r := range name {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
            result.WriteRune(r)
        } else {
            result.WriteRune('_')
        }
    }
    return result.String()
}
```

### MCP Transports

```go
// internal/mcp/transport.go
package mcp

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "sync"
    "sync/atomic"
)

// HTTPTransport implements MCP over HTTP
type HTTPTransport struct {
    url     string
    headers map[string]string
    client  *http.Client
}

func NewHTTPTransport(url string, headers map[string]string) (*HTTPTransport, error) {
    return &HTTPTransport{
        url:     url,
        headers: headers,
        client:  &http.Client{},
    }, nil
}

func (t *HTTPTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
    reqBody := map[string]any{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  method,
        "params":  params,
    }

    body, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    for k, v := range t.headers {
        req.Header.Set(k, v)
    }

    resp, err := t.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Result json.RawMessage `json:"result"`
        Error  *struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
        } `json:"error"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    if result.Error != nil {
        return nil, fmt.Errorf("MCP error %d: %s", result.Error.Code, result.Error.Message)
    }

    return result.Result, nil
}

func (t *HTTPTransport) Close() error {
    return nil
}

// StdioTransport implements MCP over stdio
type StdioTransport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout *bufio.Reader
    mu     sync.Mutex
    nextID int64
}

func NewStdioTransport(ctx context.Context, command []string, env map[string]string) (*StdioTransport, error) {
    if len(command) == 0 {
        return nil, fmt.Errorf("empty command")
    }

    cmd := exec.CommandContext(ctx, command[0], command[1:]...)

    // Set environment
    cmd.Env = os.Environ()
    for k, v := range env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }

    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    return &StdioTransport{
        cmd:    cmd,
        stdin:  stdin,
        stdout: bufio.NewReader(stdout),
    }, nil
}

func (t *StdioTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
    t.mu.Lock()
    defer t.mu.Unlock()

    id := atomic.AddInt64(&t.nextID, 1)

    req := map[string]any{
        "jsonrpc": "2.0",
        "id":      id,
        "method":  method,
    }
    if params != nil {
        req["params"] = params
    }

    reqJSON, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    // Write request (newline-delimited JSON)
    if _, err := t.stdin.Write(append(reqJSON, '\n')); err != nil {
        return nil, err
    }

    // Read response
    line, err := t.stdout.ReadBytes('\n')
    if err != nil {
        return nil, err
    }

    var resp struct {
        ID     int64           `json:"id"`
        Result json.RawMessage `json:"result"`
        Error  *struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
        } `json:"error"`
    }

    if err := json.Unmarshal(line, &resp); err != nil {
        return nil, err
    }

    if resp.Error != nil {
        return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
    }

    return resp.Result, nil
}

func (t *StdioTransport) Close() error {
    t.stdin.Close()
    return t.cmd.Process.Kill()
}
```

---

## 7.3 Multi-Agent System

### Agent Configuration

```go
// internal/agent/agent.go
package agent

import (
    "strings"
)

// Agent represents an agent configuration
type Agent struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Mode        string                 `json:"mode"` // "primary" | "subagent" | "all"
    BuiltIn     bool                   `json:"builtIn"`
    Permission  Permission             `json:"permission"`
    Tools       map[string]bool        `json:"tools"`
    Options     map[string]any         `json:"options"`
    Temperature float64                `json:"temperature,omitempty"`
    TopP        float64                `json:"topP,omitempty"`
    Model       *ModelRef              `json:"model,omitempty"`
    Prompt      string                 `json:"prompt,omitempty"`
    Color       string                 `json:"color,omitempty"`
}

type ModelRef struct {
    ProviderID string `json:"providerID"`
    ModelID    string `json:"modelID"`
}

// Permission defines agent permission settings
type Permission struct {
    Edit            string            `json:"edit,omitempty"`       // "allow" | "deny" | "ask"
    Bash            map[string]string `json:"bash,omitempty"`       // pattern -> action
    WebFetch        string            `json:"webfetch,omitempty"`
    ExternalDir     string            `json:"external_directory,omitempty"`
    DoomLoop        string            `json:"doom_loop,omitempty"`
}

// ToolEnabled checks if a tool is enabled for this agent
func (a *Agent) ToolEnabled(toolID string) bool {
    // Check exact match
    if enabled, ok := a.Tools[toolID]; ok {
        return enabled
    }

    // Check wildcard patterns
    for pattern, enabled := range a.Tools {
        if matchWildcard(pattern, toolID) {
            return enabled
        }
    }

    // Default: enabled
    return true
}

// CheckBashPermission checks bash command permission
func (a *Agent) CheckBashPermission(command string) string {
    // Check each pattern
    for pattern, action := range a.Permission.Bash {
        if matchWildcard(pattern, command) {
            return action
        }
    }

    // Default: ask
    return "ask"
}

// matchWildcard checks if a string matches a wildcard pattern
func matchWildcard(pattern, s string) bool {
    if pattern == "*" {
        return true
    }

    if strings.HasSuffix(pattern, "*") {
        prefix := strings.TrimSuffix(pattern, "*")
        return strings.HasPrefix(s, prefix)
    }

    if strings.HasPrefix(pattern, "*") {
        suffix := strings.TrimPrefix(pattern, "*")
        return strings.HasSuffix(s, suffix)
    }

    return pattern == s
}

// BuiltInAgents returns the default agent configurations
func BuiltInAgents() map[string]*Agent {
    return map[string]*Agent{
        "build": {
            Name:        "build",
            Description: "Primary agent for executing tasks, writing code, and making changes",
            Mode:        "primary",
            BuiltIn:     true,
            Permission: Permission{
                Edit:        "allow",
                Bash:        map[string]string{"*": "allow"},
                WebFetch:    "allow",
                ExternalDir: "ask",
                DoomLoop:    "ask",
            },
            Tools: map[string]bool{
                "*": true,
            },
        },
        "plan": {
            Name:        "plan",
            Description: "Planning agent for analysis and exploration without making changes",
            Mode:        "primary",
            BuiltIn:     true,
            Permission: Permission{
                Edit:        "deny",
                Bash:        map[string]string{
                    "grep*": "allow",
                    "find*": "allow",
                    "ls*":   "allow",
                    "cat*":  "allow",
                    "git status": "allow",
                    "git diff*":  "allow",
                    "git log*":   "allow",
                    "*":          "deny",
                },
                WebFetch:    "allow",
                ExternalDir: "deny",
                DoomLoop:    "deny",
            },
            Tools: map[string]bool{
                "read":  true,
                "glob":  true,
                "grep":  true,
                "ls":    true,
                "bash":  true,
                "edit":  false,
                "write": false,
            },
        },
        "general": {
            Name:        "general",
            Description: "General-purpose subagent for searches and exploration",
            Mode:        "subagent",
            BuiltIn:     true,
            Permission: Permission{
                Edit:        "deny",
                Bash:        map[string]string{"*": "deny"},
                WebFetch:    "allow",
                ExternalDir: "deny",
                DoomLoop:    "deny",
            },
            Tools: map[string]bool{
                "read":     true,
                "glob":     true,
                "grep":     true,
                "webfetch": true,
                "bash":     false,
                "edit":     false,
                "write":    false,
            },
        },
    }
}
```

### Agent Registry

```go
// internal/agent/registry.go
package agent

import (
    "fmt"
    "sync"
)

// Registry manages agent configurations
type Registry struct {
    mu     sync.RWMutex
    agents map[string]*Agent
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
    r := &Registry{
        agents: make(map[string]*Agent),
    }

    // Register built-in agents
    for name, agent := range BuiltInAgents() {
        r.agents[name] = agent
    }

    return r
}

// Get retrieves an agent by name
func (r *Registry) Get(name string) (*Agent, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    agent, ok := r.agents[name]
    if !ok {
        return nil, fmt.Errorf("agent not found: %s", name)
    }

    return agent, nil
}

// Register adds or updates an agent
func (r *Registry) Register(agent *Agent) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.agents[agent.Name] = agent
}

// List returns all registered agents
func (r *Registry) List() []*Agent {
    r.mu.RLock()
    defer r.mu.RUnlock()

    agents := make([]*Agent, 0, len(r.agents))
    for _, agent := range r.agents {
        agents = append(agents, agent)
    }
    return agents
}

// ListPrimary returns agents with primary mode
func (r *Registry) ListPrimary() []*Agent {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var agents []*Agent
    for _, agent := range r.agents {
        if agent.Mode == "primary" || agent.Mode == "all" {
            agents = append(agents, agent)
        }
    }
    return agents
}

// ListSubagents returns agents with subagent mode
func (r *Registry) ListSubagents() []*Agent {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var agents []*Agent
    for _, agent := range r.agents {
        if agent.Mode == "subagent" || agent.Mode == "all" {
            agents = append(agents, agent)
        }
    }
    return agents
}

// LoadFromConfig loads custom agents from configuration
func (r *Registry) LoadFromConfig(config map[string]AgentConfig) {
    r.mu.Lock()
    defer r.mu.Unlock()

    for name, cfg := range config {
        // Start with default or create new
        agent, exists := r.agents[name]
        if !exists {
            agent = &Agent{
                Name: name,
                Mode: "primary",
            }
        }

        // Apply config overrides
        if cfg.Model != nil {
            agent.Model = cfg.Model
        }
        if cfg.Prompt != "" {
            agent.Prompt = cfg.Prompt
        }
        if cfg.Temperature > 0 {
            agent.Temperature = cfg.Temperature
        }
        if cfg.TopP > 0 {
            agent.TopP = cfg.TopP
        }
        if cfg.Tools != nil {
            if agent.Tools == nil {
                agent.Tools = make(map[string]bool)
            }
            for k, v := range cfg.Tools {
                agent.Tools[k] = v
            }
        }
        if cfg.Permission != nil {
            // Merge permissions
            if cfg.Permission.Edit != "" {
                agent.Permission.Edit = cfg.Permission.Edit
            }
            if cfg.Permission.Bash != nil {
                if agent.Permission.Bash == nil {
                    agent.Permission.Bash = make(map[string]string)
                }
                for k, v := range cfg.Permission.Bash {
                    agent.Permission.Bash[k] = v
                }
            }
        }

        r.agents[name] = agent
    }
}

// AgentConfig represents user configuration for an agent
type AgentConfig struct {
    Model       *ModelRef         `json:"model,omitempty"`
    Prompt      string            `json:"prompt,omitempty"`
    Temperature float64           `json:"temperature,omitempty"`
    TopP        float64           `json:"topP,omitempty"`
    Tools       map[string]bool   `json:"tools,omitempty"`
    Permission  *Permission       `json:"permission,omitempty"`
}
```

---

## 7.4 Task Tool (Sub-agent Spawning)

```go
// internal/tool/task.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/opencode-ai/opencode-server/internal/agent"
    "github.com/opencode-ai/opencode-server/internal/session"
)

// TaskTool allows spawning sub-agents for complex tasks
type TaskTool struct {
    processor     *session.Processor
    agentRegistry *agent.Registry
}

type TaskInput struct {
    Description  string `json:"description"`
    Prompt       string `json:"prompt"`
    SubagentType string `json:"subagent_type"`
    Model        string `json:"model,omitempty"`
    Resume       string `json:"resume,omitempty"`
}

func NewTaskTool(processor *session.Processor, registry *agent.Registry) *TaskTool {
    return &TaskTool{
        processor:     processor,
        agentRegistry: registry,
    }
}

func (t *TaskTool) ID() string          { return "task" }
func (t *TaskTool) Description() string { return taskDescription }

func (t *TaskTool) Parameters() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "description": {
                "type": "string",
                "description": "A short (3-5 word) description of the task"
            },
            "prompt": {
                "type": "string",
                "description": "The detailed task for the agent to perform"
            },
            "subagent_type": {
                "type": "string",
                "description": "The type of specialized agent to use"
            },
            "model": {
                "type": "string",
                "description": "Optional model to use (sonnet, opus, haiku)"
            },
            "resume": {
                "type": "string",
                "description": "Optional agent ID to resume from"
            }
        },
        "required": ["description", "prompt", "subagent_type"]
    }`)
}

func (t *TaskTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
    var params TaskInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    // Get subagent configuration
    subagent, err := t.agentRegistry.Get(params.SubagentType)
    if err != nil {
        return nil, fmt.Errorf("unknown subagent type: %s", params.SubagentType)
    }

    // Verify subagent mode
    if subagent.Mode != "subagent" && subagent.Mode != "all" {
        return nil, fmt.Errorf("agent %s cannot be used as subagent", params.SubagentType)
    }

    // Update metadata
    toolCtx.SetMetadata(params.Description, map[string]any{
        "subagent": params.SubagentType,
        "status":   "starting",
    })

    // Create subtask session (fork from current)
    subtaskSession, err := t.createSubtaskSession(ctx, toolCtx.SessionID, params)
    if err != nil {
        return nil, err
    }

    // Collect results
    var finalOutput string

    // Process subtask with streaming updates
    err = t.processor.Process(ctx, subtaskSession.ID, func(msg *types.Message, parts []types.Part) {
        // Extract latest text output
        for _, part := range parts {
            if textPart, ok := part.(*types.TextPart); ok {
                finalOutput = textPart.Text
            }
        }

        // Update metadata with progress
        toolCtx.SetMetadata(params.Description, map[string]any{
            "subagent": params.SubagentType,
            "status":   "running",
            "output":   truncate(finalOutput, 500),
        })
    })

    if err != nil {
        return &Result{
            Title:  fmt.Sprintf("Subtask failed: %s", params.Description),
            Output: fmt.Sprintf("Error: %s", err.Error()),
        }, nil
    }

    return &Result{
        Title:  fmt.Sprintf("Completed: %s", params.Description),
        Output: finalOutput,
        Metadata: map[string]any{
            "subagent":  params.SubagentType,
            "sessionID": subtaskSession.ID,
        },
    }, nil
}

func (t *TaskTool) createSubtaskSession(ctx context.Context, parentSessionID string, params TaskInput) (*types.Session, error) {
    // This would create a child session for the subtask
    // Implementation depends on session service
    return nil, fmt.Errorf("not implemented")
}

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}

const taskDescription = `Launch a new agent to handle complex, multi-step tasks autonomously.

The Task tool launches specialized agents (subprocesses) that autonomously handle complex tasks.

Available agent types:
- general-purpose: General exploration and research
- Explore: Fast codebase exploration
- Plan: Planning without making changes

Usage notes:
- Launch multiple agents concurrently when possible
- Each agent invocation is stateless
- The agent's outputs should be trusted`
```

---

## 7.5 Deliverables

### Files to Create

| File | Lines (Est.) | Complexity |
|------|--------------|------------|
| `internal/lsp/client.go` | 250 | High |
| `internal/lsp/operations.go` | 200 | Medium |
| `internal/mcp/client.go` | 300 | High |
| `internal/mcp/transport.go` | 200 | Medium |
| `internal/agent/agent.go` | 150 | Medium |
| `internal/agent/registry.go` | 150 | Low |
| `internal/tool/task.go` | 200 | High |
| `internal/tool/plugin.go` | 150 | Medium |

### Integration Tests

```go
// test/integration/lsp_test.go

func TestLSP_WorkspaceSymbol(t *testing.T) { /* ... */ }
func TestLSP_Hover(t *testing.T) { /* ... */ }
func TestLSP_DocumentSymbol(t *testing.T) { /* ... */ }
func TestLSP_MultipleServers(t *testing.T) { /* ... */ }

// test/integration/mcp_test.go

func TestMCP_HTTPTransport(t *testing.T) { /* ... */ }
func TestMCP_StdioTransport(t *testing.T) { /* ... */ }
func TestMCP_ListTools(t *testing.T) { /* ... */ }
func TestMCP_ExecuteTool(t *testing.T) { /* ... */ }

// test/integration/agent_test.go

func TestAgent_BuiltInAgents(t *testing.T) { /* ... */ }
func TestAgent_ToolPermissions(t *testing.T) { /* ... */ }
func TestAgent_BashPermissions(t *testing.T) { /* ... */ }
func TestAgent_CustomConfig(t *testing.T) { /* ... */ }

// test/integration/task_test.go

func TestTaskTool_SpawnSubagent(t *testing.T) { /* ... */ }
func TestTaskTool_SubagentTypes(t *testing.T) { /* ... */ }
```

### Acceptance Criteria

- [ ] LSP client connects to TypeScript, Go, Python, Rust servers
- [ ] LSP workspace symbol search returns results
- [ ] LSP hover provides information
- [ ] MCP HTTP transport connects and lists tools
- [ ] MCP stdio transport spawns local servers
- [ ] MCP tools are exposed to the LLM
- [ ] Agent registry loads built-in agents
- [ ] Custom agents can be configured
- [ ] Agent permissions correctly filter tools and bash commands
- [ ] Task tool spawns subagents for complex tasks
- [ ] Subagent results are returned to parent session
- [ ] Test coverage >75% for advanced features
