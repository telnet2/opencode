# Comparative Testing Plan: TypeScript vs Go OpenCode Implementations

## Executive Summary

This document outlines a comprehensive strategy for comparing the TypeScript (`packages/opencode`) and Go (`go-opencode`) implementations of the OpenCode server API. The goal is to ensure feature parity, validate API compatibility, and automate continuous comparison testing.

---

## 1. Implementation Overview

### 1.1 TypeScript Implementation (packages/opencode)

- **Framework:** Hono with OpenAPI support
- **Runtime:** Bun
- **Validation:** Zod schemas
- **Documentation:** OpenAPI 3.1.1 at `/doc`
- **Test Framework:** Bun test runner

### 1.2 Go Implementation (go-opencode)

- **Framework:** Chi router
- **Runtime:** Native Go
- **Validation:** Manual + struct tags
- **Documentation:** OpenAPI 3.0.0 at `/doc`
- **Test Framework:** Ginkgo/Gomega

---

## 2. Endpoint Comparison Matrix

### 2.1 Core Endpoints (Full Parity)

| Category | Endpoint | TS | Go | Notes |
|----------|----------|----|----|-------|
| **Session** | `GET /session` | ✅ | ✅ | List sessions |
| | `POST /session` | ✅ | ✅ | Create session |
| | `GET /session/:id` | ✅ | ✅ | Get session |
| | `PATCH /session/:id` | ✅ | ✅ | Update session |
| | `DELETE /session/:id` | ✅ | ✅ | Delete session |
| | `GET /session/:id/children` | ✅ | ✅ | Get child sessions |
| | `GET /session/:id/todo` | ✅ | ✅ | Get todos |
| | `GET /session/status` | ✅ | ✅ | Session status |
| **Messages** | `GET /session/:id/message` | ✅ | ✅ | List messages |
| | `POST /session/:id/message` | ✅ | ✅ | Send message (streaming) |
| | `GET /session/:id/message/:msgId` | ✅ | ✅ | Get message |
| | `POST /session/:id/command` | ✅ | ✅ | Execute command |
| | `POST /session/:id/shell` | ✅ | ✅ | Run shell |
| **Session Control** | `POST /session/:id/abort` | ✅ | ✅ | Abort session |
| | `POST /session/:id/revert` | ✅ | ✅ | Revert message |
| | `POST /session/:id/unrevert` | ✅ | ✅ | Undo revert |
| | `POST /session/:id/fork` | ✅ | ✅ | Fork session |
| | `POST /session/:id/init` | ✅ | ✅ | Initialize |
| | `POST /session/:id/summarize` | ✅ | ✅ | Summarize |
| | `POST /session/:id/share` | ✅ | ✅ | Share session |
| | `DELETE /session/:id/share` | ✅ | ✅ | Unshare |
| | `POST /session/:id/permissions/:pid` | ✅ | ✅ | Permission response |
| | `GET /session/:id/diff` | ✅ | ✅ | Get diffs |
| **Config** | `GET /config` | ✅ | ✅ | Get config |
| | `PATCH /config` | ✅ | ✅ | Update config |
| | `GET /config/providers` | ✅ | ✅ | List providers |
| **Provider** | `GET /provider` | ✅ | ✅ | List providers |
| | `GET /provider/auth` | ✅ | ✅ | Auth methods |
| | `POST /provider/:id/oauth/authorize` | ✅ | ⚠️ | OAuth (partial) |
| | `POST /provider/:id/oauth/callback` | ✅ | ⚠️ | OAuth (partial) |
| | `PUT /auth/:id` | ✅ | ✅ | Set auth |
| **Files** | `GET /file` | ✅ | ✅ | List files |
| | `GET /file/content` | ✅ | ✅ | Read file |
| | `GET /file/status` | ✅ | ✅ | Git status |
| **Search** | `GET /find` | ✅ | ✅ | Text search |
| | `GET /find/file` | ✅ | ✅ | File search |
| | `GET /find/symbol` | ✅ | ⚠️ | Symbol search (stub) |
| **Project** | `GET /project` | ✅ | ✅ | List projects |
| | `GET /project/current` | ✅ | ✅ | Current project |
| **Events** | `GET /event` | ✅ | ✅ | SSE stream |
| | `GET /global/event` | ✅ | ✅ | Global SSE |
| **MCP** | `GET /mcp` | ✅ | ✅ | MCP status |
| | `POST /mcp` | ✅ | ✅ | Add server |
| | `DELETE /mcp/:name` | ✅ | ✅ | Remove server |
| | `GET /mcp/tools` | ✅ | ✅ | List tools |
| | `POST /mcp/tool/:name` | ✅ | ✅ | Execute tool |
| | `GET /mcp/resources` | ✅ | ✅ | List resources |
| | `GET /mcp/resource` | ✅ | ✅ | Read resource |
| **Commands** | `GET /command` | ✅ | ✅ | List commands |
| | `GET /command/:name` | ✅ | ✅ | Get command |
| | `POST /command/:name` | ✅ | ✅ | Execute command |
| **Formatter** | `GET /formatter` | ✅ | ✅ | Status |
| | `POST /formatter/format` | ✅ | ✅ | Format code |
| **LSP** | `GET /lsp` | ✅ | ✅ | LSP status |
| **Agents** | `GET /agent` | ✅ | ✅ | List agents |
| **Tools** | `GET /experimental/tool/ids` | ✅ | ✅ | Tool IDs |
| | `GET /experimental/tool` | ✅ | ✅ | Tool list |
| **Client Tools** | `POST /client-tools/register` | ✅ | ✅ | Register |
| | `DELETE /client-tools/unregister` | ✅ | ✅ | Unregister |
| | `POST /client-tools/result` | ✅ | ✅ | Submit result |
| | `GET /client-tools/pending/:id` | ✅ | ❌ | Pending SSE (missing) |
| | `GET /client-tools/tools/:id` | ✅ | ❌ | Get tools (missing) |
| | `GET /client-tools/tools` | ✅ | ❌ | All tools (missing) |
| **Instance** | `POST /instance/dispose` | ✅ | ✅ | Dispose |
| | `GET /path` | ✅ | ✅ | Get paths |
| | `POST /log` | ✅ | ✅ | Write log |
| **TUI** | `POST /tui/append-prompt` | ✅ | ✅ | Append prompt |
| | `POST /tui/execute-command` | ✅ | ✅ | Execute |
| | `POST /tui/show-toast` | ✅ | ✅ | Show toast |
| | `POST /tui/publish` | ✅ | ✅ | Publish event |
| | `POST /tui/open-help` | ✅ | ✅ | Open help |
| | `POST /tui/open-sessions` | ✅ | ✅ | Open sessions |
| | `POST /tui/open-themes` | ✅ | ✅ | Open themes |
| | `POST /tui/open-models` | ✅ | ✅ | Open models |
| | `POST /tui/submit-prompt` | ✅ | ✅ | Submit |
| | `POST /tui/clear-prompt` | ✅ | ✅ | Clear |
| | `GET /tui/control/next` | ✅ | ✅ | Next request |
| | `POST /tui/control/response` | ✅ | ✅ | Response |
| **Docs** | `GET /doc` | ✅ | ✅ | OpenAPI spec |

