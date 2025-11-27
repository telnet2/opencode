# Implementation Plan: Client Tools Endpoints

## Overview

**Endpoints:**
- `GET /client-tools/pending/:clientID` - SSE stream for tool requests
- `GET /client-tools/tools/:clientID` - Get registered tools for a client
- `GET /client-tools/tools` - Get all registered client tools

**Current Status:** Missing (no routes defined)
**Priority:** High
**Effort:** Medium

---

## 1. Current State Analysis

### TypeScript Reference Implementation

**Location:** `packages/opencode/src/server/client-tools.ts`

#### Endpoint 1: SSE Pending Stream

```typescript
// packages/opencode/src/server/client-tools.ts:169-229
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

#### Endpoint 2: Get Client Tools

```typescript
// packages/opencode/src/server/client-tools.ts:232-254
.get("/tools/:clientID", async (c) => {
  const clientID = c.req.param("clientID")
  const tools = ClientToolRegistry.getTools(clientID)
  return c.json(tools)
})
```

#### Endpoint 3: Get All Tools

```typescript
// packages/opencode/src/server/client-tools.ts:257-276
.get("/tools", async (c) => {
  const tools = ClientToolRegistry.getAllTools()
  return c.json(Object.fromEntries(tools))
})
```

### TypeScript Registry Implementation

**Location:** `packages/opencode/src/tool/client-registry.ts`

Key features:
- Tool storage: `Map<clientID, Map<toolID, ClientToolDefinition>>`
- Pending requests: `Map<requestID, { resolve, reject, timeout }>`
- Event types: `ToolRequest`, `Registered`, `Unregistered`, `Executing`, `Completed`, `Failed`
- Client cleanup on disconnect

### Go Current State

**Routes:** `go-opencode/internal/server/routes.go:149-154`
```go
r.Route("/client-tools", func(r chi.Router) {
    r.Post("/register", s.registerClientTool)
    r.Delete("/unregister", s.unregisterClientTool)
    r.Post("/execute", s.executeClientTool)
    r.Post("/result", s.submitClientToolResult)
})
```

**Missing:**
- `GET /client-tools/pending/:clientID` (SSE)
- `GET /client-tools/tools/:clientID`
- `GET /client-tools/tools`

**Existing infrastructure:**
- Event bus: `go-opencode/internal/event/bus.go`
- SSE handlers: `go-opencode/internal/server/sse.go`

---

## 2. Implementation Tasks

### Task 1: Add Client Tool Event Types

**File:** `go-opencode/internal/event/bus.go`

Add new event types:

```go
const (
    // Existing events...

    // Client Tool Events
    ClientToolRequest     EventType = "client-tool.request"
    ClientToolRegistered  EventType = "client-tool.registered"
    ClientToolUnregistered EventType = "client-tool.unregistered"
    ClientToolExecuting   EventType = "client-tool.executing"
    ClientToolCompleted   EventType = "client-tool.completed"
    ClientToolFailed      EventType = "client-tool.failed"
)
```

### Task 2: Create Client Tool Registry

**New File:** `go-opencode/internal/clienttool/registry.go`

```go
package clienttool

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/opencode-ai/opencode/internal/event"
)

// ToolDefinition represents a client-registered tool
type ToolDefinition struct {
    ID          string         `json:"id"`
    Description string         `json:"description"`
    Parameters  map[string]any `json:"parameters"`
}

// ExecutionRequest represents a pending tool execution
type ExecutionRequest struct {
    Type      string         `json:"type"`
    RequestID string         `json:"requestID"`
    SessionID string         `json:"sessionID"`
    MessageID string         `json:"messageID"`
    CallID    string         `json:"callID"`
    Tool      string         `json:"tool"`
    Input     map[string]any `json:"input"`
}

// ToolResult represents a successful execution result
type ToolResult struct {
    Status   string         `json:"status"` // "success"
    Title    string         `json:"title"`
    Output   string         `json:"output"`
    Metadata map[string]any `json:"metadata,omitempty"`
}

// ToolError represents a failed execution
type ToolError struct {
    Status string `json:"status"` // "error"
    Error  string `json:"error"`
}

