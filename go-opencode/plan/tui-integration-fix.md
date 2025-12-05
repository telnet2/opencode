# TUI Integration Fix

> **Document tracking fixes required to connect the opencode TUI client to the Go server.**

## Summary

| Fix | Issue | Status |
|-----|-------|--------|
| #1 | `/session/status` and `/event` endpoints | ✅ Fixed |
| #2 | Provider data format (`/config/providers`, `/provider`) | ✅ Fixed |
| #3 | MCP status format (`/mcp`) | ✅ Fixed |
| #4 | Session creation with empty body (`POST /session`) | ✅ Fixed |
| #5 | Null vs empty array in responses | ✅ Fixed |
| #6 | `/provider/auth` response format | ✅ Fixed |
| #7 | Null response in `POST /session/{id}/message` | ✅ Fixed |

## Current Status

After all fixes, the TUI can:
- ✅ Connect to the Go server without crashing
- ✅ Display the home screen with model selection
- ✅ Create new sessions
- ⏳ Send prompts (requires message handling implementation)

## Files Modified

- `internal/server/handlers_session.go` - Session status and creation
- `internal/server/handlers_config.go` - Provider and MCP endpoints
- `internal/server/sse.go` - SSE event streaming
- `internal/server/routes.go` - Route configuration

---

## Fix #1: Session Status and Event Endpoints

### Problem

When connecting the opencode TUI client to the Go server (`make serve`), the TUI crashed with:

```
Error: undefined is not an object (evaluating 'status().type')
    at packages/opencode/src/cli/cmd/tui/component/prompt/index.tsx:196:24
```

Server logs showed two endpoints returning 400 errors:
- `GET /session/status` - 400
- `GET /event` - 400

## Root Cause Analysis

### Issue 1: `/session/status` endpoint

**Go server (broken):**
- Required `?sessionID=...` query parameter
- Returned single object: `{"sessionID": "...", "title": "...", "status": "idle"}`

**TypeScript server (expected):**
- No parameters required
- Returns map of all session statuses: `{"sessionID1": {"type": "idle"}, "sessionID2": {"type": "busy"}}`

**TUI behavior:**
- Calls `GET /session/status` without parameters at startup
- Expects `status().type` field (not `status` string)

### Issue 2: `/event` SSE endpoint

**Go server (broken):**
- Required `?sessionID=...` query parameter
- Routed to `sessionEvents` handler

**TypeScript server (expected):**
- No parameters required
- Sends `server.connected` event upon connection
- Subscribes to ALL events

## Fixes Applied

### 1. Fixed `/session/status` (`handlers_session.go`)

```go
// Before
func (s *Server) getSessionStatus(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("sessionID")
    if sessionID == "" {
        writeError(w, http.StatusBadRequest, ...)
        return
    }
    // ... returned wrong format
}

// After
func (s *Server) getSessionStatus(w http.ResponseWriter, r *http.Request) {
    // Return map of sessionID -> status for all non-idle sessions
    // Sessions not in the map are considered idle by the client
    statuses := make(map[string]SessionStatusInfo)
    writeJSON(w, http.StatusOK, statuses)
}
```

### 2. Fixed `/event` SSE endpoint (`sse.go`, `routes.go`)

Added new `allEvents` handler that:
- Does NOT require sessionID
- Sends `server.connected` event first
- Subscribes to all events

Updated route:
```go
r.Get("/event", s.allEvents)  // Was: s.sessionEvents
```

## What's Still Missing: Session Status Tracking

The current fix returns an empty map (all sessions appear "idle"). This is sufficient for basic TUI operation because the TUI defaults to `{type: "idle"}` for missing entries.

### Impact of Missing Status Tracking

| Feature | Current Behavior |
|---------|------------------|
| "Interrupt session" button | Always disabled |
| Busy indicator | Never shows |
| Retry indicator | Never shows |

### Session Status Types