**Legend:** ✅ Full parity | ⚠️ Partial implementation | ❌ Missing

---

## 2.2 Parity Gaps: Detailed Analysis

This section documents the specific missing functionality in the Go implementation that requires attention for full feature parity.

### Gap 1: Symbol Search (`GET /find/symbol`)

**Status:** ⚠️ Stub - Returns 501 Not Implemented

**TypeScript Implementation** (`packages/opencode/src/server/server.ts:1399`):
- Fully functional LSP-based workspace symbol search
- Integrates with LSP clients to query workspace symbols
- Returns array of `LSP.Symbol` objects with location information
- Supports symbol query filtering

**Go Implementation** (`go-opencode/internal/server/handlers_file.go:247-250`):
```go
// searchSymbols handles GET /find/symbol
func (s *Server) searchSymbols(w http.ResponseWriter, r *http.Request) {
    // LSP-based symbol search (placeholder)
    notImplemented(w)
}
```

**Required for Parity:**
1. Integrate with LSP client infrastructure (`go-opencode/internal/lsp/`)
2. Implement `workspace/symbol` LSP request
3. Parse and return symbol results matching TypeScript response schema:
   ```typescript
   interface Symbol {
     name: string
     kind: SymbolKind
     location: Location
     containerName?: string
   }
   ```

**Priority:** Medium - Used for code navigation features in IDEs

---

### Gap 2: Client Tools Pending Endpoint (`GET /client-tools/pending/:clientID`)

**Status:** ❌ Missing - No route defined

**TypeScript Implementation** (`packages/opencode/src/server/client-tools.ts:169-229`):
- SSE endpoint for streaming pending tool execution requests to clients
- Subscribes to `ClientToolRegistry.Event.ToolRequest` events
- Filters events by `clientID` parameter
- Sends `tool-request` events with full request payload
- Implements keepalive pings every 30 seconds
- Handles client disconnection with automatic cleanup via `ClientToolRegistry.cleanup(clientID)`

```typescript
// SSE endpoint for tool execution requests
.get("/pending/:clientID", async (c) => {
  const clientID = c.req.param("clientID")

  return streamSSE(c, async (stream) => {
    // Subscribe to tool request events for this client
    const unsubscribe = Bus.subscribe(ClientToolRegistry.Event.ToolRequest, async (event) => {
      if (event.properties.clientID === clientID) {
        await stream.writeSSE({
          event: "tool-request",
          data: JSON.stringify(event.properties.request),
        })
      }
    })

    // Keep connection alive with periodic pings
    const keepAlive = setInterval(async () => {
      await stream.writeSSE({ event: "ping", data: "" })
    }, 30000)

    // Wait for disconnect and cleanup
    await new Promise<void>((resolve) => {
      stream.onAbort(() => {
        unsubscribe()
        clearInterval(keepAlive)
        ClientToolRegistry.cleanup(clientID)
        resolve()
      })
    })
  })
})
```

**Go Implementation:**
- Route not defined in `go-opencode/internal/server/routes.go`
- Only basic client tool routes exist:
  - `POST /client-tools/register`
  - `DELETE /client-tools/unregister`
  - `POST /client-tools/execute`
  - `POST /client-tools/result`

**Additional Missing Endpoints:**
| Endpoint | Description | Status |
|----------|-------------|--------|
| `GET /client-tools/pending/:clientID` | SSE stream for tool requests | ❌ Missing |
| `GET /client-tools/tools/:clientID` | Get tools for specific client | ❌ Missing |
| `GET /client-tools/tools` | Get all registered client tools | ❌ Missing |

**Required for Parity:**
1. Add SSE endpoint with client ID filtering
2. Implement event bus subscription for tool requests
3. Add keepalive ping mechanism (30s interval)
4. Implement proper cleanup on client disconnect
5. Add the two missing GET endpoints for tool introspection

**Priority:** High - Essential for external tool integrations (VSCode extension, etc.)

---

### Gap 3: Provider OAuth Endpoints

**Status:** ⚠️ Stub - Returns 501 Not Implemented

#### 3a. OAuth Authorize (`POST /provider/:id/oauth/authorize`)

**TypeScript Implementation** (`packages/opencode/src/provider/auth.ts:54-72`):
- Initiates OAuth flow for supported providers
- Returns authorization URL and method type (`auto` or `code`)
- Stores pending OAuth state for callback handling
- Returns `Authorization` object:
  ```typescript
  interface Authorization {
    url: string        // OAuth authorization URL
    method: "auto" | "code"  // Callback method type
    instructions: string     // User instructions
  }
  ```

**Go Implementation** (`go-opencode/internal/server/handlers_config.go:85-88`):
```go
// oauthAuthorize handles POST /provider/{providerID}/oauth/authorize
func (s *Server) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
    notImplemented(w)
}
```

#### 3b. OAuth Callback (`POST /provider/:id/oauth/callback`)

**TypeScript Implementation** (`packages/opencode/src/provider/auth.ts:74-114`):
- Handles OAuth callback with authorization code (for `code` method)
- Handles automatic token exchange (for `auto` method)
- Stores resulting tokens via `Auth.set()`:
  - API key for simple auth: `{ type: "api", key: string }`
  - OAuth tokens: `{ type: "oauth", access: string, refresh: string, expires: number }`
- Throws typed errors: `OauthMissing`, `OauthCodeMissing`, `OauthCallbackFailed`

**Go Implementation** (`go-opencode/internal/server/handlers_config.go:90-93`):
```go
// oauthCallback handles POST /provider/{providerID}/oauth/callback
func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
    notImplemented(w)
}
```

**Supported OAuth Providers (TypeScript):**
- Providers with OAuth support are loaded dynamically via plugin system
- Common providers: GitHub Copilot, Amazon Q, etc.
- Auth methods defined per-provider with `type: "oauth"` or `type: "api"`

