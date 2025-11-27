# Client Tools Design Document

## Overview

This document describes the implementation of client-side tools for the OpenCode Go server. Client tools allow external clients to register custom tools that can be invoked during AI assistant sessions, enabling extensibility beyond the built-in tools.

## Background

The TypeScript implementation provides three endpoints for client tools:
- `GET /client-tools/pending/:clientID` - SSE stream for tool execution requests
- `GET /client-tools/tools/:clientID` - Get registered tools for a specific client
- `GET /client-tools/tools` - Get all registered client tools

The Go implementation currently has placeholders for register/unregister/execute/result but lacks the query endpoints and SSE streaming for pending requests.

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        External Client                           │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │  Register   │    │  SSE Stream │    │   Submit Result     │  │
│  │   Tools     │    │  (pending)  │    │                     │  │
│  └──────┬──────┘    └──────┬──────┘    └──────────┬──────────┘  │
└─────────│──────────────────│───────────────────────│────────────┘
          │                  │                       │
          ▼                  ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                      HTTP Server                                 │
│  POST /register   GET /pending/{id}   POST /result              │
│  DELETE /unregister   GET /tools/{id}   GET /tools              │
└─────────────────────────────────────────────────────────────────┘
          │                  │                       │
          ▼                  ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Client Tool Registry                          │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ tools:   map[clientID]map[toolID]ToolDefinition            │ │
│  │ pending: map[requestID]*pendingRequest                     │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
          │                  │
          ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Event Bus                                  │
│  ClientToolRequest | ClientToolRegistered | ClientToolCompleted │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Tool Registration**: Client registers tools via `POST /client-tools/register`
2. **SSE Connection**: Client connects to `GET /client-tools/pending/{clientID}` to receive execution requests
3. **Execution Request**: When a tool is invoked during session processing:
   - Server publishes `ClientToolRequest` event
   - SSE handler forwards request to connected client
4. **Result Submission**: Client executes tool and submits result via `POST /client-tools/result`
5. **Completion**: Server updates pending request and continues session processing

## API Specification

### GET /client-tools/pending/{clientID}

Server-Sent Events stream for receiving tool execution requests.

**Request:**
- Path param: `clientID` - Unique client identifier

**Response:**
- Content-Type: `text/event-stream`
- Events:
  - `tool-request`: Tool execution request
  - `ping`: Keepalive (every 30 seconds)

**Event Format:**
```
event: tool-request
data: {"type":"client-tool-request","requestID":"...","sessionID":"...","messageID":"...","callID":"...","tool":"...","input":{...}}

event: ping
data:
```

**Lifecycle:**
- Connection cleanup triggers `ClientToolRegistry.Cleanup(clientID)`
- Unregisters all tools and cancels pending requests for that client

### GET /client-tools/tools/{clientID}

Returns tools registered for a specific client.

**Request:**
- Path param: `clientID` - Unique client identifier

**Response:**
```json
[
  {
    "id": "client_clientID_toolName",
    "description": "Tool description",
    "parameters": { "type": "object", ... }
  }
]
```

### GET /client-tools/tools

Returns all registered client tools across all clients.

**Response:**
```json
{
  "client_clientID_tool1": {
    "id": "client_clientID_tool1",
    "description": "...",
    "parameters": {...}
  },
  "client_clientID_tool2": {...}
}
```

## Implementation Details

### New Event Types

Add to `internal/event/bus.go`:

```go
const (
    // Client Tool Events
    ClientToolRequest      EventType = "client-tool.request"
    ClientToolRegistered   EventType = "client-tool.registered"
    ClientToolUnregistered EventType = "client-tool.unregistered"
    ClientToolExecuting    EventType = "client-tool.executing"
    ClientToolCompleted    EventType = "client-tool.completed"
    ClientToolFailed       EventType = "client-tool.failed"
)
```

### New Event Data Types

Add to `internal/event/types.go`:

```go
// ClientToolRequestData is the data for client-tool.request events.
type ClientToolRequestData struct {
    ClientID string                          `json:"clientID"`
    Request  *clienttool.ExecutionRequest    `json:"request"`
}

// ClientToolRegisteredData is the data for client-tool.registered events.
type ClientToolRegisteredData struct {
    ClientID string   `json:"clientID"`
    ToolIDs  []string `json:"toolIDs"`
}

// ClientToolUnregisteredData is the data for client-tool.unregistered events.
type ClientToolUnregisteredData struct {
    ClientID string   `json:"clientID"`
    ToolIDs  []string `json:"toolIDs"`
}

// ClientToolStatusData is the data for client-tool.executing/completed/failed events.
type ClientToolStatusData struct {
    SessionID string `json:"sessionID"`
    MessageID string `json:"messageID"`
    CallID    string `json:"callID"`
    Tool      string `json:"tool"`
    ClientID  string `json:"clientID"`
    Error     string `json:"error,omitempty"`
    Success   bool   `json:"success,omitempty"`
}
```