```go
type SessionStatusInfo struct {
    Type    string `json:"type"`              // "idle", "busy", "retry"
    Attempt int    `json:"attempt,omitempty"` // Only for retry
    Message string `json:"message,omitempty"` // Only for retry
    Next    int64  `json:"next,omitempty"`    // Only for retry (timestamp)
}
```

### When to Set Status (from TypeScript implementation)

| Location | Status | When |
|----------|--------|------|
| `processor.ts:53` | `{type: "busy"}` | Starting to process a message |
| `processor.ts:339` | `{type: "retry", attempt, message, next}` | Retrying after error |
| `prompt.ts:228` | `{type: "idle"}` | Done processing |

### Implementation Plan for Status Tracking

1. **Create session status store** (`internal/session/status.go`):
   ```go
   var statusStore = make(map[string]SessionStatusInfo)
   var statusMu sync.RWMutex

   func SetStatus(sessionID string, status SessionStatusInfo) {
       statusMu.Lock()
       defer statusMu.Unlock()
       if status.Type == "idle" {
           delete(statusStore, sessionID)
       } else {
           statusStore[sessionID] = status
       }
       // Publish event for SSE
       event.Publish(event.Event{
           Type: "session.status",
           Data: map[string]any{
               "sessionID": sessionID,
               "status":    status,
           },
       })
   }

   func GetAllStatuses() map[string]SessionStatusInfo {
       statusMu.RLock()
       defer statusMu.RUnlock()
       result := make(map[string]SessionStatusInfo)
       for k, v := range statusStore {
           result[k] = v
       }
       return result
   }
   ```

2. **Update message processing** to call `SetStatus`:
   - Set "busy" when starting message processing
   - Set "retry" on retryable errors
   - Set "idle" when done

3. **Update `/session/status` handler** to use the store:
   ```go
   func (s *Server) getSessionStatus(w http.ResponseWriter, r *http.Request) {
       statuses := session.GetAllStatuses()
       writeJSON(w, http.StatusOK, statuses)
   }
   ```

## Testing

After fixes, run:
```bash
cd go-opencode && make serve
```

Then connect TUI from packages/opencode. The TUI should:
- Connect without crashing
- Display sessions
- Allow basic interaction

The "Interrupt session" feature will work once status tracking is implemented.

---

## Fix #2: Provider Data Format (2024-11-28)

### Problem

After the first fix, TUI crashed with:
```
Error: undefined is not an object (evaluating 'local.model.parsed().model')
    at packages/opencode/src/cli/cmd/tui/component/prompt/index.tsx:715:50
```

### Root Cause

The `/config/providers` and `/provider` endpoints returned wrong data format.

**Go server (broken):**
- `/config/providers` returned `[]ProviderInfo` (just an array)
- `/provider` returned `[]ProviderInfo` (just an array)
- `models` field was an array, not a map

**TypeScript server (expected):**
- `/config/providers` returns `{providers: [...], default: {...}}`
- `/provider` returns `{all: [...], default: {...}, connected: [...]}`
- `models` field must be `Record<string, Model>` (a map/object)

### Fixes Applied

#### 1. Fixed `/config/providers` response format

```go
// Before
func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
    // ... returned []ProviderInfo
}

// After
type ProvidersResponse struct {
    Providers []ProviderInfo    `json:"providers"`
    Default   map[string]string `json:"default"`
}

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
    providers := getDefaultProviders()
    defaultModels := make(map[string]string)
    // ... build defaults
    response := ProvidersResponse{
        Providers: providers,
        Default:   defaultModels,
    }
    writeJSON(w, http.StatusOK, response)
}
```

#### 2. Fixed `/provider` response format

```go
type ProviderListResponse struct {
    All       []ProviderInfo    `json:"all"`
    Default   map[string]string `json:"default"`
    Connected []string          `json:"connected"`
}

func (s *Server) listAllProviders(w http.ResponseWriter, r *http.Request) {
    // ... returns ProviderListResponse
}
```