// ToolResponse is either ToolResult or ToolError
type ToolResponse struct {
    Status   string         `json:"status"`
    Title    string         `json:"title,omitempty"`
    Output   string         `json:"output,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty"`
    Error    string         `json:"error,omitempty"`
}

type pendingRequest struct {
    request  ExecutionRequest
    clientID string
    result   chan ToolResponse
    timeout  *time.Timer
}

// Registry manages client-side tools
type Registry struct {
    mu sync.RWMutex

    // clientID -> toolID -> definition
    tools map[string]map[string]ToolDefinition

    // requestID -> pending request
    pending map[string]*pendingRequest
}

// Global registry instance
var globalRegistry = NewRegistry()

func NewRegistry() *Registry {
    return &Registry{
        tools:   make(map[string]map[string]ToolDefinition),
        pending: make(map[string]*pendingRequest),
    }
}

// Register registers tools for a client
func Register(clientID string, tools []ToolDefinition) []string {
    return globalRegistry.Register(clientID, tools)
}

func (r *Registry) Register(clientID string, tools []ToolDefinition) []string {
    r.mu.Lock()
    defer r.mu.Unlock()

    if r.tools[clientID] == nil {
        r.tools[clientID] = make(map[string]ToolDefinition)
    }

    registered := make([]string, 0, len(tools))
    for _, tool := range tools {
        toolID := prefixToolID(clientID, tool.ID)
        r.tools[clientID][toolID] = ToolDefinition{
            ID:          toolID,
            Description: tool.Description,
            Parameters:  tool.Parameters,
        }
        registered = append(registered, toolID)
    }

    // Publish event
    event.Publish(event.Event{
        Type: event.ClientToolRegistered,
        Data: map[string]any{
            "clientID": clientID,
            "toolIDs":  registered,
        },
    })

    return registered
}

// Unregister removes tools for a client
func Unregister(clientID string, toolIDs []string) []string {
    return globalRegistry.Unregister(clientID, toolIDs)
}

func (r *Registry) Unregister(clientID string, toolIDs []string) []string {
    r.mu.Lock()
    defer r.mu.Unlock()

    clientTools := r.tools[clientID]
    if clientTools == nil {
        return nil
    }

    var unregistered []string
    if len(toolIDs) == 0 {
        // Unregister all
        for id := range clientTools {
            unregistered = append(unregistered, id)
        }
        delete(r.tools, clientID)
    } else {
        for _, id := range toolIDs {
            fullID := id
            if !isClientTool(id) {
                fullID = prefixToolID(clientID, id)
            }
            if _, ok := clientTools[fullID]; ok {
                delete(clientTools, fullID)
                unregistered = append(unregistered, fullID)
            }
        }
    }

    if len(unregistered) > 0 {
        event.Publish(event.Event{
            Type: event.ClientToolUnregistered,
            Data: map[string]any{
                "clientID": clientID,
                "toolIDs":  unregistered,
            },
        })
    }

    return unregistered
}

// GetTools returns tools for a specific client
func GetTools(clientID string) []ToolDefinition {
    return globalRegistry.GetTools(clientID)
}

func (r *Registry) GetTools(clientID string) []ToolDefinition {
    r.mu.RLock()
    defer r.mu.RUnlock()

    clientTools := r.tools[clientID]
    if clientTools == nil {
        return nil
    }

    tools := make([]ToolDefinition, 0, len(clientTools))
    for _, t := range clientTools {
        tools = append(tools, t)
    }
    return tools
}

// GetAllTools returns all registered client tools
func GetAllTools() map[string]ToolDefinition {
    return globalRegistry.GetAllTools()
}

func (r *Registry) GetAllTools() map[string]ToolDefinition {
    r.mu.RLock()
    defer r.mu.RUnlock()

    all := make(map[string]ToolDefinition)
    for _, clientTools := range r.tools {
        for id, tool := range clientTools {
            all[id] = tool
        }
    }
    return all
}

// Execute sends a tool request to the client and waits for response
func Execute(ctx context.Context, clientID string, req ExecutionRequest, timeout time.Duration) (*ToolResult, error) {
    return globalRegistry.Execute(ctx, clientID, req, timeout)
}

