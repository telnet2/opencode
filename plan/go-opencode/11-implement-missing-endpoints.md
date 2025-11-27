# Phase 11: Implement Missing API Endpoints

## Overview

This plan addresses 8 missing API endpoints identified when comparing `go-opencode` against the OpenAPI specification (`openapi.org.json`). These endpoints are required for full API compatibility with the original TypeScript implementation.

---

## 11.1 Missing Endpoints Summary

| Endpoint | Method | Description | Priority |
|----------|--------|-------------|----------|
| `/project` | GET | List all projects | Medium |
| `/project/current` | GET | Get current project | Medium |
| `/session/{id}/message/{messageID}` | GET | Get specific message with parts | High |
| `/tui/control/next` | GET | Get next TUI request from queue | Low |
| `/tui/control/response` | POST | Submit response to TUI queue | Low |
| `/client-tools/pending/{clientID}` | GET | Stream pending tool requests (SSE) | Medium |
| `/client-tools/tools/{clientID}` | GET | Get tools for specific client | Medium |
| `/client-tools/tools` | GET | Get all registered client tools | Medium |

---

## 11.2 Project Endpoints

### 11.2.1 Project Type Definition

**File: `pkg/types/project.go`**

```go
package types

// Project represents a workspace project
type Project struct {
    ID       string       `json:"id"`
    Worktree string       `json:"worktree"`
    VCS      string       `json:"vcs,omitempty"` // "git" | nil
    Time     ProjectTime  `json:"time"`
}

// ProjectTime contains project timestamps
type ProjectTime struct {
    Created     int64  `json:"created"`
    Initialized *int64 `json:"initialized,omitempty"`
}
```

### 11.2.2 Project Service

**File: `internal/project/service.go`**

```go
package project

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "os"
    "path/filepath"
    "time"

    "github.com/opencode-ai/opencode/pkg/types"
)

// Service manages project information
type Service struct {
    workDir string
}

// NewService creates a new project service
func NewService(workDir string) *Service {
    return &Service{workDir: workDir}
}

// List returns all projects (currently just the current project)
func (s *Service) List(ctx context.Context) ([]types.Project, error) {
    current, err := s.Current(ctx)
    if err != nil {
        return nil, err
    }
    return []types.Project{*current}, nil
}

// Current returns the current project based on workDir
func (s *Service) Current(ctx context.Context) (*types.Project, error) {
    absPath, err := filepath.Abs(s.workDir)
    if err != nil {
        return nil, err
    }

    // Generate ID from path hash
    hash := sha256.Sum256([]byte(absPath))
    id := hex.EncodeToString(hash[:])[:16]

    // Check for VCS
    var vcs string
    if _, err := os.Stat(filepath.Join(absPath, ".git")); err == nil {
        vcs = "git"
    }

    // Get directory creation time (or use current time as fallback)
    info, _ := os.Stat(absPath)
    created := time.Now().UnixMilli()
    if info != nil {
        created = info.ModTime().UnixMilli()
    }

    return &types.Project{
        ID:       id,
        Worktree: absPath,
        VCS:      vcs,
        Time: types.ProjectTime{
            Created: created,
        },
    }, nil
}
```

### 11.2.3 Project Handlers

**File: `internal/server/handlers_project.go`**

```go
package server

import (
    "encoding/json"
    "net/http"
)

// listProjects handles GET /project
func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
    projects, err := s.projectService.List(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(projects)
}

// getCurrentProject handles GET /project/current
func (s *Server) getCurrentProject(w http.ResponseWriter, r *http.Request) {
    project, err := s.projectService.Current(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(project)
}
```

---

## 11.3 Session Message Endpoint

### 11.3.1 Get Single Message Handler

**File: `internal/server/handlers_session.go`** (add to existing file)