#### 3. Fixed Provider model structure

Changed `models` from array to map:
```go
type ProviderInfo struct {
    ID     string                   `json:"id"`
    Name   string                   `json:"name"`
    // ...
    Models map[string]ProviderModel `json:"models"` // Map, not array!
}
```

#### 4. Added mock provider data

Added default providers (Anthropic, OpenAI) with models in the correct format to enable TUI to work without actual provider registration.

### What's Still Missing: Dynamic Provider Loading

The current implementation uses hardcoded mock providers. Future work:
- Integrate with models.dev API to fetch real provider/model data
- Support custom provider configuration from config file
- Dynamic provider discovery based on available API keys

---

## Fix #3: MCP Status Format (2024-11-28)

### Problem

After fix #2, TUI crashed with:
```
Error: null is not an object (evaluating 'x.status')
    at packages/opencode/src/cli/cmd/tui/routes/home.tsx:27:51
```

The error occurs in `Object.values(sync.data.mcp).some((x) => x.status === "failed")`.

### Root Cause

The `/mcp` endpoint returned wrong data format.

**Go server (broken):**
```json
{"enabled": false, "servers": []}
```

**TypeScript server (expected):**
```json
{
  "server1": {"status": "connected"},
  "server2": {"status": "failed", "error": "connection refused"}
}
```

The response must be `Record<string, MCPStatus>` - a map from server name to status object.

### Fix Applied

```go
// Before
func (s *Server) getMCPStatus(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{
        "enabled": false,
        "servers": []any{},
    })
}

// After
func (s *Server) getMCPStatus(w http.ResponseWriter, r *http.Request) {
    statuses := make(map[string]MCPServerStatus)
    if s.mcpClient != nil {
        for _, server := range s.mcpClient.Status() {
            status := MCPServerStatus{
                Status: string(server.Status),
            }
            if server.Error != nil {
                status.Error = *server.Error
            }
            statuses[server.Name] = status
        }
    }
    writeJSON(w, http.StatusOK, statuses)  // Returns {} when no MCP servers
}
```

Now returns empty map `{}` instead of an object with wrong structure.

---

## Fix #4: Session Creation with Empty Body (2024-11-28)

### Problem

When sending a prompt, TUI crashed with:
```
Error: undefined is not an object (evaluating 'x.data.id')
```

Server log showed:
```
POST http://localhost:8080/session - 400 67B
```

### Root Cause

The TUI sends `POST /session` with an empty body to create a new session. The Go server's JSON decoder fails on empty body.

### Fix Applied

```go
// Before
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
    var req CreateSessionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, ...)  // Fails on empty body
        return
    }
    // ...
}

// After
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
    var req CreateSessionRequest

    // Body is optional - handle empty body
    if r.ContentLength > 0 {
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeError(w, http.StatusBadRequest, ...)
            return
        }
    }

    // Uses context directory (set by middleware from server working dir)
    directory := req.Directory
    if directory == "" {
        directory = getDirectory(r.Context())
    }
    // ...
}
```

The server middleware already injects the working directory into context, so empty body requests now use that default.

---

## Next Steps

To fully support the TUI, the following features need implementation:

### High Priority (Required for basic chat)

1. **Message handling** (`POST /session/{id}/message`)
   - Accept prompt from TUI
   - Stream response back via SSE
   - Handle tool calls and permissions

2. **Session status tracking**
   - Track "busy"/"idle"/"retry" states
   - Publish status events via SSE
   - Enable "Interrupt session" button

### Medium Priority (Enhanced functionality)

3. **Dynamic provider loading**
   - Fetch providers from models.dev API
   - Support custom provider config
   - Auto-detect available API keys

4. **Tool execution**
   - Implement built-in tools (Read, Write, Bash, etc.)
   - Handle tool permissions
   - Support MCP tool integration

### Low Priority (Nice to have)

5. **Session management**
   - Fork/branch sessions
   - Session summaries
   - Share functionality