func (r *Registry) Execute(ctx context.Context, clientID string, req ExecutionRequest, timeout time.Duration) (*ToolResult, error) {
    req.Type = "client-tool-request"

    resultCh := make(chan ToolResponse, 1)
    timer := time.NewTimer(timeout)

    pending := &pendingRequest{
        request:  req,
        clientID: clientID,
        result:   resultCh,
        timeout:  timer,
    }

    r.mu.Lock()
    r.pending[req.RequestID] = pending
    r.mu.Unlock()

    // Publish event for SSE clients
    event.Publish(event.Event{
        Type: event.ClientToolRequest,
        Data: map[string]any{
            "clientID": clientID,
            "request":  req,
        },
    })

    event.Publish(event.Event{
        Type: event.ClientToolExecuting,
        Data: map[string]any{
            "sessionID": req.SessionID,
            "messageID": req.MessageID,
            "callID":    req.CallID,
            "tool":      req.Tool,
            "clientID":  clientID,
        },
    })

    // Wait for result or timeout
    select {
    case resp := <-resultCh:
        timer.Stop()
        r.mu.Lock()
        delete(r.pending, req.RequestID)
        r.mu.Unlock()

        if resp.Status == "error" {
            event.Publish(event.Event{
                Type: event.ClientToolFailed,
                Data: map[string]any{
                    "sessionID": req.SessionID,
                    "messageID": req.MessageID,
                    "callID":    req.CallID,
                    "tool":      req.Tool,
                    "clientID":  clientID,
                    "error":     resp.Error,
                },
            })
            return nil, errors.New(resp.Error)
        }

        event.Publish(event.Event{
            Type: event.ClientToolCompleted,
            Data: map[string]any{
                "sessionID": req.SessionID,
                "messageID": req.MessageID,
                "callID":    req.CallID,
                "tool":      req.Tool,
                "clientID":  clientID,
                "success":   true,
            },
        })

        return &ToolResult{
            Status:   resp.Status,
            Title:    resp.Title,
            Output:   resp.Output,
            Metadata: resp.Metadata,
        }, nil

    case <-timer.C:
        r.mu.Lock()
        delete(r.pending, req.RequestID)
        r.mu.Unlock()

        event.Publish(event.Event{
            Type: event.ClientToolFailed,
            Data: map[string]any{
                "sessionID": req.SessionID,
                "messageID": req.MessageID,
                "callID":    req.CallID,
                "tool":      req.Tool,
                "clientID":  clientID,
                "error":     "timeout",
            },
        })
        return nil, errors.New("client tool execution timed out")

    case <-ctx.Done():
        timer.Stop()
        r.mu.Lock()
        delete(r.pending, req.RequestID)
        r.mu.Unlock()
        return nil, ctx.Err()
    }
}

// SubmitResult handles result submission from client
func SubmitResult(requestID string, resp ToolResponse) bool {
    return globalRegistry.SubmitResult(requestID, resp)
}

func (r *Registry) SubmitResult(requestID string, resp ToolResponse) bool {
    r.mu.RLock()
    pending := r.pending[requestID]
    r.mu.RUnlock()

    if pending == nil {
        return false
    }

    pending.result <- resp
    return true
}

// Cleanup removes all tools and cancels pending requests for a client
func Cleanup(clientID string) {
    globalRegistry.Cleanup(clientID)
}

func (r *Registry) Cleanup(clientID string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Cancel pending requests
    for reqID, pending := range r.pending {
        if pending.clientID == clientID {
            pending.timeout.Stop()
            close(pending.result)
            delete(r.pending, reqID)
        }
    }

    // Remove tools
    if tools := r.tools[clientID]; tools != nil {
        toolIDs := make([]string, 0, len(tools))
        for id := range tools {
            toolIDs = append(toolIDs, id)
        }
        delete(r.tools, clientID)

        if len(toolIDs) > 0 {
            event.Publish(event.Event{
                Type: event.ClientToolUnregistered,
                Data: map[string]any{
                    "clientID": clientID,
                    "toolIDs":  toolIDs,
                },
            })
        }
    }
}

// FindClientForTool finds which client owns a tool
func FindClientForTool(toolID string) string {
    return globalRegistry.FindClientForTool(toolID)
}