```go
// getMessage handles GET /session/{sessionID}/message/{messageID}
func (s *Server) getMessage(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionID")
    messageID := chi.URLParam(r, "messageID")

    if sessionID == "" || messageID == "" {
        writeError(w, http.StatusBadRequest, "missing sessionID or messageID")
        return
    }

    // Verify session exists
    session, err := s.sessionService.Get(r.Context(), sessionID)
    if err != nil {
        writeError(w, http.StatusNotFound, "session not found")
        return
    }

    // Get message
    message, parts, err := s.sessionService.GetMessage(r.Context(), sessionID, messageID)
    if err != nil {
        writeError(w, http.StatusNotFound, "message not found")
        return
    }

    response := map[string]any{
        "info":  message,
        "parts": parts,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 11.3.2 Session Service Extension

**File: `internal/session/service.go`** (add method)

```go
// GetMessage retrieves a specific message and its parts
func (s *Service) GetMessage(ctx context.Context, sessionID, messageID string) (*types.Message, []types.Part, error) {
    // First verify session exists
    _, err := s.Get(ctx, sessionID)
    if err != nil {
        return nil, nil, err
    }

    // Get message from storage
    message, err := s.storage.GetMessage(ctx, messageID)
    if err != nil {
        return nil, nil, fmt.Errorf("message not found: %s", messageID)
    }

    // Verify message belongs to session
    if message.SessionID != sessionID {
        return nil, nil, fmt.Errorf("message does not belong to session")
    }

    // Get parts
    parts, err := s.storage.GetParts(ctx, messageID)
    if err != nil {
        return nil, nil, err
    }

    return message, parts, nil
}
```

---

## 11.4 TUI Control Endpoints

### 11.4.1 TUI Control Queue

**File: `internal/tui/control.go`**

```go
package tui

import (
    "context"
    "sync"
    "time"
)

// Request represents a TUI control request
type Request struct {
    ID   string `json:"id"`
    Path string `json:"path"`
    Body any    `json:"body"`
}

// Response represents a response to a TUI request
type Response struct {
    ID     string `json:"id"`
    Result any    `json:"result"`
    Error  string `json:"error,omitempty"`
}

// ControlQueue manages TUI control requests
type ControlQueue struct {
    mu        sync.Mutex
    requests  []Request
    responses map[string]chan Response
    waiters   []chan Request
}

// NewControlQueue creates a new control queue
func NewControlQueue() *ControlQueue {
    return &ControlQueue{
        responses: make(map[string]chan Response),
    }
}

// Push adds a request to the queue
func (q *ControlQueue) Push(req Request) {
    q.mu.Lock()
    defer q.mu.Unlock()

    // If there are waiters, deliver immediately
    if len(q.waiters) > 0 {
        waiter := q.waiters[0]
        q.waiters = q.waiters[1:]
        waiter <- req
        return
    }

    q.requests = append(q.requests, req)
}

// Next returns the next request (blocks until available or context cancelled)
func (q *ControlQueue) Next(ctx context.Context) (*Request, error) {
    q.mu.Lock()

    // Check if there's already a request
    if len(q.requests) > 0 {
        req := q.requests[0]
        q.requests = q.requests[1:]
        q.mu.Unlock()
        return &req, nil
    }

    // Create a waiter channel
    waiter := make(chan Request, 1)
    q.waiters = append(q.waiters, waiter)
    q.mu.Unlock()

    select {
    case req := <-waiter:
        return &req, nil
    case <-ctx.Done():
        // Remove waiter on cancellation
        q.mu.Lock()
        for i, w := range q.waiters {
            if w == waiter {
                q.waiters = append(q.waiters[:i], q.waiters[i+1:]...)
                break
            }
        }
        q.mu.Unlock()
        return nil, ctx.Err()
    }
}