---

## Fix #5: Null vs Empty Array in Responses (2024-11-28)

### Problem

TUI showed "Failed to parse JSON" error. Server logs showed 200 OK responses but the TUI couldn't parse them.

### Root Cause

Go nil slices serialize to `null` in JSON, but the TUI expects empty arrays `[]`.

Affected endpoints:
- `GET /session/{id}/diff` - returned `null` (5B)
- `GET /session/{id}/todo` - could return `null`
- `GET /session/{id}/message` - could return `null`

### Fix Applied

Ensure all slice responses return `[]` not `null`:

```go
// getDiff
if diffs == nil {
    diffs = []types.FileDiff{}
}

// getTodo
if todos == nil {
    todos = []map[string]any{}
}

// getMessages - use make() to initialize
result := make([]MessageResponse, 0, len(messages))

// Also ensure nested parts are not null
if parts == nil {
    parts = []types.Part{}
}
```

**Files modified:**
- `internal/server/handlers_session.go` - getDiff, getTodo
- `internal/server/handlers_message.go` - getMessages, getMessage

---

## Fix #6: Provider Auth Response Format (2024-11-28)

### Problem

TUI showed "Failed to parse JSON" error during initial sync.

### Root Cause

The `/provider/auth` endpoint returned wrong format.

**Go server (broken):**
```json
[{"envVar":"ANTHROPIC_API_KEY","provider":"anthropic","type":"api_key"}, ...]
```

**TypeScript server (expected):**
```json
{
  "anthropic": [{"type": "api", "label": "Manually enter API Key"}],
  "openai": [{"type": "oauth", "label": "..."}, {"type": "api", "label": "..."}]
}
```

The response must be `Record<string, AuthMethod[]>` - a map from provider ID to auth methods.

### Fix Applied

```go
// Before - returned array
authMethods := []map[string]any{
    {"provider": "anthropic", "type": "api_key", ...},
}

// After - returns map
type AuthMethod struct {
    Type  string `json:"type"`  // "oauth" or "api"
    Label string `json:"label"`
}

authMethods := map[string][]AuthMethod{
    "anthropic": {{Type: "api", Label: "Manually enter API Key"}},
    "openai":    {{Type: "api", Label: "Manually enter API Key"}},
}
```

---

## Fix #7: Null Response in Message Endpoint (2024-11-28)

### Problem

TUI showed "Failed to parse JSON" error when sending messages.

### Root Cause (Found via mitmproxy traffic capture)

The `POST /session/{id}/message` endpoint was sending TWO JSON responses:

```json
{"info":{...user message...},"parts":[...]}
{"info":null,"parts":null}    <-- PROBLEM!
```

The second line with `null` values caused the JSON parse error.

### Why It Happened

When `ProcessMessage` was called with an uninitialized processor, it returned `nil, nil, nil`. The HTTP handler then blindly encoded this null response.

### Fix Applied

Only send the final assistant message if it's not null - **BOTH in the error case AND success case**:

```go
// Before (error case) - always sent final message (even if null)
if err != nil {
    errResp := MessageResponse{
        Info:  assistantMsg,  // Could be nil!
        Parts: parts,
    }
    encoder.Encode(errResp)
    flusher.Flush()
    return
}

// After (error case) - only send if valid
if err != nil {
    if assistantMsg != nil {
        encoder.Encode(MessageResponse{
            Info:  assistantMsg,
            Parts: parts,
        })
        flusher.Flush()
    }
    return
}

// After (success case) - also check for nil
if assistantMsg != nil {
    encoder.Encode(MessageResponse{
        Info:  assistantMsg,
        Parts: parts,
    })
    flusher.Flush()
}
```

**Important:** The initial fix only covered the success case (lines 160-168) but missed the error case (lines 149-158). The error case was still sending `{"info":null,"parts":null}`.

**File modified:** `internal/server/handlers_message.go`