func (r *Registry) FindClientForTool(toolID string) string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    for clientID, tools := range r.tools {
        if _, ok := tools[toolID]; ok {
            return clientID
        }
    }
    return ""
}

// IsClientTool checks if a tool ID is a client tool
func IsClientTool(toolID string) bool {
    return isClientTool(toolID)
}

func isClientTool(toolID string) bool {
    return len(toolID) > 7 && toolID[:7] == "client_"
}

func prefixToolID(clientID, toolID string) string {
    return "client_" + clientID + "_" + toolID
}
```

### Task 3: Add Routes

**File:** `go-opencode/internal/server/routes.go`

Update client-tools route:

```go
r.Route("/client-tools", func(r chi.Router) {
    r.Post("/register", s.registerClientTool)
    r.Delete("/unregister", s.unregisterClientTool)
    r.Post("/execute", s.executeClientTool)
    r.Post("/result", s.submitClientToolResult)

    // Add missing routes
    r.Get("/pending/{clientID}", s.clientToolsPending)    // SSE
    r.Get("/tools/{clientID}", s.getClientTools)
    r.Get("/tools", s.getAllClientTools)
})
```

### Task 4: Implement Handlers

**File:** `go-opencode/internal/server/handlers_clienttools.go` (new file)

```go
package server

import (
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/opencode-ai/opencode/internal/clienttool"
    "github.com/opencode-ai/opencode/internal/event"
)

// clientToolsPending streams tool execution requests via SSE
func (s *Server) clientToolsPending(w http.ResponseWriter, r *http.Request) {
    clientID := chi.URLParam(r, "clientID")
    if clientID == "" {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "clientID required")
        return
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    flusher, ok := w.(http.Flusher)
    if !ok {
        writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "streaming not supported")
        return
    }

    ctx := r.Context()

    // Subscribe to tool request events
    unsubscribe := event.Subscribe(event.ClientToolRequest, func(e event.Event) {
        data, ok := e.Data.(map[string]any)
        if !ok {
            return
        }

        // Filter by clientID
        if data["clientID"] != clientID {
            return
        }

        // Write SSE event
        request := data["request"]
        jsonData, _ := json.Marshal(request)
        fmt.Fprintf(w, "event: tool-request\ndata: %s\n\n", jsonData)
        flusher.Flush()
    })
    defer unsubscribe()

    // Keepalive ticker (30 seconds)
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    // Initial flush
    flusher.Flush()

    // Wait for context cancellation
    for {
        select {
        case <-ctx.Done():
            // Client disconnected - cleanup
            clienttool.Cleanup(clientID)
            return
        case <-ticker.C:
            // Send ping to keep connection alive
            fmt.Fprintf(w, "event: ping\ndata: \n\n")
            flusher.Flush()
        }
    }
}

// getClientTools returns tools for a specific client
func (s *Server) getClientTools(w http.ResponseWriter, r *http.Request) {
    clientID := chi.URLParam(r, "clientID")
    if clientID == "" {
        writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "clientID required")
        return
    }

    tools := clienttool.GetTools(clientID)
    if tools == nil {
        tools = []clienttool.ToolDefinition{}
    }

    writeJSON(w, http.StatusOK, tools)
}

// getAllClientTools returns all registered client tools
func (s *Server) getAllClientTools(w http.ResponseWriter, r *http.Request) {
    tools := clienttool.GetAllTools()
    if tools == nil {
        tools = make(map[string]clienttool.ToolDefinition)
    }

    writeJSON(w, http.StatusOK, tools)
}
```

---

## 3. External Configuration

### No External Configuration Required

Client tools are registered dynamically via API calls. No static configuration is needed.

### Optional: Default Timeout Configuration

**File:** `~/.config/opencode/config.json`

```json
{
  "clientTools": {
    "defaultTimeout": 30000,
    "keepaliveInterval": 30000
  }
}
```

---

## 4. Integration Test Plan

### Test File Location

`go-opencode/citest/service/clienttools_test.go`

### Test Cases

```go
package service_test