// Respond submits a response for a request
func (q *ControlQueue) Respond(resp Response) bool {
    q.mu.Lock()
    defer q.mu.Unlock()

    ch, ok := q.responses[resp.ID]
    if !ok {
        return false
    }

    ch <- resp
    delete(q.responses, resp.ID)
    return true
}
```

### 11.4.2 TUI Control Handlers

**File: `internal/server/handlers_tui.go`** (add to existing file)

```go
// tuiControlNext handles GET /tui/control/next
func (s *Server) tuiControlNext(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    req, err := s.tuiQueue.Next(ctx)
    if err != nil {
        // Context cancelled, client disconnected
        return
    }

    response := map[string]any{
        "path": req.Path,
        "body": req.Body,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// tuiControlResponse handles POST /tui/control/response
func (s *Server) tuiControlResponse(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ID     string `json:"id"`
        Result any    `json:"result,omitempty"`
        Error  string `json:"error,omitempty"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    success := s.tuiQueue.Respond(tui.Response{
        ID:     req.ID,
        Result: req.Result,
        Error:  req.Error,
    })

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(success)
}
```

---

## 11.5 Client Tools Extended Endpoints

### 11.5.1 Client Tools Service Extensions

**File: `internal/clienttools/service.go`** (extend existing)

```go
// GetPendingExecutions returns an SSE stream of pending tool executions
func (s *Service) GetPendingExecutions(ctx context.Context, clientID string) <-chan ExecutionRequest {
    ch := make(chan ExecutionRequest)

    s.mu.Lock()
    if s.pendingListeners == nil {
        s.pendingListeners = make(map[string][]chan ExecutionRequest)
    }
    s.pendingListeners[clientID] = append(s.pendingListeners[clientID], ch)
    s.mu.Unlock()

    go func() {
        <-ctx.Done()
        s.mu.Lock()
        listeners := s.pendingListeners[clientID]
        for i, l := range listeners {
            if l == ch {
                s.pendingListeners[clientID] = append(listeners[:i], listeners[i+1:]...)
                break
            }
        }
        s.mu.Unlock()
        close(ch)
    }()

    return ch
}

// GetClientTools returns tools registered by a specific client
func (s *Service) GetClientTools(clientID string) []ClientToolDefinition {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var tools []ClientToolDefinition
    for _, tool := range s.tools {
        if tool.ClientID == clientID {
            tools = append(tools, tool)
        }
    }
    return tools
}

// GetAllTools returns all registered tools across all clients
func (s *Service) GetAllTools() map[string]ClientToolDefinition {
    s.mu.RLock()
    defer s.mu.RUnlock()

    result := make(map[string]ClientToolDefinition)
    for name, tool := range s.tools {
        result[name] = tool
    }
    return result
}
```

### 11.5.2 Client Tools Handlers

**File: `internal/server/handlers_clienttools.go`** (add to existing file)

```go
// clientToolsPending handles GET /client-tools/pending/{clientID} (SSE)
func (s *Server) clientToolsPending(w http.ResponseWriter, r *http.Request) {
    clientID := chi.URLParam(r, "clientID")
    if clientID == "" {
        writeError(w, http.StatusBadRequest, "missing clientID")
        return
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    ctx := r.Context()
    pendingCh := s.clientToolsService.GetPendingExecutions(ctx, clientID)

    for {
        select {
        case req, ok := <-pendingCh:
            if !ok {
                return
            }

            data, _ := json.Marshal(req)
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()

        case <-ctx.Done():
            return
        }
    }
}

// getClientTools handles GET /client-tools/tools/{clientID}
func (s *Server) getClientTools(w http.ResponseWriter, r *http.Request) {
    clientID := chi.URLParam(r, "clientID")
    if clientID == "" {
        writeError(w, http.StatusBadRequest, "missing clientID")
        return
    }

    tools := s.clientToolsService.GetClientTools(clientID)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tools)
}

// getAllClientTools handles GET /client-tools/tools
func (s *Server) getAllClientTools(w http.ResponseWriter, r *http.Request) {
    tools := s.clientToolsService.GetAllTools()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tools)
}
```

---

## 11.6 Route Registration

**File: `internal/server/routes.go`** (update)

```go
func (s *Server) setupRoutes() {
    r := s.router

    // Project routes (NEW)
    r.Route("/project", func(r chi.Router) {
        r.Get("/", s.listProjects)
        r.Get("/current", s.getCurrentProject)
    })

    // Session routes
    r.Route("/session", func(r chi.Router) {
        r.Get("/", s.listSessions)
        r.Post("/", s.createSession)
        r.Get("/status", s.getSessionStatus)

        r.Route("/{sessionID}", func(r chi.Router) {
            r.Get("/", s.getSession)
            r.Patch("/", s.updateSession)
            r.Delete("/", s.deleteSession)

            // Messages
            r.Get("/message", s.getMessages)
            r.Post("/message", s.sendMessage)
            r.Get("/message/{messageID}", s.getMessage)  // NEW

            // ... rest of session routes
        })
    })

    // TUI control
    r.Route("/tui", func(r chi.Router) {
        // Existing routes...
        r.Post("/append-prompt", s.tuiAppendPrompt)
        // ...

        // NEW control routes
        r.Route("/control", func(r chi.Router) {
            r.Get("/next", s.tuiControlNext)
            r.Post("/response", s.tuiControlResponse)
        })
    })

    // Client tools
    r.Route("/client-tools", func(r chi.Router) {
        r.Post("/register", s.registerClientTool)
        r.Delete("/unregister", s.unregisterClientTool)
        r.Post("/execute", s.executeClientTool)
        r.Post("/result", s.submitClientToolResult)

        // NEW routes
        r.Get("/tools", s.getAllClientTools)
        r.Get("/tools/{clientID}", s.getClientTools)
        r.Get("/pending/{clientID}", s.clientToolsPending)
    })

    // ... rest of routes
}
```

---

## 11.7 Server Initialization

**File: `internal/server/server.go`** (update)

```go
type Server struct {
    router             *chi.Mux
    sessionService     *session.Service
    projectService     *project.Service      // NEW
    clientToolsService *clienttools.Service
    tuiQueue           *tui.ControlQueue     // NEW
    // ... other fields
}

func NewServer(cfg *Config) *Server {
    s := &Server{
        router:             chi.NewRouter(),
        sessionService:     session.NewService(cfg.Storage),
        projectService:     project.NewService(cfg.WorkDir),  // NEW
        clientToolsService: clienttools.NewService(),
        tuiQueue:           tui.NewControlQueue(),            // NEW
        // ... other fields
    }

    s.setupMiddleware()
    s.setupRoutes()

    return s
}
```

---

## 11.8 Files to Create/Modify

### New Files

| File | Lines (Est.) | Description |
|------|--------------|-------------|
| `pkg/types/project.go` | 20 | Project type definition |
| `internal/project/service.go` | 80 | Project service implementation |
| `internal/server/handlers_project.go` | 50 | Project HTTP handlers |
| `internal/tui/control.go` | 120 | TUI control queue |

### Modified Files

| File | Changes |
|------|---------|
| `internal/server/routes.go` | Add 8 new routes |
| `internal/server/server.go` | Add projectService, tuiQueue |
| `internal/server/handlers_session.go` | Add getMessage handler |
| `internal/server/handlers_tui.go` | Add control handlers |
| `internal/server/handlers_clienttools.go` | Add 3 new handlers |
| `internal/session/service.go` | Add GetMessage method |
| `internal/clienttools/service.go` | Add 3 new methods |

---

## 11.9 Testing

### Unit Tests

```go
// internal/project/service_test.go
func TestProjectService_List(t *testing.T)
func TestProjectService_Current(t *testing.T)
func TestProjectService_VCSDetection(t *testing.T)

// internal/tui/control_test.go
func TestControlQueue_PushNext(t *testing.T)
func TestControlQueue_NextBlocking(t *testing.T)
func TestControlQueue_Respond(t *testing.T)
func TestControlQueue_ContextCancellation(t *testing.T)
```

### Integration Tests

```go
// test/integration/endpoints_test.go
func TestEndpoint_ListProjects(t *testing.T)
func TestEndpoint_GetCurrentProject(t *testing.T)
func TestEndpoint_GetMessage(t *testing.T)
func TestEndpoint_TUIControlNext(t *testing.T)
func TestEndpoint_TUIControlResponse(t *testing.T)
func TestEndpoint_ClientToolsPending(t *testing.T)
func TestEndpoint_GetClientTools(t *testing.T)
func TestEndpoint_GetAllClientTools(t *testing.T)
```

---

## 11.10 Implementation Order

1. **Phase A: Project Endpoints** (Low complexity)
   - Create `pkg/types/project.go`
   - Create `internal/project/service.go`
   - Create `internal/server/handlers_project.go`
   - Update routes
   - Write tests

2. **Phase B: Session Message Endpoint** (Low complexity)
   - Add `GetMessage` to session service
   - Add handler
   - Update routes
   - Write tests

3. **Phase C: Client Tools Extended Endpoints** (Medium complexity)
   - Extend clienttools service
   - Add handlers
   - Update routes
   - Write tests (including SSE test)

4. **Phase D: TUI Control Endpoints** (Medium complexity)
   - Create `internal/tui/control.go`
   - Add handlers
   - Update routes
   - Write tests

---

## 11.11 Acceptance Criteria

- [ ] `GET /project` returns array of projects
- [ ] `GET /project/current` returns current project with VCS detection
- [ ] `GET /session/{id}/message/{messageID}` returns message with parts
- [ ] `GET /tui/control/next` blocks and returns next request
- [ ] `POST /tui/control/response` delivers response to waiting caller
- [ ] `GET /client-tools/pending/{clientID}` streams SSE events
- [ ] `GET /client-tools/tools/{clientID}` returns client-specific tools
- [ ] `GET /client-tools/tools` returns all registered tools
- [ ] All new endpoints match OpenAPI spec types
- [ ] Unit test coverage >80% for new code
- [ ] Integration tests pass for all 8 endpoints