### Client Tool Registry Package

New package: `internal/clienttool/registry.go`

Key types:
- `ToolDefinition` - Tool metadata (ID, description, parameters)
- `ExecutionRequest` - Request sent to client
- `ToolResponse` - Response from client (success or error)
- `Registry` - Manages tools and pending requests

Key functions:
- `Register(clientID string, tools []ToolDefinition) []string`
- `Unregister(clientID string, toolIDs []string) []string`
- `GetTools(clientID string) []ToolDefinition`
- `GetAllTools() map[string]ToolDefinition`
- `Execute(ctx context.Context, clientID string, req ExecutionRequest, timeout time.Duration) (*ToolResult, error)`
- `SubmitResult(requestID string, resp ToolResponse) bool`
- `Cleanup(clientID string)`
- `FindClientForTool(toolID string) string`
- `IsClientTool(toolID string) bool`

### Tool ID Format

Client tools are prefixed to avoid collisions with built-in tools:
```
client_{clientID}_{toolName}
```

Example: `client_my-client_search-docs`

### SSE Handler

The SSE pending handler follows the same pattern as `globalEvents` and `sessionEvents`:

1. Set SSE headers
2. Create `sseWriter`
3. Subscribe to `ClientToolRequest` events
4. Filter by `clientID`
5. Forward matching events to client
6. Send keepalive pings every 30 seconds
7. Cleanup on disconnect

### Route Registration

Add to `internal/server/routes.go`:

```go
r.Route("/client-tools", func(r chi.Router) {
    r.Post("/register", s.registerClientTool)
    r.Delete("/unregister", s.unregisterClientTool)
    r.Post("/execute", s.executeClientTool)
    r.Post("/result", s.submitClientToolResult)

    // New routes
    r.Get("/pending/{clientID}", s.clientToolsPending)
    r.Get("/tools/{clientID}", s.getClientTools)
    r.Get("/tools", s.getAllClientTools)
})
```

## Testing Strategy

### Unit Tests

Location: `internal/clienttool/registry_test.go`

- Registry CRUD operations
- Tool ID prefixing
- Concurrent access safety
- Cleanup behavior

### Integration Tests

Location: `citest/service/clienttools_test.go`

Test cases:
1. Empty tools list returns empty array
2. Register and retrieve tools for client
3. Get all tools across clients
4. SSE connection establishes with correct headers
5. SSE receives ping events
6. Tools are cleaned up on disconnect

### SSE Tests

The SSE tests use the existing `testutil.SSEClient` helper to:
- Verify Content-Type header
- Collect events over time
- Test connection lifecycle

## Error Handling

| Scenario | HTTP Status | Error Code |
|----------|-------------|------------|
| Missing clientID | 400 | INVALID_REQUEST |
| Streaming not supported | 500 | INTERNAL_ERROR |
| Tool not found | 404 | NOT_FOUND |
| Execution timeout | 504 | TIMEOUT |

## Configuration

Optional configuration in `~/.config/opencode/config.json`:

```json
{
  "clientTools": {
    "defaultTimeout": 30000,
    "keepaliveInterval": 30000
  }
}
```

Default values:
- `defaultTimeout`: 30 seconds (execution timeout)
- `keepaliveInterval`: 30 seconds (SSE ping interval)

## Migration Path

This is a new feature addition with no breaking changes:
1. New event types are additive
2. Existing routes remain unchanged
3. New routes extend the `/client-tools` prefix

## Files Changed

| File | Change |
|------|--------|
| `internal/event/bus.go` | Add ClientTool* event types |
| `internal/event/types.go` | Add ClientTool*Data structs |
| `internal/clienttool/registry.go` | New file - registry implementation |
| `internal/server/routes.go` | Add new routes |
| `internal/server/handlers_clienttools.go` | New file - handler implementations |
| `citest/service/clienttools_test.go` | New file - integration tests |

## References

- TypeScript implementation: `packages/opencode/src/server/client-tools.ts`
- TypeScript registry: `packages/opencode/src/tool/client-registry.ts`
- Go SSE implementation: `go-opencode/internal/server/sse.go`
- Implementation plan: `plan/go-opencode/14-impl-client-tools.md`