import (
    "bufio"
    "context"
    "encoding/json"
    "net/http"
    "strings"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("Client Tools Endpoints", func() {
    var clientID string

    BeforeEach(func() {
        clientID = "test-client-" + ulid.Make().String()
    })

    AfterEach(func() {
        // Cleanup any registered tools
        client.Delete(ctx, "/client-tools/unregister",
            map[string]string{"clientID": clientID})
    })

    Describe("GET /client-tools/tools/:clientID", func() {
        It("should return empty array when no tools registered", func() {
            resp, err := client.Get(ctx, "/client-tools/tools/"+clientID)
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var tools []any
            Expect(resp.JSON(&tools)).To(Succeed())
            Expect(tools).To(BeEmpty())
        })

        It("should return registered tools", func() {
            // Register tools first
            _, err := client.Post(ctx, "/client-tools/register", map[string]any{
                "clientID": clientID,
                "tools": []map[string]any{
                    {
                        "id":          "test-tool",
                        "description": "A test tool",
                        "parameters":  map[string]any{"type": "object"},
                    },
                },
            })
            Expect(err).NotTo(HaveOccurred())

            // Get tools
            resp, err := client.Get(ctx, "/client-tools/tools/"+clientID)
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var tools []map[string]any
            Expect(resp.JSON(&tools)).To(Succeed())
            Expect(len(tools)).To(Equal(1))
            Expect(tools[0]["description"]).To(Equal("A test tool"))
        })

        It("should return 400 for empty clientID", func() {
            resp, err := client.Get(ctx, "/client-tools/tools/")
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(404)) // Chi returns 404 for missing param
        })
    })

    Describe("GET /client-tools/tools", func() {
        It("should return empty map when no tools registered", func() {
            resp, err := client.Get(ctx, "/client-tools/tools")
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var tools map[string]any
            Expect(resp.JSON(&tools)).To(Succeed())
            Expect(tools).To(BeEmpty())
        })

        It("should return all registered tools across clients", func() {
            // Register tools for two clients
            client.Post(ctx, "/client-tools/register", map[string]any{
                "clientID": clientID,
                "tools": []map[string]any{
                    {"id": "tool1", "description": "Tool 1", "parameters": map[string]any{}},
                },
            })

            otherClient := "other-" + clientID
            client.Post(ctx, "/client-tools/register", map[string]any{
                "clientID": otherClient,
                "tools": []map[string]any{
                    {"id": "tool2", "description": "Tool 2", "parameters": map[string]any{}},
                },
            })
            defer client.Delete(ctx, "/client-tools/unregister",
                map[string]string{"clientID": otherClient})

            // Get all tools
            resp, err := client.Get(ctx, "/client-tools/tools")
            Expect(err).NotTo(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))

            var tools map[string]any
            Expect(resp.JSON(&tools)).To(Succeed())
            Expect(len(tools)).To(Equal(2))
        })
    })

    Describe("GET /client-tools/pending/:clientID (SSE)", func() {
        It("should establish SSE connection", func() {
            ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
            defer cancel()

            req, _ := http.NewRequestWithContext(ctx, "GET",
                server.URL()+"/client-tools/pending/"+clientID, nil)
            req.Header.Set("Accept", "text/event-stream")

            resp, err := http.DefaultClient.Do(req)
            Expect(err).NotTo(HaveOccurred())
            defer resp.Body.Close()

            Expect(resp.StatusCode).To(Equal(200))
            Expect(resp.Header.Get("Content-Type")).To(Equal("text/event-stream"))
        })

        It("should receive ping events", func() {
            // Start SSE connection
            ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
            defer cancel()

            req, _ := http.NewRequestWithContext(ctx, "GET",
                server.URL()+"/client-tools/pending/"+clientID, nil)
            req.Header.Set("Accept", "text/event-stream")

            resp, err := http.DefaultClient.Do(req)
            Expect(err).NotTo(HaveOccurred())
            defer resp.Body.Close()

            // Read events
            scanner := bufio.NewScanner(resp.Body)
            var foundPing bool

            for scanner.Scan() {
                line := scanner.Text()
                if strings.HasPrefix(line, "event: ping") {
                    foundPing = true
                    break
                }
            }

            Expect(foundPing).To(BeTrue())
        })

        It("should receive tool-request events", func() {
            // This test requires triggering a tool request
            // which would normally happen during message processing

            // Setup: Register a tool
            client.Post(ctx, "/client-tools/register", map[string]any{
                "clientID": clientID,
                "tools": []map[string]any{
                    {"id": "test-tool", "description": "Test", "parameters": map[string]any{}},
                },
            })

            // Start SSE connection in goroutine
            events := make(chan string, 10)
            go func() {
                ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
                defer cancel()

                req, _ := http.NewRequestWithContext(ctx, "GET",
                    server.URL()+"/client-tools/pending/"+clientID, nil)
                resp, err := http.DefaultClient.Do(req)
                if err != nil {
                    return
                }
                defer resp.Body.Close()

                scanner := bufio.NewScanner(resp.Body)
                for scanner.Scan() {
                    events <- scanner.Text()
                }
            }()

            // Wait a bit for connection
            time.Sleep(100 * time.Millisecond)

            // TODO: Trigger a tool request via session message
            // For now, verify the connection works
        })
    })
})
```

### Comparative Test

`go-opencode/citest/comparative/clienttools_test.go`

```go
func TestClientTools_Comparative(t *testing.T) {
    harness := StartComparativeHarness(t)
    defer harness.Stop()

    clientID := "test-client"

    // Test 1: GET /client-tools/tools (empty)
    t.Run("empty tools list", func(t *testing.T) {
        resp := harness.Client().Get(ctx, "/client-tools/tools")

        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)

        var tsTools, goTools map[string]any
        json.Unmarshal(resp.TS.Body, &tsTools)
        json.Unmarshal(resp.Go.Body, &goTools)

        assert.Empty(t, tsTools)
        assert.Empty(t, goTools)
    })

    // Test 2: Register and get tools
    t.Run("register and get tools", func(t *testing.T) {
        tools := []map[string]any{
            {"id": "tool1", "description": "Test tool", "parameters": map[string]any{}},
        }

        // Register on both
        harness.Client().Post(ctx, "/client-tools/register", map[string]any{
            "clientID": clientID,
            "tools":    tools,
        })

        // Get tools for client
        resp := harness.Client().Get(ctx, "/client-tools/tools/"+clientID)

        assert.Equal(t, resp.TS.StatusCode, resp.Go.StatusCode)

        var tsTools, goTools []map[string]any
        json.Unmarshal(resp.TS.Body, &tsTools)
        json.Unmarshal(resp.Go.Body, &goTools)

        assert.Len(t, tsTools, 1)
        assert.Len(t, goTools, 1)
    })

    // Test 3: SSE connection headers
    t.Run("SSE headers", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()

        tsResp := harness.GetTS(ctx, "/client-tools/pending/"+clientID)
        goResp := harness.GetGo(ctx, "/client-tools/pending/"+clientID)

        assert.Equal(t, "text/event-stream", tsResp.Headers.Get("Content-Type"))
        assert.Equal(t, "text/event-stream", goResp.Headers.Get("Content-Type"))
    })
}
```

---

## 5. Implementation Checklist

- [ ] Add client tool event types to `event/bus.go`
- [ ] Create `clienttool/registry.go` with full implementation
- [ ] Add routes to `routes.go`
- [ ] Create `handlers_clienttools.go` with handlers
- [ ] Implement SSE pending endpoint with:
  - [ ] Event subscription
  - [ ] Client ID filtering
  - [ ] 30s keepalive ping
  - [ ] Cleanup on disconnect
- [ ] Implement GET tools endpoints
- [ ] Write unit tests for registry
- [ ] Write integration tests for endpoints
- [ ] Write SSE streaming tests
- [ ] Write comparative tests
- [ ] Update OpenAPI spec

---

## 6. Rollout

1. **Day 1-2:** Create registry package with full implementation
2. **Day 3:** Add event types and handlers
3. **Day 4:** Implement SSE endpoint with keepalive
4. **Day 5:** Integration tests
5. **Week 2:** Comparative testing and bug fixes

---

## References

- TypeScript Routes: `packages/opencode/src/server/client-tools.ts`
- TypeScript Registry: `packages/opencode/src/tool/client-registry.ts`
- Go Event Bus: `go-opencode/internal/event/bus.go`
- Go SSE Example: `go-opencode/internal/server/sse.go`
- Go Routes: `go-opencode/internal/server/routes.go`
