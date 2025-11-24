# OpenCode TUI Client-Server Protocol Specification

**Version:** 1.0.0
**Last Updated:** 2025-11-24

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Transport Layer](#transport-layer)
- [Data Formats](#data-formats)
- [Communication Patterns](#communication-patterns)
- [API Endpoints](#api-endpoints)
- [Event System](#event-system)
- [Error Handling](#error-handling)
- [Security Considerations](#security-considerations)
- [Examples](#examples)

---

## Overview

The OpenCode TUI (Text User Interface) uses a client-server architecture where the TUI client and server run as separate processes and communicate over HTTP. This document specifies the complete protocol for communication between these components.

### Key Characteristics

- **Process Model**: Client and server run in separate processes
- **Transport**: HTTP/1.1 over TCP
- **Data Format**: JSON
- **Event Streaming**: Server-Sent Events (SSE)
- **Communication Model**: Request-Response + Event Streaming + Bidirectional Queue

---

## Architecture

### Process Separation

```
┌─────────────────────┐         HTTP/SSE          ┌─────────────────────┐
│   TUI Client        │◄──────────────────────────►│   HTTP Server       │
│   (UI Process)      │    127.0.0.1:PORT         │   (Backend Process) │
│                     │                            │                     │
│  - Renders UI       │                            │  - Business Logic   │
│  - Handles Input    │                            │  - Session Mgmt     │
│  - Makes HTTP Calls │                            │  - File Operations  │
└─────────────────────┘                            └─────────────────────┘
```

### Startup Sequence

1. **Server Process** starts and listens on `127.0.0.1:<port>`
2. **TUI Client Process** spawns via `attach` command with server URL
3. Client establishes HTTP connection to server
4. Client subscribes to SSE event stream at `/event`
5. Communication begins

**Implementation Reference:**
- Server startup: `packages/opencode/src/server/server.ts:2022-2029`
- Client spawn: `packages/opencode/src/cli/cmd/tui/spawn.ts:27-56`
- Client attach: `packages/opencode/src/cli/cmd/tui/attach.ts`

---

## Transport Layer

### HTTP Server

- **Framework**: Hono (lightweight HTTP framework)
- **Protocol**: HTTP/1.1
- **Host**: `127.0.0.1` (localhost only)
- **Port**: Dynamic (default 0, assigned by OS)
- **Timeout**: Disabled for long-running operations

### Connection Parameters

```typescript
{
  hostname: "127.0.0.1",
  port: number,        // Dynamically assigned
  idleTimeout: 0       // No timeout
}
```

### HTTP Headers

#### Request Headers

| Header | Required | Description | Example |
|--------|----------|-------------|---------|
| `Content-Type` | Yes (POST/PUT/PATCH) | Request body format | `application/json` |
| `x-opencode-directory` | No | Working directory for operation | `/path/to/project` |
| `Accept` | No | Expected response format | `application/json` |

#### Response Headers

| Header | Description | Example |
|--------|-------------|---------|
| `Content-Type` | Response body format | `application/json` or `text/event-stream` |
| `Access-Control-Allow-Origin` | CORS header | `*` |

---

## Data Formats

### JSON Schema

All request and response bodies use JSON format with schema validation via Zod.

#### Standard Response Format

```json
{
  "success": true,
  "data": { /* response data */ }
}
```

#### Error Response Format

```json
{
  "success": false,
  "name": "ErrorName",
  "data": {
    "message": "Error description",
    /* additional error-specific fields */
  },
  "errors": [
    {
      "field": "error details"
    }
  ]
}
```

### Common Data Types

#### Session Info

```typescript
{
  id: string,
  title: string,
  agent: string,
  time: {
    created: number,
    updated: number
  },
  parent?: string,
  shared?: {
    url: string
  }
}
```

#### Message Info

```typescript
{
  id: string,
  sessionID: string,
  role: "user" | "assistant" | "system",
  time: {
    created: number,
    updated: number
  },
  status: "pending" | "streaming" | "completed" | "error",
  agent?: string
}
```

#### Event Payload

```typescript
{
  type: string,
  properties: {
    /* event-specific data */
  }
}
```

---

## Communication Patterns

### 1. Request-Response Pattern

Standard HTTP request-response for synchronous operations.

```
Client                                    Server
  │                                         │
  │──── POST /session ────────────────────► │
  │      { "title": "New Session" }         │
  │                                         │
  │◄─── 200 OK ───────────────────────────  │
  │      { "id": "abc123", ... }            │
  │                                         │
```

### 2. Server-Sent Events (SSE) Pattern

Unidirectional event streaming from server to client.

```
Client                                    Server
  │                                         │
  │──── GET /event ────────────────────────►│
  │                                         │
  │◄─── SSE Stream ────────────────────────│
  │      data: {"type":"server.connected"}  │
  │◄───────────────────────────────────────│
  │      data: {"type":"session.created"}   │
  │◄───────────────────────────────────────│
  │      data: {"type":"message.updated"}   │
  │      ...                                │
```

### 3. Bidirectional Queue Pattern

For cases where server needs to "call back" to client (e.g., requesting user input).

```
Client                                    Server
  │                                         │
  │──── GET /tui/control/next ────────────►│
  │      (long-poll)                        │
  │                                         │
  │◄─── 200 OK ───────────────────────────│
  │      {                                  │
  │        "path": "/some/endpoint",        │
  │        "body": { ... }                  │
  │      }                                  │
  │                                         │
  │──── POST /tui/control/response ───────►│
  │      { "result": "..." }                │
  │                                         │
  │◄─── 200 OK ───────────────────────────│
  │      true                               │
```

**Implementation Reference:**
- Queue mechanism: `packages/opencode/src/server/tui.ts:13-23`
- AsyncQueue implementation: `packages/opencode/src/util/queue.ts`

### 4. Streaming Pattern (AI Response Generation)

For AI response generation, streaming works through **SSE events, not HTTP response streaming**.

```
Client                                    Server
  │                                         │
  │──── POST /session/abc/message ────────►│
  │      { "text": "Explain this code" }   │
  │      (HTTP request blocks)              │
  │                                         │
  │◄─── SSE: message.updated ─────────────│  (status: streaming)
  │◄─── SSE: message.part.updated ────────│  (text delta: "Let")
  │◄─── SSE: message.part.updated ────────│  (text delta: " me")
  │◄─── SSE: message.part.updated ────────│  (text delta: " explain")
  │◄─── SSE: message.part.updated ────────│  (tool call: Read)
  │◄─── SSE: message.part.updated ────────│  (tool result)
  │◄─── SSE: message.part.updated ────────│  (text delta: "This")
  │◄─── SSE: message.updated ─────────────│  (status: completed)
  │                                         │
  │◄─── 200 OK ───────────────────────────│
  │      { /* complete message */ }         │
```

**Key Points:**
1. Client makes POST request to `/session/:id/message`
2. Server processes AI request using Vercel AI SDK's `streamText()`
3. As AI generates response, server publishes SSE events:
   - `message.part.updated` with `delta` field for text chunks
   - `message.part.updated` for tool calls and results
   - `message.updated` for status changes
4. Client receives real-time updates via existing SSE connection
5. When AI completes, HTTP response returns with final message object

**Why This Design?**
- Allows single SSE connection for all events (not just AI streaming)
- Maintains simple request-response semantics for HTTP API
- Enables multiple clients to observe same session in real-time
- Batches events efficiently (16ms batching window)

**Implementation Reference:**
- Processor: `packages/opencode/src/session/processor.ts:49-328`
- Text delta handling: Line 296-305 (publishes `delta` field)
- Reasoning delta: Line 73-79
- Tool call streaming: Line 97-227

---

## API Endpoints

### Session Management

#### List Sessions

```http
GET /session?directory=/path/to/project
```

**Response:**
```json
[
  {
    "id": "session-id",
    "title": "Session Title",
    "agent": "build",
    "time": {
      "created": 1700000000000,
      "updated": 1700000001000
    }
  }
]
```

#### Get Session

```http
GET /session/:id?directory=/path/to/project
```

**Response:** Single Session object

#### Create Session

```http
POST /session?directory=/path/to/project
Content-Type: application/json

{
  "title": "Optional Title",
  "agent": "build",
  "parent": "optional-parent-id"
}
```

**Response:** Created Session object

#### Update Session

```http
PATCH /session/:id?directory=/path/to/project
Content-Type: application/json

{
  "title": "Updated Title"
}
```

#### Delete Session

```http
DELETE /session/:id?directory=/path/to/project
```

**Response:** `true`

#### Send Message

```http
POST /session/:id/message?directory=/path/to/project
Content-Type: application/json

{
  "text": "User message",
  "agent": "build"
}
```

**Response:** Returns complete assistant message after processing

**⚠️ Important - Streaming Behavior:**
While the HTTP response returns after completion, **real-time streaming updates are delivered via SSE events**. As the AI generates its response:

1. Text deltas are sent via `message.part.updated` events with `delta` field
2. Tool calls are sent as they occur
3. Client receives incremental updates in real-time through the `/event` SSE stream
4. HTTP response waits for full completion, then returns final message

See [Streaming Pattern](#streaming-pattern) for details.

#### Abort Session

```http
POST /session/:id/abort?directory=/path/to/project
```

**Response:** `true`

### TUI-Specific Endpoints

#### Append to Prompt

```http
POST /tui/append-prompt?directory=/path/to/project
Content-Type: application/json

{
  "text": "text to append"
}
```

#### Submit Prompt

```http
POST /tui/submit-prompt?directory=/path/to/project
```

#### Clear Prompt

```http
POST /tui/clear-prompt?directory=/path/to/project
```

#### Show Toast Notification

```http
POST /tui/show-toast?directory=/path/to/project
Content-Type: application/json

{
  "title": "Optional Title",
  "message": "Toast message",
  "variant": "info" | "success" | "warning" | "error",
  "duration": 5000
}
```

#### Execute Command

```http
POST /tui/execute-command?directory=/path/to/project
Content-Type: application/json

{
  "command": "session.new" | "session.list" | "agent.cycle" | ...
}
```

**Available Commands:**
- `session.list` - List all sessions
- `session.new` - Create new session
- `session.share` - Share current session
- `session.interrupt` - Interrupt current session
- `session.compact` - Compact session
- `session.page.up` - Scroll page up
- `session.page.down` - Scroll page down
- `session.half.page.up` - Scroll half page up
- `session.half.page.down` - Scroll half page down
- `session.first` - Go to first message
- `session.last` - Go to last message
- `prompt.clear` - Clear prompt
- `prompt.submit` - Submit prompt
- `agent.cycle` - Cycle through agents

### TUI Control Queue

#### Get Next Request

```http
GET /tui/control/next
```

**Response:**
```json
{
  "path": "/some/path",
  "body": { /* request data */ }
}
```

This endpoint blocks (long-polls) until a request is available.

#### Submit Response

```http
POST /tui/control/response
Content-Type: application/json

{ /* response data */ }
```

**Response:** `true`

### Configuration

#### Get Config

```http
GET /config?directory=/path/to/project
```

#### Update Config

```http
PATCH /config?directory=/path/to/project
Content-Type: application/json

{
  "tui": {
    "theme": "dark",
    "keybinds": { ... }
  }
}
```

### File Operations

#### List Files

```http
GET /file?path=/relative/path&directory=/path/to/project
```

#### Read File

```http
GET /file/content?path=/relative/path&directory=/path/to/project
```

#### Get File Status

```http
GET /file/status?directory=/path/to/project
```

### Search Operations

#### Find Text

```http
GET /find?pattern=search_term&directory=/path/to/project
```

#### Find Files

```http
GET /find/file?query=filename&directory=/path/to/project
```

### Provider Management

#### List Providers

```http
GET /provider?directory=/path/to/project
```

#### Get Provider Auth Methods

```http
GET /provider/auth?directory=/path/to/project
```

---

## Event System

### Event Stream Connection

The client subscribes to server events via SSE:

```http
GET /event?directory=/path/to/project
Accept: text/event-stream
```

The server responds with a continuous stream:

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"type":"server.connected","properties":{}}

data: {"type":"session.created","properties":{"sessionID":"abc123"}}

data: {"type":"message.updated","properties":{"sessionID":"abc123","messageID":"msg1","status":"streaming"}}
```

### Event Types

#### Connection Events

##### server.connected

Sent immediately upon connection.

```json
{
  "type": "server.connected",
  "properties": {}
}
```

#### Installation Events

##### installation.updated

Installation status changed.

```json
{
  "type": "installation.updated",
  "properties": {
    "status": "installing" | "installed" | "error"
  }
}
```

##### installation.update.available

Update available for OpenCode.

```json
{
  "type": "installation.update.available",
  "properties": {
    "version": "1.2.3"
  }
}
```

#### Session Events

##### session.created

New session was created.

```json
{
  "type": "session.created",
  "properties": {
    "sessionID": "string"
  }
}
```

##### session.updated

Session metadata updated.

```json
{
  "type": "session.updated",
  "properties": {
    "sessionID": "string",
    "title": "string"
  }
}
```

##### session.deleted

Session was deleted.

```json
{
  "type": "session.deleted",
  "properties": {
    "sessionID": "string"
  }
}
```

##### session.status

Session status changed.

```json
{
  "type": "session.status",
  "properties": {
    "sessionID": "string",
    "status": "pending" | "running" | "completed" | "error"
  }
}
```

##### session.idle

Session became idle (no active operations).

```json
{
  "type": "session.idle",
  "properties": {
    "sessionID": "string"
  }
}
```

##### session.compacted

Session was compacted (history compressed).

```json
{
  "type": "session.compacted",
  "properties": {
    "sessionID": "string"
  }
}
```

##### session.diff

Session diff calculated.

```json
{
  "type": "session.diff",
  "properties": {
    "sessionID": "string",
    "diff": [/* file diffs */]
  }
}
```

##### session.error

Session encountered an error.

```json
{
  "type": "session.error",
  "properties": {
    "sessionID": "string",
    "error": "string"
  }
}
```

#### Message Events

##### message.updated

Message was created or updated.

```json
{
  "type": "message.updated",
  "properties": {
    "sessionID": "string",
    "messageID": "string",
    "status": "pending" | "streaming" | "completed" | "error"
  }
}
```

##### message.removed

Message was deleted.

```json
{
  "type": "message.removed",
  "properties": {
    "sessionID": "string",
    "messageID": "string"
  }
}
```

##### message.part.updated

Message part (tool call, text block, etc.) updated.

**For streaming text/reasoning**, includes `delta` field with incremental text chunk.

```json
{
  "type": "message.part.updated",
  "properties": {
    "part": {
      "id": "string",
      "sessionID": "string",
      "messageID": "string",
      "type": "text" | "reasoning" | "tool" | ...,
      "text": "accumulated text so far",
      // ... other part-specific fields
    },
    "delta": "incremental text chunk"  // Only present during streaming
  }
}
```

**Streaming vs Non-Streaming:**
- **With `delta`**: Real-time text generation (e.g., `delta: " me"`)
- **Without `delta`**: Part structure update (e.g., tool call status change)

##### message.part.removed

Message part was removed.

```json
{
  "type": "message.part.removed",
  "properties": {
    "sessionID": "string",
    "messageID": "string",
    "partID": "string"
  }
}
```

#### Permission Events

##### permission.updated

Permission request created or updated.

```json
{
  "type": "permission.updated",
  "properties": {
    "sessionID": "string",
    "permissionID": "string",
    "tool": "string",
    "status": "pending" | "approved" | "denied"
  }
}
```

##### permission.replied

Permission request was answered.

```json
{
  "type": "permission.replied",
  "properties": {
    "sessionID": "string",
    "permissionID": "string",
    "response": "allow" | "deny" | "allow_all"
  }
}
```

#### File Events

##### file.edited

File was modified.

```json
{
  "type": "file.edited",
  "properties": {
    "sessionID": "string",
    "messageID": "string",
    "path": "string"
  }
}
```

##### file.watcher.updated

File watcher detected changes.

```json
{
  "type": "file.watcher.updated",
  "properties": {
    "path": "string",
    "event": "create" | "modify" | "delete"
  }
}
```

#### Todo Events

##### todo.updated

Todo list updated.

```json
{
  "type": "todo.updated",
  "properties": {
    "sessionID": "string",
    "todos": [
      {
        "content": "string",
        "status": "pending" | "in_progress" | "completed",
        "activeForm": "string"
      }
    ]
  }
}
```

#### Command Events

##### command.executed

TUI command was executed.

```json
{
  "type": "command.executed",
  "properties": {
    "command": "string"
  }
}
```

#### TUI Events

##### tui.prompt.append

Text appended to TUI prompt.

```json
{
  "type": "tui.prompt.append",
  "properties": {
    "text": "string"
  }
}
```

##### tui.command.execute

TUI command execution requested.

```json
{
  "type": "tui.command.execute",
  "properties": {
    "command": "session.list" | "session.new" | ...
  }
}
```

##### tui.toast.show

Show toast notification in TUI.

```json
{
  "type": "tui.toast.show",
  "properties": {
    "title": "string",
    "message": "string",
    "variant": "info" | "success" | "warning" | "error",
    "duration": 5000
  }
}
```

#### LSP Events

##### lsp.updated

LSP server status changed.

```json
{
  "type": "lsp.updated",
  "properties": {
    "language": "string",
    "status": "starting" | "running" | "stopped" | "error"
  }
}
```

##### lsp.client.diagnostics

LSP diagnostics received.

```json
{
  "type": "lsp.client.diagnostics",
  "properties": {
    "uri": "string",
    "diagnostics": [
      {
        "severity": "error" | "warning" | "info" | "hint",
        "message": "string",
        "range": {
          "start": { "line": 0, "character": 0 },
          "end": { "line": 0, "character": 10 }
        }
      }
    ]
  }
}
```

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | Usage |
|------|---------|-------|
| 200 | OK | Successful operation |
| 400 | Bad Request | Invalid request parameters or body |
| 404 | Not Found | Resource (session, file, etc.) not found |
| 500 | Internal Server Error | Unexpected server error |

### Error Response Structure

All errors follow the NamedError pattern:

```typescript
{
  success: false,
  name: string,        // Error type identifier
  data: {              // Error-specific data
    message: string,
    // ... other fields
  },
  errors?: Array<{     // Validation errors (optional)
    field: string,
    message: string
  }>
}
```

### Error Types

#### UnknownError

Generic error for unexpected conditions.

```json
{
  "name": "UnknownError",
  "data": {
    "message": "An unexpected error occurred"
  }
}
```

#### NotFoundError

Resource not found.

```json
{
  "name": "NotFoundError",
  "data": {
    "resource": "session",
    "id": "abc123"
  }
}
```

#### ModelNotFoundError

Requested AI model not available.

```json
{
  "name": "ModelNotFoundError",
  "data": {
    "provider": "anthropic",
    "model": "claude-3-opus"
  }
}
```

#### ValidationError

Request validation failed.

```json
{
  "name": "ValidationError",
  "data": {
    "message": "Invalid request parameters"
  },
  "errors": [
    {
      "field": "title",
      "message": "Title must be a string"
    }
  ]
}
```

### Error Handling Best Practices

1. **Client Should:**
   - Check HTTP status code first
   - Parse error response for `name` and `data` fields
   - Display user-friendly error messages based on error type
   - Retry failed requests with exponential backoff for network errors
   - Log errors for debugging

2. **Server Will:**
   - Return consistent error format across all endpoints
   - Include stack traces only in development mode
   - Log all errors with context
   - Return appropriate HTTP status codes

### SSE Error Handling

If the SSE connection drops:

1. Client receives connection close event
2. Client should attempt to reconnect with exponential backoff
3. Start with 1s delay, double on each failure, max 30s
4. On reconnect, sync state by fetching current session/message data

---

## Security Considerations

### Network Binding

- Server binds only to `127.0.0.1` (localhost)
- Not accessible from external networks
- No authentication required (local-only access)

### Directory Parameter

- The `directory` query parameter specifies working directory
- Server validates directory exists and is accessible
- Prevents path traversal attacks
- All file operations are scoped to specified directory

### CORS

- CORS enabled with `Access-Control-Allow-Origin: *`
- Safe because server only listens on localhost

### Timeout Configuration

- No idle timeout on connections
- Allows long-running operations
- Client responsible for managing connection lifecycle

---

## Examples

### Complete Session Flow

#### 1. Client Connects

```http
GET /event HTTP/1.1
Host: 127.0.0.1:12345
Accept: text/event-stream
```

Server responds with SSE stream:

```
data: {"type":"server.connected","properties":{}}
```

#### 2. Create Session

```http
POST /session?directory=/home/user/project HTTP/1.1
Host: 127.0.0.1:12345
Content-Type: application/json

{
  "title": "Fix login bug",
  "agent": "build"
}
```

Response:

```json
{
  "id": "ses_abc123",
  "title": "Fix login bug",
  "agent": "build",
  "time": {
    "created": 1700000000000,
    "updated": 1700000000000
  }
}
```

Event emitted:

```
data: {"type":"session.created","properties":{"sessionID":"ses_abc123"}}
```

#### 3. Send Message

```http
POST /session/ses_abc123/message?directory=/home/user/project HTTP/1.1
Host: 127.0.0.1:12345
Content-Type: application/json

{
  "text": "Please analyze the login.ts file and identify the bug",
  "agent": "build"
}
```

Events emitted during processing (showing streaming):

```
data: {"type":"message.updated","properties":{"info":{"id":"msg_user1","status":"completed",...}}}

data: {"type":"message.updated","properties":{"info":{"id":"msg_asst1","status":"streaming",...}}}

data: {"type":"message.part.updated","properties":{"part":{"id":"text1","type":"text","text":"I"},"delta":"I"}}

data: {"type":"message.part.updated","properties":{"part":{"id":"text1","type":"text","text":"I'll"},"delta":"'ll"}}

data: {"type":"message.part.updated","properties":{"part":{"id":"text1","type":"text","text":"I'll analyze"},"delta":" analyze"}}

data: {"type":"message.part.updated","properties":{"part":{"id":"tool1","type":"tool","tool":"Read","state":{"status":"running"}}}}

data: {"type":"message.part.updated","properties":{"part":{"id":"tool1","type":"tool","tool":"Read","state":{"status":"completed","output":"..."}}}}

data: {"type":"message.part.updated","properties":{"part":{"id":"text2","type":"text","text":"The"},"delta":"The"}}

data: {"type":"message.part.updated","properties":{"part":{"id":"text2","type":"text","text":"The bug"},"delta":" bug"}}

data: {"type":"message.updated","properties":{"info":{"id":"msg_asst1","status":"completed",...}}}

data: {"type":"session.idle","properties":{"sessionID":"ses_abc123"}}
```

**Note:** Each `message.part.updated` event during text generation includes:
- `part.text`: Accumulated text so far
- `delta`: Just the new chunk (for efficient rendering)

### TUI Command Example

#### Show Toast Notification

```http
POST /tui/show-toast?directory=/home/user/project HTTP/1.1
Host: 127.0.0.1:12345
Content-Type: application/json

{
  "title": "Success",
  "message": "Session created successfully",
  "variant": "success",
  "duration": 3000
}
```

Event emitted:

```
data: {"type":"tui.toast.show","properties":{"title":"Success","message":"Session created successfully","variant":"success","duration":3000}}
```

### Bidirectional Queue Example

Server needs client to execute a command:

#### Server pushes to queue

```typescript
// Server code
request.push({
  path: "/tui/execute-command",
  body: { command: "session.list" }
})
```

#### Client polls for request

```http
GET /tui/control/next HTTP/1.1
Host: 127.0.0.1:12345
```

Response:

```json
{
  "path": "/tui/execute-command",
  "body": {
    "command": "session.list"
  }
}
```

#### Client executes and responds

```http
POST /tui/control/response HTTP/1.1
Host: 127.0.0.1:12345
Content-Type: application/json

{
  "success": true,
  "result": "Sessions dialog opened"
}
```

---

## Implementation Notes

### Client Implementation

**Location:** `packages/opencode/src/cli/cmd/tui/`

Key files:
- `app.tsx` - Main TUI application
- `context/sdk.tsx` - HTTP client setup
- `attach.ts` - Connection to server

The client uses:
- `@opencode-ai/sdk` for typed API calls
- `fetch()` for HTTP requests
- EventSource API for SSE (handled by SDK)

### Server Implementation

**Location:** `packages/opencode/src/server/`

Key files:
- `server.ts` - Main HTTP server and route definitions
- `tui.ts` - TUI-specific endpoints and queue

The server uses:
- Hono framework for HTTP routing
- Bun.serve() for HTTP server
- AsyncQueue for bidirectional communication
- Zod for request/response validation

### Event Bus Implementation

**Location:** `packages/opencode/src/bus/`

- `index.ts` - Event bus implementation
- `global.ts` - Global event emitter

Events are:
- Type-safe via Zod schemas
- Published to all subscribers
- Streamed to SSE clients
- Batched for performance (16ms batching window)

---

## Version History

- **1.0.0** (2025-11-24) - Initial protocol specification

---

## References

- Hono Documentation: https://hono.dev/
- Server-Sent Events Specification: https://html.spec.whatwg.org/multipage/server-sent-events.html
- Zod Documentation: https://zod.dev/
- OpenCode Repository: https://github.com/anthropics/opencode