**Required for Parity:**
1. Implement OAuth state management (pending authorizations)
2. Add provider-specific OAuth configuration loading
3. Implement authorization URL generation
4. Handle callback with code exchange
5. Store tokens securely (consider using Go's equivalent to the TypeScript Auth module)
6. Support both `auto` and `code` OAuth methods
7. Implement proper error types matching TypeScript errors

**Priority:** Medium - Required for OAuth-based providers (GitHub Copilot, etc.)

---

### Summary: Implementation Priority

| Gap | Priority | Effort | Impact |
|-----|----------|--------|--------|
| Client Tools Pending SSE | 🔴 High | Medium | Blocks external tool integrations |
| Client Tools GET endpoints | 🔴 High | Low | Blocks tool introspection |
| Provider OAuth | 🟡 Medium | High | Blocks OAuth-based providers |
| Symbol Search | 🟡 Medium | Medium | Reduces IDE integration quality |

### Recommended Implementation Order

1. **Phase 1: Client Tools (Week 1)**
   - Add `GET /client-tools/pending/:clientID` SSE endpoint
   - Add `GET /client-tools/tools/:clientID` endpoint
   - Add `GET /client-tools/tools` endpoint
   - Implement event bus subscription and cleanup

2. **Phase 2: Symbol Search (Week 2)**
   - Integrate with existing LSP infrastructure
   - Implement `workspace/symbol` request handling
   - Return properly formatted symbol results

3. **Phase 3: Provider OAuth (Week 3-4)**
   - Design OAuth state management
   - Implement authorize flow
   - Implement callback handling
   - Add token storage integration

---

## 3. Comparative Testing Framework Architecture

### 3.1 Test Infrastructure

```
comparative-tests/
├── cmd/
│   └── compare/
│       └── main.go              # CLI tool for running comparisons
├── internal/
│   ├── harness/
│   │   ├── harness.go           # Test harness orchestration
│   │   ├── ts_server.go         # TypeScript server manager
│   │   └── go_server.go         # Go server manager
│   ├── client/
│   │   └── dual_client.go       # Dual-server client
│   ├── compare/
│   │   ├── compare.go           # Response comparison logic
│   │   ├── json_diff.go         # JSON deep diff
│   │   ├── schema_validate.go   # Schema validation
│   │   └── tolerances.go        # Known difference tolerances
│   └── reporter/
│       ├── reporter.go          # Test result reporting
│       ├── html_report.go       # HTML report generation
│       └── json_report.go       # JSON report output
├── tests/
│   ├── session_test.go          # Session endpoint tests
│   ├── message_test.go          # Message endpoint tests
│   ├── file_test.go             # File endpoint tests
│   ├── config_test.go           # Config endpoint tests
│   ├── mcp_test.go              # MCP endpoint tests
│   ├── event_test.go            # SSE event tests
│   └── streaming_test.go        # Streaming response tests
├── fixtures/
│   ├── shared_config.json       # Shared test configuration
│   └── test_data/               # Test files and fixtures
└── Makefile                     # Build and test commands
```

### 3.2 Dual-Server Test Harness

```go
// internal/harness/harness.go
package harness

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// TestHarness manages both TS and Go servers for comparative testing
type TestHarness struct {
    TSServer   *TSServerManager
    GoServer   *GoServerManager
    Config     *SharedConfig
    WorkDir    string
    mu         sync.Mutex
}

// SharedConfig ensures both servers use identical configuration
type SharedConfig struct {
    Model           string            `json:"model"`
    Provider        map[string]any    `json:"provider"`
    Permission      map[string]string `json:"permission"`
    WorkDirectory   string            `json:"workDirectory"`
    StateDirectory  string            `json:"stateDirectory"`
    ConfigDirectory string            `json:"configDirectory"`
}

// NewTestHarness creates a new comparative test harness
func NewTestHarness(config *SharedConfig) (*TestHarness, error) {
    workDir, err := os.MkdirTemp("", "opencode-compare-*")
    if err != nil {
        return nil, err
    }

    return &TestHarness{
        Config:  config,
        WorkDir: workDir,
    }, nil
}

// Start launches both servers with identical configuration
func (h *TestHarness) Start(ctx context.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    // Generate shared configuration files
    if err := h.writeSharedConfig(); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    // Start TypeScript server
    h.TSServer = NewTSServerManager(h.Config, h.WorkDir)
    tsPort, err := h.TSServer.Start(ctx)
    if err != nil {
        return fmt.Errorf("failed to start TS server: %w", err)
    }

    // Start Go server
    h.GoServer = NewGoServerManager(h.Config, h.WorkDir)
    goPort, err := h.GoServer.Start(ctx)
    if err != nil {
        h.TSServer.Stop()
        return fmt.Errorf("failed to start Go server: %w", err)
    }

    // Wait for both servers to be ready
    if err := h.waitForServers(ctx, tsPort, goPort); err != nil {
        h.Stop()
        return err
    }

    return nil
}

// Stop shuts down both servers
func (h *TestHarness) Stop() error {
    var errs []error

    if h.TSServer != nil {
        if err := h.TSServer.Stop(); err != nil {
            errs = append(errs, err)
        }
    }

    if h.GoServer != nil {
        if err := h.GoServer.Stop(); err != nil {
            errs = append(errs, err)
        }
    }

    os.RemoveAll(h.WorkDir)

    if len(errs) > 0 {
        return fmt.Errorf("errors stopping servers: %v", errs)
    }
    return nil
}

// Client returns a dual client for making parallel requests
func (h *TestHarness) Client() *DualClient {
    return NewDualClient(h.TSServer.URL(), h.GoServer.URL())
}
```

### 3.3 TypeScript Server Manager

```go
// internal/harness/ts_server.go
package harness

import (
    "context"
    "fmt"
    "net"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

type TSServerManager struct {
    config  *SharedConfig
    workDir string
    cmd     *exec.Cmd
    port    int
    baseURL string
}

func NewTSServerManager(config *SharedConfig, workDir string) *TSServerManager {
    return &TSServerManager{
        config:  config,
        workDir: workDir,
    }
}

func (m *TSServerManager) Start(ctx context.Context) (int, error) {
    // Find available port
    port, err := findAvailablePort()
    if err != nil {
        return 0, err
    }
    m.port = port
    m.baseURL = fmt.Sprintf("http://localhost:%d", port)

    // Set up environment
    env := os.Environ()
    env = append(env,
        fmt.Sprintf("OPENCODE_PORT=%d", port),
        fmt.Sprintf("OPENCODE_STATE_DIR=%s/ts-state", m.workDir),
        fmt.Sprintf("OPENCODE_CONFIG_DIR=%s/ts-config", m.workDir),
    )

    // Add provider API keys from config
    for provider, cfg := range m.config.Provider {
        if cfgMap, ok := cfg.(map[string]any); ok {
            if apiKey, ok := cfgMap["apiKey"].(string); ok {
                envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
                env = append(env, fmt.Sprintf("%s=%s", envKey, apiKey))
            }
        }
    }

    // Start the TypeScript server using bun
    m.cmd = exec.CommandContext(ctx, "bun", "run", "start:server")
    m.cmd.Dir = filepath.Join(getProjectRoot(), "packages/opencode")
    m.cmd.Env = env
    m.cmd.Stdout = os.Stdout // For debugging
    m.cmd.Stderr = os.Stderr

    if err := m.cmd.Start(); err != nil {
        return 0, fmt.Errorf("failed to start TS server: %w", err)
    }

    return port, nil
}

func (m *TSServerManager) Stop() error {
    if m.cmd != nil && m.cmd.Process != nil {
        return m.cmd.Process.Kill()
    }
    return nil
}

func (m *TSServerManager) URL() string {
    return m.baseURL
}
```

### 3.4 Go Server Manager

```go
// internal/harness/go_server.go
package harness

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

type GoServerManager struct {
    config  *SharedConfig
    workDir string
    cmd     *exec.Cmd
    port    int
    baseURL string
}

func NewGoServerManager(config *SharedConfig, workDir string) *GoServerManager {
    return &GoServerManager{
        config:  config,
        workDir: workDir,
    }
}

func (m *GoServerManager) Start(ctx context.Context) (int, error) {
    port, err := findAvailablePort()
    if err != nil {
        return 0, err
    }
    m.port = port
    m.baseURL = fmt.Sprintf("http://localhost:%d", port)

    // Build the Go server if needed
    serverBin := filepath.Join(getProjectRoot(), "go-opencode/bin/opencode-server")
    if _, err := os.Stat(serverBin); os.IsNotExist(err) {
        buildCmd := exec.Command("go", "build", "-o", serverBin, "./cmd/opencode-server")
        buildCmd.Dir = filepath.Join(getProjectRoot(), "go-opencode")
        if err := buildCmd.Run(); err != nil {
            return 0, fmt.Errorf("failed to build Go server: %w", err)
        }
    }

    // Set up environment
    env := os.Environ()
    env = append(env,
        fmt.Sprintf("OPENCODE_PORT=%d", port),
        fmt.Sprintf("OPENCODE_STATE_DIR=%s/go-state", m.workDir),
        fmt.Sprintf("OPENCODE_CONFIG_DIR=%s/go-config", m.workDir),
        fmt.Sprintf("OPENCODE_DIRECTORY=%s", m.config.WorkDirectory),
    )

    // Add provider API keys
    for provider, cfg := range m.config.Provider {
        if cfgMap, ok := cfg.(map[string]any); ok {
            if apiKey, ok := cfgMap["apiKey"].(string); ok {
                envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
                env = append(env, fmt.Sprintf("%s=%s", envKey, apiKey))
            }
        }
    }

    m.cmd = exec.CommandContext(ctx, serverBin)
    m.cmd.Env = env
    m.cmd.Stdout = os.Stdout
    m.cmd.Stderr = os.Stderr

    if err := m.cmd.Start(); err != nil {
        return 0, fmt.Errorf("failed to start Go server: %w", err)
    }

    return port, nil
}

func (m *GoServerManager) Stop() error {
    if m.cmd != nil && m.cmd.Process != nil {
        return m.cmd.Process.Kill()
    }
    return nil
}

func (m *GoServerManager) URL() string {
    return m.baseURL
}
```

---

## 4. Dual Client for Parallel Requests

```go
// internal/client/dual_client.go
package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

// DualClient makes parallel requests to both servers
type DualClient struct {
    tsURL      string
    goURL      string
    httpClient *http.Client
}

// DualResponse contains responses from both servers
type DualResponse struct {
    TS     *Response
    Go     *Response
    TSErr  error
    GoErr  error
    Timing DualTiming
}

type DualTiming struct {
    TSLatency time.Duration
    GoLatency time.Duration
}

type Response struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
}

func NewDualClient(tsURL, goURL string) *DualClient {
    return &DualClient{
        tsURL: tsURL,
        goURL: goURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// Get performs parallel GET requests
func (c *DualClient) Get(ctx context.Context, path string, opts ...RequestOption) *DualResponse {
    return c.do(ctx, http.MethodGet, path, nil, opts...)
}

// Post performs parallel POST requests
func (c *DualClient) Post(ctx context.Context, path string, body any, opts ...RequestOption) *DualResponse {
    return c.do(ctx, http.MethodPost, path, body, opts...)
}

// Patch performs parallel PATCH requests
func (c *DualClient) Patch(ctx context.Context, path string, body any, opts ...RequestOption) *DualResponse {
    return c.do(ctx, http.MethodPatch, path, body, opts...)
}

// Delete performs parallel DELETE requests
func (c *DualClient) Delete(ctx context.Context, path string, opts ...RequestOption) *DualResponse {
    return c.do(ctx, http.MethodDelete, path, nil, opts...)
}

func (c *DualClient) do(ctx context.Context, method, path string, body any, opts ...RequestOption) *DualResponse {
    var wg sync.WaitGroup
    result := &DualResponse{}

    // Make requests in parallel
    wg.Add(2)

    go func() {
        defer wg.Done()
        start := time.Now()
        result.TS, result.TSErr = c.request(ctx, c.tsURL+path, method, body, opts...)
        result.Timing.TSLatency = time.Since(start)
    }()

    go func() {
        defer wg.Done()
        start := time.Now()
        result.Go, result.GoErr = c.request(ctx, c.goURL+path, method, body, opts...)
        result.Timing.GoLatency = time.Since(start)
    }()

    wg.Wait()
    return result
}

func (c *DualClient) request(ctx context.Context, url, method string, body any, opts ...RequestOption) (*Response, error) {
    var bodyReader io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal body: %w", err)
        }
        bodyReader = bytes.NewReader(jsonBody)
    }

    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    for _, opt := range opts {
        opt(req)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return &Response{
        StatusCode: resp.StatusCode,
        Headers:    resp.Header,
        Body:       respBody,
    }, nil
}
```

---

## 5. Response Comparison Logic

### 5.1 JSON Deep Diff

```go
// internal/compare/json_diff.go
package compare

import (
    "encoding/json"
    "fmt"
    "reflect"
    "sort"
    "strings"
)

// Difference represents a single difference between two JSON values
type Difference struct {
    Path     string      `json:"path"`
    Type     DiffType    `json:"type"`
    TSValue  interface{} `json:"tsValue,omitempty"`
    GoValue  interface{} `json:"goValue,omitempty"`
    Severity Severity    `json:"severity"`
}

type DiffType string

const (
    DiffTypeValueMismatch DiffType = "value_mismatch"
    DiffTypeMissingInTS   DiffType = "missing_in_ts"
    DiffTypeMissingInGo   DiffType = "missing_in_go"
    DiffTypeTypeMismatch  DiffType = "type_mismatch"
)

type Severity string

const (
    SeverityCritical Severity = "critical"
    SeverityWarning  Severity = "warning"
    SeverityInfo     Severity = "info"
)

// CompareJSON compares two JSON responses and returns differences
func CompareJSON(tsBody, goBody []byte, tolerances *Tolerances) ([]Difference, error) {
    var tsData, goData interface{}

    if err := json.Unmarshal(tsBody, &tsData); err != nil {
        return nil, fmt.Errorf("failed to parse TS response: %w", err)
    }

    if err := json.Unmarshal(goBody, &goData); err != nil {
        return nil, fmt.Errorf("failed to parse Go response: %w", err)
    }

    var diffs []Difference
    compareValues("$", tsData, goData, tolerances, &diffs)
    return diffs, nil
}

func compareValues(path string, ts, go_ interface{}, tolerances *Tolerances, diffs *[]Difference) {
    // Check if this path should be ignored
    if tolerances != nil && tolerances.ShouldIgnore(path) {
        return
    }

    // Handle nil cases
    if ts == nil && go_ == nil {
        return
    }
    if ts == nil {
        *diffs = append(*diffs, Difference{
            Path:     path,
            Type:     DiffTypeMissingInTS,
            GoValue:  go_,
            Severity: tolerances.GetSeverity(path, DiffTypeMissingInTS),
        })
        return
    }
    if go_ == nil {
        *diffs = append(*diffs, Difference{
            Path:     path,
            Type:     DiffTypeMissingInGo,
            TSValue:  ts,
            Severity: tolerances.GetSeverity(path, DiffTypeMissingInGo),
        })
        return
    }

    // Check type match
    tsType := reflect.TypeOf(ts)
    goType := reflect.TypeOf(go_)

    if tsType != goType {
        // Special case: numeric types may differ (float64 vs int)
        if isNumeric(ts) && isNumeric(go_) {
            if !tolerances.NumericEqual(ts, go_) {
                *diffs = append(*diffs, Difference{
                    Path:     path,
                    Type:     DiffTypeValueMismatch,
                    TSValue:  ts,
                    GoValue:  go_,
                    Severity: tolerances.GetSeverity(path, DiffTypeValueMismatch),
                })
            }
            return
        }

        *diffs = append(*diffs, Difference{
            Path:     path,
            Type:     DiffTypeTypeMismatch,
            TSValue:  fmt.Sprintf("%T", ts),
            GoValue:  fmt.Sprintf("%T", go_),
            Severity: SeverityCritical,
        })
        return
    }

    // Compare based on type
    switch tsVal := ts.(type) {
    case map[string]interface{}:
        compareObjects(path, tsVal, go_.(map[string]interface{}), tolerances, diffs)
    case []interface{}:
        compareArrays(path, tsVal, go_.([]interface{}), tolerances, diffs)
    default:
        if !tolerances.ValuesEqual(path, ts, go_) {
            *diffs = append(*diffs, Difference{
                Path:     path,
                Type:     DiffTypeValueMismatch,
                TSValue:  ts,
                GoValue:  go_,
                Severity: tolerances.GetSeverity(path, DiffTypeValueMismatch),
            })
        }
    }
}

func compareObjects(path string, ts, go_ map[string]interface{}, tolerances *Tolerances, diffs *[]Difference) {
    // Get all keys from both
    allKeys := make(map[string]bool)
    for k := range ts {
        allKeys[k] = true
    }
    for k := range go_ {
        allKeys[k] = true
    }

    // Sort keys for deterministic output
    keys := make([]string, 0, len(allKeys))
    for k := range allKeys {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    for _, key := range keys {
        keyPath := path + "." + key
        tsVal, tsOk := ts[key]
        goVal, goOk := go_[key]

        if !tsOk {
            if !tolerances.ShouldIgnore(keyPath) {
                *diffs = append(*diffs, Difference{
                    Path:     keyPath,
                    Type:     DiffTypeMissingInTS,
                    GoValue:  goVal,
                    Severity: tolerances.GetSeverity(keyPath, DiffTypeMissingInTS),
                })
            }
            continue
        }
        if !goOk {
            if !tolerances.ShouldIgnore(keyPath) {
                *diffs = append(*diffs, Difference{
                    Path:     keyPath,
                    Type:     DiffTypeMissingInGo,
                    TSValue:  tsVal,
                    Severity: tolerances.GetSeverity(keyPath, DiffTypeMissingInGo),
                })
            }
            continue
        }

        compareValues(keyPath, tsVal, goVal, tolerances, diffs)
    }
}

func compareArrays(path string, ts, go_ []interface{}, tolerances *Tolerances, diffs *[]Difference) {
    maxLen := len(ts)
    if len(go_) > maxLen {
        maxLen = len(go_)
    }

    for i := 0; i < maxLen; i++ {
        elemPath := fmt.Sprintf("%s[%d]", path, i)

        if i >= len(ts) {
            *diffs = append(*diffs, Difference{
                Path:     elemPath,
                Type:     DiffTypeMissingInTS,
                GoValue:  go_[i],
                Severity: tolerances.GetSeverity(elemPath, DiffTypeMissingInTS),
            })
            continue
        }
        if i >= len(go_) {
            *diffs = append(*diffs, Difference{
                Path:     elemPath,
                Type:     DiffTypeMissingInGo,
                TSValue:  ts[i],
                Severity: tolerances.GetSeverity(elemPath, DiffTypeMissingInGo),
            })
            continue
        }

        compareValues(elemPath, ts[i], go_[i], tolerances, diffs)
    }
}
```

### 5.2 Tolerance Configuration

```go
// internal/compare/tolerances.go
package compare

import (
    "regexp"
    "strings"
)

// Tolerances defines acceptable differences between implementations
type Tolerances struct {
    // Paths to completely ignore (e.g., timestamps, random IDs)
    IgnorePaths []string

    // Regex patterns for paths to ignore
    IgnorePatterns []*regexp.Regexp

    // Paths with reduced severity (known differences)
    KnownDifferences map[string]Severity

    // Numeric tolerance for floating point comparisons
    NumericTolerance float64

    // Fields that may have different casing
    CaseInsensitivePaths []string

    // Fields where order doesn't matter in arrays
    UnorderedArrayPaths []string
}

// DefaultTolerances returns sensible defaults for OpenCode comparison
func DefaultTolerances() *Tolerances {
    return &Tolerances{
        IgnorePaths: []string{
            // Timestamps will always differ slightly
            "$.time.created",
            "$.time.updated",
            "$[*].time.created",
            "$[*].time.updated",
            "$.info.time.created",
            "$.info.time.updated",

            // Server-generated IDs may use different formats
            // (but we should compare if they're present in both)

            // Performance metrics
            "$.latency",
            "$.duration",
        },
        IgnorePatterns: []*regexp.Regexp{
            regexp.MustCompile(`^\$\..*\.time\.(created|updated)$`),
            regexp.MustCompile(`^\$\[\d+\]\.time\.(created|updated)$`),
        },
        KnownDifferences: map[string]Severity{
            // Version field format may differ
            "$.version": SeverityInfo,

            // OpenAPI version differences
            "$.openapi": SeverityInfo,
        },
        NumericTolerance: 0.0001,
        CaseInsensitivePaths: []string{
            "$.error.code",
        },
        UnorderedArrayPaths: []string{
            "$.providers",
            "$.tools",
            "$.commands",
        },
    }
}

func (t *Tolerances) ShouldIgnore(path string) bool {
    // Check exact matches
    for _, p := range t.IgnorePaths {
        if matchJSONPath(p, path) {
            return true
        }
    }

    // Check patterns
    for _, pattern := range t.IgnorePatterns {
        if pattern.MatchString(path) {
            return true
        }
    }

    return false
}

func (t *Tolerances) GetSeverity(path string, diffType DiffType) Severity {
    if severity, ok := t.KnownDifferences[path]; ok {
        return severity
    }

    // Default severities
    switch diffType {
    case DiffTypeMissingInGo:
        return SeverityCritical // Go implementation should have all TS features
    case DiffTypeMissingInTS:
        return SeverityWarning // Extra Go features are OK
    case DiffTypeTypeMismatch:
        return SeverityCritical
    case DiffTypeValueMismatch:
        return SeverityWarning
    }

    return SeverityWarning
}

func (t *Tolerances) ValuesEqual(path string, ts, go_ interface{}) bool {
    // Check case insensitive paths
    for _, p := range t.CaseInsensitivePaths {
        if matchJSONPath(p, path) {
            tsStr, tsOk := ts.(string)
            goStr, goOk := go_.(string)
            if tsOk && goOk {
                return strings.EqualFold(tsStr, goStr)
            }
        }
    }

    return ts == go_
}

func (t *Tolerances) NumericEqual(ts, go_ interface{}) bool {
    tsFloat := toFloat64(ts)
    goFloat := toFloat64(go_)

    diff := tsFloat - goFloat
    if diff < 0 {
        diff = -diff
    }

    return diff <= t.NumericTolerance
}
```

---

## 6. Test Cases

### 6.1 Session Endpoint Tests

```go
// tests/session_test.go
package tests

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/opencode-ai/opencode/comparative-tests/internal/compare"
    "github.com/opencode-ai/opencode/comparative-tests/internal/harness"
)

func TestSessionEndpoints(t *testing.T) {
    ctx := context.Background()
    h, err := harness.NewTestHarness(testConfig)
    require.NoError(t, err)
    defer h.Stop()

    require.NoError(t, h.Start(ctx))
    client := h.Client()
    tolerances := compare.DefaultTolerances()

    t.Run("POST /session - Create Session", func(t *testing.T) {
        resp := client.Post(ctx, "/session", map[string]string{
            "directory": h.WorkDir,
            "title":     "Test Session",
        })

        // Both should succeed
        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, 200, resp.TS.StatusCode)
        assert.Equal(t, 200, resp.Go.StatusCode)

        // Compare responses
        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs, "Critical differences found: %v", criticalDiffs)

        // Verify required fields present in both
        assertFieldExists(t, resp.TS.Body, "id")
        assertFieldExists(t, resp.Go.Body, "id")
        assertFieldExists(t, resp.TS.Body, "title")
        assertFieldExists(t, resp.Go.Body, "title")
    })

    t.Run("GET /session - List Sessions", func(t *testing.T) {
        resp := client.Get(ctx, "/session")

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)

        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs)
    })

    t.Run("GET /session/:id - Get Session", func(t *testing.T) {
        // First create a session
        createResp := client.Post(ctx, "/session", map[string]string{
            "directory": h.WorkDir,
        })
        require.NoError(t, createResp.TSErr)

        tsSession := parseSession(t, createResp.TS.Body)
        goSession := parseSession(t, createResp.Go.Body)

        // Get by ID (use respective IDs)
        tsGetResp := client.GetTS(ctx, "/session/"+tsSession.ID)
        goGetResp := client.GetGo(ctx, "/session/"+goSession.ID)

        require.NoError(t, tsGetResp.Err)
        require.NoError(t, goGetResp.Err)
        assert.Equal(t, 200, tsGetResp.StatusCode)
        assert.Equal(t, 200, goGetResp.StatusCode)
    })

    t.Run("DELETE /session/:id - Delete Session", func(t *testing.T) {
        // Create and delete
        createResp := client.Post(ctx, "/session", map[string]string{
            "directory": h.WorkDir,
        })

        tsSession := parseSession(t, createResp.TS.Body)
        goSession := parseSession(t, createResp.Go.Body)

        tsDelResp := client.DeleteTS(ctx, "/session/"+tsSession.ID)
        goDelResp := client.DeleteGo(ctx, "/session/"+goSession.ID)

        assert.Equal(t, tsDelResp.StatusCode, goDelResp.StatusCode)
    })

    t.Run("GET /session/:id - 404 for non-existent", func(t *testing.T) {
        resp := client.Get(ctx, "/session/nonexistent-id")

        // Both should return 404
        assert.Equal(t, 404, resp.TS.StatusCode)
        assert.Equal(t, 404, resp.Go.StatusCode)

        // Error response format should match
        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs)
    })
}
```

### 6.2 Config Endpoint Tests

```go
// tests/config_test.go
package tests

func TestConfigEndpoints(t *testing.T) {
    ctx := context.Background()
    h, err := harness.NewTestHarness(testConfig)
    require.NoError(t, err)
    defer h.Stop()
    require.NoError(t, h.Start(ctx))

    client := h.Client()
    tolerances := compare.DefaultTolerances()

    t.Run("GET /config", func(t *testing.T) {
        resp := client.Get(ctx, "/config")

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, 200, resp.TS.StatusCode)
        assert.Equal(t, 200, resp.Go.StatusCode)

        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs)
    })

    t.Run("GET /config/providers", func(t *testing.T) {
        resp := client.Get(ctx, "/config/providers")

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)

        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        // Provider list structure should match
        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs)
    })

    t.Run("PATCH /config", func(t *testing.T) {
        update := map[string]string{
            "model": "openai/gpt-4o-mini",
        }

        resp := client.Patch(ctx, "/config", update)

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)
    })
}
```

### 6.3 File Endpoint Tests

```go
// tests/file_test.go
package tests

func TestFileEndpoints(t *testing.T) {
    ctx := context.Background()
    h, err := harness.NewTestHarness(testConfig)
    require.NoError(t, err)
    defer h.Stop()
    require.NoError(t, h.Start(ctx))

    // Create test files in work directory
    testFile := filepath.Join(h.WorkDir, "test.txt")
    os.WriteFile(testFile, []byte("hello world"), 0644)

    client := h.Client()
    tolerances := compare.DefaultTolerances()

    t.Run("GET /file", func(t *testing.T) {
        resp := client.Get(ctx, "/file", WithQuery("path", h.WorkDir))

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, 200, resp.TS.StatusCode)
        assert.Equal(t, 200, resp.Go.StatusCode)

        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs)
    })

    t.Run("GET /file/content", func(t *testing.T) {
        resp := client.Get(ctx, "/file/content", WithQuery("path", testFile))

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, 200, resp.TS.StatusCode)
        assert.Equal(t, 200, resp.Go.StatusCode)

        // Content should be identical
        tsContent := parseFileContent(t, resp.TS.Body)
        goContent := parseFileContent(t, resp.Go.Body)
        assert.Equal(t, tsContent.Content, goContent.Content)
    })

    t.Run("GET /file/status", func(t *testing.T) {
        resp := client.Get(ctx, "/file/status")

        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)
    })
}
```

### 6.4 SSE Event Tests

```go
// tests/event_test.go
package tests

import (
    "bufio"
    "context"
    "net/http"
    "strings"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSSEEvents(t *testing.T) {
    ctx := context.Background()
    h, err := harness.NewTestHarness(testConfig)
    require.NoError(t, err)
    defer h.Stop()
    require.NoError(t, h.Start(ctx))

    t.Run("GET /event - SSE Stream", func(t *testing.T) {
        // Create session first
        createResp := h.Client().Post(ctx, "/session", map[string]string{
            "directory": h.WorkDir,
        })
        require.NoError(t, createResp.TSErr)
        tsSession := parseSession(t, createResp.TS.Body)
        goSession := parseSession(t, createResp.Go.Body)

        // Connect to SSE streams
        ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()

        tsEvents := make(chan SSEEvent, 10)
        goEvents := make(chan SSEEvent, 10)

        go collectSSEEvents(ctx, h.TSServer.URL()+"/event?sessionID="+tsSession.ID, tsEvents)
        go collectSSEEvents(ctx, h.GoServer.URL()+"/event?sessionID="+goSession.ID, goEvents)

        // Trigger an event by updating the session
        h.Client().Patch(ctx, "/session/"+tsSession.ID, map[string]string{"title": "Updated"})
        h.Client().Patch(ctx, "/session/"+goSession.ID, map[string]string{"title": "Updated"})

        // Wait for events
        time.Sleep(500 * time.Millisecond)

        // Compare event types received
        var tsEventTypes, goEventTypes []string

        for {
            select {
            case e := <-tsEvents:
                tsEventTypes = append(tsEventTypes, e.Type)
            case e := <-goEvents:
                goEventTypes = append(goEventTypes, e.Type)
            default:
                goto compare
            }
        }

    compare:
        // Both should emit session.updated event
        assert.Contains(t, tsEventTypes, "session.updated")
        assert.Contains(t, goEventTypes, "session.updated")
    })
}

type SSEEvent struct {
    Type string
    Data string
}

func collectSSEEvents(ctx context.Context, url string, events chan<- SSEEvent) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Accept", "text/event-stream")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    var currentEvent SSEEvent

    for scanner.Scan() {
        line := scanner.Text()

        if strings.HasPrefix(line, "event:") {
            currentEvent.Type = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
        } else if strings.HasPrefix(line, "data:") {
            currentEvent.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
        } else if line == "" && currentEvent.Type != "" {
            events <- currentEvent
            currentEvent = SSEEvent{}
        }
    }
}
```

### 6.5 Streaming Response Tests

```go
// tests/streaming_test.go
package tests

func TestStreamingResponses(t *testing.T) {
    ctx := context.Background()
    h, err := harness.NewTestHarness(testConfig)
    require.NoError(t, err)
    defer h.Stop()
    require.NoError(t, h.Start(ctx))

    t.Run("POST /session/:id/message - Streaming", func(t *testing.T) {
        // Create sessions
        createResp := h.Client().Post(ctx, "/session", map[string]string{
            "directory": h.WorkDir,
        })

        tsSession := parseSession(t, createResp.TS.Body)
        goSession := parseSession(t, createResp.Go.Body)

        // Send message and collect streaming chunks
        messageReq := map[string]any{
            "content": "Hello, please respond with 'Hi'",
        }

        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()

        tsChunks := collectStreamingChunks(ctx, h.TSServer.URL()+"/session/"+tsSession.ID+"/message", messageReq)
        goChunks := collectStreamingChunks(ctx, h.GoServer.URL()+"/session/"+goSession.ID+"/message", messageReq)

        // Both should produce chunks
        assert.NotEmpty(t, tsChunks, "TS should produce streaming chunks")
        assert.NotEmpty(t, goChunks, "Go should produce streaming chunks")

        // Final chunk should have complete message
        tsFinal := tsChunks[len(tsChunks)-1]
        goFinal := goChunks[len(goChunks)-1]

        assert.NotNil(t, tsFinal.Info)
        assert.NotNil(t, goFinal.Info)

        // Compare final message structure
        tolerances := compare.DefaultTolerances()
        tolerances.IgnorePaths = append(tolerances.IgnorePaths,
            "$.info.id",
            "$.info.sessionID",
            "$.parts[*].id",
            "$.parts[*].text", // Content may differ based on model
        )

        diffs, _ := compare.CompareJSON(
            toJSON(tsFinal),
            toJSON(goFinal),
            tolerances,
        )

        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs, "Streaming response structure should match")
    })
}
```

---

## 7. CI/CD Integration

### 7.1 GitHub Actions Workflow

```yaml
# .github/workflows/comparative-tests.yml
name: Comparative Tests

on:
  push:
    branches: [main]
    paths:
      - 'packages/opencode/**'
      - 'go-opencode/**'
  pull_request:
    paths:
      - 'packages/opencode/**'
      - 'go-opencode/**'
  schedule:
    # Run daily at midnight UTC
    - cron: '0 0 * * *'

jobs:
  comparative-test:
    runs-on: ubuntu-latest
    timeout-minutes: 30

    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Setup Bun
        uses: oven-sh/setup-bun@v1
        with:
          bun-version: latest

      - name: Install TypeScript dependencies
        run: bun install
        working-directory: packages/opencode

      - name: Build Go server
        run: go build -o bin/opencode-server ./cmd/opencode-server
        working-directory: go-opencode

      - name: Run comparative tests
        run: |
          go test -v -timeout 20m ./tests/...
        working-directory: comparative-tests
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          TEST_PROVIDER: openai
          TEST_MODEL: gpt-4o-mini

      - name: Generate comparison report
        if: always()
        run: |
          go run ./cmd/compare --report=html --output=report.html
        working-directory: comparative-tests

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: comparison-report
          path: comparative-tests/report.html

      - name: Comment on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const report = fs.readFileSync('comparative-tests/report.json', 'utf8');
            const results = JSON.parse(report);

            let comment = '## Comparative Test Results\n\n';
            comment += `| Metric | Value |\n|--------|-------|\n`;
            comment += `| Total Tests | ${results.total} |\n`;
            comment += `| Passed | ${results.passed} |\n`;
            comment += `| Failed | ${results.failed} |\n`;
            comment += `| Parity Score | ${results.parityScore}% |\n`;

            if (results.criticalDifferences.length > 0) {
              comment += '\n### Critical Differences\n\n';
              for (const diff of results.criticalDifferences) {
                comment += `- **${diff.endpoint}**: ${diff.description}\n`;
              }
            }

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
```

### 7.2 Makefile

```makefile
# comparative-tests/Makefile
.PHONY: test test-quick test-full report clean setup

# Setup dependencies
setup:
	cd ../packages/opencode && bun install
	cd ../go-opencode && go build -o bin/opencode-server ./cmd/opencode-server

# Run quick comparison (no LLM calls)
test-quick:
	go test -v -short ./tests/...

# Run full comparison (includes LLM streaming)
test-full:
	go test -v -timeout 30m ./tests/...

# Run specific test
test-%:
	go test -v -run $* ./tests/...

# Generate HTML report
report:
	go run ./cmd/compare --report=html --output=report.html
	@echo "Report generated: report.html"

# Generate JSON report
report-json:
	go run ./cmd/compare --report=json --output=report.json

# Clean up
clean:
	rm -f report.html report.json
	rm -rf /tmp/opencode-compare-*

# Watch for changes and rerun tests
watch:
	watchexec -e go,ts -r make test-quick
```

---

## 8. Known Differences and Acceptable Tolerances

### 8.1 Expected Differences

| Category | Path | TS Behavior | Go Behavior | Severity |
|----------|------|-------------|-------------|----------|
| Timestamps | `$.time.*` | Unix ms | Unix ms | Ignore |
| IDs | `$.id` | ULID | ULID | Ignore value |
| OpenAPI Version | `$.openapi` | 3.1.1 | 3.0.0 | Info |
| Provider Order | `$.providers[*]` | Sorted by name | Sorted by ID | Warning |
| Error Format | `$.error.details` | May include stack | No stack | Info |

### 8.2 Acceptable Response Time Variance

- **Read operations:** < 50ms difference
- **Write operations:** < 100ms difference
- **Streaming start:** < 200ms difference
- **LLM responses:** Provider-dependent (ignored)

### 8.3 Feature Parity Requirements

**Critical (must match):**
- HTTP status codes for all endpoints
- Request body validation (400 errors)
- Resource not found (404 errors)
- Response structure for all data types
- SSE event types and ordering

**Warning (should match):**
- Error message wording
- Default values
- Array ordering (where not semantically significant)

**Info (nice to have):**
- Header casing
- Optional field presence
- Numeric precision beyond 4 decimal places

---

## 9. Reporting and Metrics

### 9.1 Parity Score Calculation

```go
// ParityScore calculates overall API compatibility percentage
func (r *Reporter) ParityScore() float64 {
    if r.TotalEndpoints == 0 {
        return 0
    }

    // Weight by severity
    criticalWeight := 1.0
    warningWeight := 0.5
    infoWeight := 0.1

    maxScore := float64(r.TotalEndpoints) * criticalWeight

    deductions := float64(r.CriticalDiffs) * criticalWeight +
                  float64(r.WarningDiffs) * warningWeight +
                  float64(r.InfoDiffs) * infoWeight

    score := ((maxScore - deductions) / maxScore) * 100
    if score < 0 {
        return 0
    }
    return score
}
```

### 9.2 Report Output Format

```json
{
  "generated": "2025-01-15T10:30:00Z",
  "summary": {
    "totalEndpoints": 62,
    "testedEndpoints": 58,
    "passingEndpoints": 55,
    "failingEndpoints": 3,
    "parityScore": 94.5
  },
  "endpoints": [
    {
      "path": "GET /session",
      "status": "pass",
      "differences": [],
      "timing": {
        "tsLatency": "12ms",
        "goLatency": "8ms"
      }
    },
    {
      "path": "POST /session/:id/message",
      "status": "warning",
      "differences": [
        {
          "path": "$.parts[0].metadata.streamIndex",
          "type": "missing_in_go",
          "severity": "warning"
        }
      ]
    }
  ],
  "criticalDifferences": [],
  "recommendations": [
    "Add streamIndex to Go streaming responses for full parity"
  ]
}
```

---

## 10. Implementation Roadmap

### Phase 1: Foundation (Week 1)
- [ ] Set up comparative-tests directory structure
- [ ] Implement test harness for dual-server management
- [ ] Implement dual client for parallel requests
- [ ] Create basic JSON comparison logic

### Phase 2: Core Tests (Week 2)
- [ ] Session CRUD tests
- [ ] Message endpoints tests
- [ ] File endpoints tests
- [ ] Config endpoints tests

### Phase 3: Advanced Tests (Week 3)
- [ ] SSE event comparison
- [ ] Streaming response comparison
- [ ] MCP endpoint tests
- [ ] Permission flow tests

### Phase 4: CI/CD Integration (Week 4)
- [ ] GitHub Actions workflow
- [ ] Automated reporting
- [ ] PR commenting
- [ ] Dashboard metrics

### Phase 5: Maintenance (Ongoing)
- [ ] Add tests for new endpoints
- [ ] Update tolerances as needed
- [ ] Track parity score over time
- [ ] Document known differences

---

## 11. Success Criteria

1. **Parity Score > 95%** for all tested endpoints
2. **Zero critical differences** in production-facing APIs
3. **Automated CI** running on every PR affecting either implementation
4. **Clear documentation** of any intentional differences
5. **Response time parity** within acceptable thresholds

---

## Appendix A: Quick Start

```bash
# Setup
cd comparative-tests
make setup

# Run quick tests (no LLM, ~2 minutes)
make test-quick

# Run full tests (with LLM, ~10 minutes)
export OPENAI_API_KEY=your-key
make test-full

# Generate report
make report
open report.html
```

## Appendix B: Adding New Endpoint Tests

```go
// Example: Adding test for new endpoint GET /foo
func TestFooEndpoint(t *testing.T) {
    ctx := context.Background()
    h, err := harness.NewTestHarness(testConfig)
    require.NoError(t, err)
    defer h.Stop()
    require.NoError(t, h.Start(ctx))

    client := h.Client()
    tolerances := compare.DefaultTolerances()

    t.Run("GET /foo", func(t *testing.T) {
        resp := client.Get(ctx, "/foo")

        // 1. Check both succeed/fail identically
        require.NoError(t, resp.TSErr)
        require.NoError(t, resp.GoErr)
        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)

        // 2. Compare response bodies
        diffs, err := compare.CompareJSON(resp.TS.Body, resp.Go.Body, tolerances)
        require.NoError(t, err)

        // 3. Assert no critical differences
        criticalDiffs := filterBySeverity(diffs, compare.SeverityCritical)
        assert.Empty(t, criticalDiffs)
    })
}
```
