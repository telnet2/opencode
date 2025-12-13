# OpenCode Server Endpoints Reference

**Version:** 1.0.0
**Last Updated:** 2025-12-05

This document provides complete specifications for all OpenCode server HTTP endpoints. Use this reference to implement a TUI-compatible OpenCode server.

## Table of Contents

- [Overview](#overview)
- [TUI Connection Sequence](#tui-connection-sequence)
- [Response Formats](#response-formats)
- [Event Streaming (SSE)](#event-streaming-sse)
- [Session Endpoints](#session-endpoints)
- [Message Endpoints](#message-endpoints)
- [Configuration Endpoints](#configuration-endpoints)
- [Provider Endpoints](#provider-endpoints)
- [File Endpoints](#file-endpoints)
- [Search Endpoints](#search-endpoints)
- [MCP Endpoints](#mcp-endpoints)
- [Command Endpoints](#command-endpoints)
- [TUI Control Endpoints](#tui-control-endpoints)
- [Client Tools Endpoints](#client-tools-endpoints)
- [Utility Endpoints](#utility-endpoints)

---

## Overview

### Server Configuration

| Setting | Value | Description |
|---------|-------|-------------|
| Protocol | HTTP/1.1 | Standard HTTP |
| Host | `127.0.0.1` | Localhost only |
| Port | Dynamic (default 8080) | Configured at startup |
| Content-Type | `application/json` | JSON for all endpoints except SSE |

### File References

| Component | File Path |
|-----------|-----------|
| Routes | `go-opencode/internal/server/routes.go` |
| SSE Handler | `go-opencode/internal/server/sse.go` |
| Session Handlers | `go-opencode/internal/server/handlers_session.go` |
| Message Handlers | `go-opencode/internal/server/handlers_message.go` |
| Config Handlers | `go-opencode/internal/server/handlers_config.go` |
| File Handlers | `go-opencode/internal/server/handlers_file.go` |
| TUI Handlers | `go-opencode/internal/server/handlers_tui.go` |

---

## TUI Connection Sequence

When a TUI client connects, endpoints are called in this order:

```
1. GET /event                    ← Establish SSE connection (long-running)
   ↓ (receives server.connected event)
2. GET /config/providers         ← Get available model providers
3. GET /provider                 ← Get provider list with connection status
4. GET /agent                    ← Get available agents
5. GET /config                   ← Get application configuration
6. GET /mcp                      ← Get MCP server status
7. GET /lsp                      ← Get LSP server status
8. GET /command                  ← Get available slash commands
9. GET /session                  ← Get existing sessions
10. GET /formatter               ← Get formatter status
11. GET /provider/auth           ← Get authentication methods
12. GET /session/status          ← Get active session statuses
13. GET /vcs                     ← Get VCS (git) branch info
```

### Creating a Session and Sending a Message

```
1. POST /session                 ← Create new session
   ↓ (receives session.created event via SSE)
2. POST /session/{id}/message    ← Send user message
   ↓ (receives streaming events via SSE)
   ↓ message.updated, message.part.updated, session.status, etc.
3. GET /session/{id}/todo        ← Get session todos
4. GET /session/{id}/diff        ← Get file diffs
5. GET /session/{id}/message     ← Get all messages
6. GET /session/{id}             ← Get session details
```

### Permission Flow

```
1. Tool requests permission      ← permission.updated event via SSE
2. POST /session/{id}/permissions/{permissionID}
   ↓ (grants or denies permission)
   ↓ permission.replied event via SSE
3. Tool continues or aborts
```

---

## Response Formats

### Success Response

```json
{
  "success": true,
  "data": { ... }
}
```

Or for list endpoints, the array is returned directly:
```json
[ ... ]
```

### Error Response

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request or missing required fields |
| `NOT_FOUND` | 404 | Resource not found |
| `INTERNAL_ERROR` | 500 | Server-side error |

### Simple Success Response

For endpoints that only return success/failure:
```json
{
  "success": true
}
```

---

## Event Streaming (SSE)

### GET /event

Establishes an SSE connection for real-time events.

**Response Headers:**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

**Event Format:**
```
event: message
data: {"type":"event.type","properties":{...}}

: heartbeat

```

**Initial Event:**
```json
{"type": "server.connected", "properties": {}}
```

**Heartbeat:** Sent every 30 seconds as a comment (`: heartbeat\n\n`)

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `sessionID` | string | Optional. Filter events to specific session |
| `directory` | string | Optional. Working directory context |

**File Reference:** `go-opencode/internal/server/sse.go:88-153`

### GET /global/event

Global event stream (cross-project events).

Same format as `/event` but without session filtering.

---

## Session Endpoints

### GET /session

List all sessions.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `directory` | string | No | Filter by directory |

**Response:**
```json
[
  {
    "id": "ses_xxxx",
    "projectID": "hash",
    "directory": "/path/to/project",
    "parentID": null,
    "title": "Session Title",
    "version": "local",
    "summary": {
      "additions": 0,
      "deletions": 0,
      "files": 0,
      "diffs": []
    },
    "share": null,
    "time": {
      "created": 1764964062216,
      "updated": 1764964062216
    }
  }
]
```

**Session Object Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Session ID (format: `ses_xxxx`) |
| `projectID` | string | Hash of project directory |
| `directory` | string | Absolute path to working directory |
| `parentID` | string? | Parent session ID (for forks) |
| `title` | string | Session display title |
| `version` | string | Always "local" for local sessions |
| `summary.additions` | number | Total lines added |
| `summary.deletions` | number | Total lines deleted |
| `summary.files` | number | Number of files changed |
| `summary.diffs` | FileDiff[] | Array of file diffs |
| `share` | object? | `{url: string}` if shared |
| `time.created` | number | Unix timestamp (ms) |
| `time.updated` | number | Unix timestamp (ms) |

**File Reference:** `go-opencode/internal/server/handlers_session.go:22-39`

---

### POST /session

Create a new session.

**Request Body:**
```json
{
  "directory": "/path/to/project",
  "title": "Optional Session Title"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `directory` | string | No | Working directory (uses context if omitted) |
| `title` | string | No | Custom title |

**Response:** Session object (same as GET /session item)

**SSE Event:** `session.created` with `{info: Session}`

**File Reference:** `go-opencode/internal/server/handlers_session.go:42-76`

---

### GET /session/{sessionID}

Get a single session.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `sessionID` | string | Session ID |

**Response:** Session object

**File Reference:** `go-opencode/internal/server/handlers_session.go:78-89`

---

### PATCH /session/{sessionID}

Update session metadata.

**Request Body:**
```json
{
  "title": "New Title"
}
```

**Response:** Updated Session object

**SSE Event:** `session.updated` with `{info: Session}`

**File Reference:** `go-opencode/internal/server/handlers_session.go:91-114`

---

### DELETE /session/{sessionID}

Delete a session.

**Response:**
```json
{"success": true}
```

**SSE Event:** `session.deleted` with `{info: Session}`

**File Reference:** `go-opencode/internal/server/handlers_session.go:116-135`

---

### GET /session/status

Get status of all active sessions.

**Response:**
```json
{
  "ses_xxxx": {
    "type": "busy",
    "attempt": 0,
    "message": "",
    "next": 0
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `"idle"`, `"busy"`, or `"retry"` |
| `attempt` | number | Retry attempt count (for retry status) |
| `message` | string | Status message (for retry status) |
| `next` | number | Next retry timestamp (for retry status) |

**Note:** Sessions not in the map are considered idle.

**File Reference:** `go-opencode/internal/server/handlers_session.go:146-156`

---

### GET /session/{sessionID}/children

Get child sessions (forks).

**Response:**
```json
[Session, Session, ...]
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:158-169`

---

### POST /session/{sessionID}/fork

Fork a session from a specific message.

**Request Body:**
```json
{
  "messageID": "msg_xxxx"
}
```

**Response:** New Session object

**SSE Event:** `session.created` with `{info: Session}`

**File Reference:** `go-opencode/internal/server/handlers_session.go:176-199`

---

### POST /session/{sessionID}/abort

Abort the current message processing.

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:201-211`

---

### POST /session/{sessionID}/share

Share a session publicly.

**Response:**
```json
{
  "url": "https://opencode.ai/share/xxx"
}
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:213-224`

---

### DELETE /session/{sessionID}/share

Unshare a session.

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:226-236`

---

### POST /session/{sessionID}/summarize

Trigger session summarization.

**Request Body:**
```json
{
  "providerID": "anthropic",
  "modelID": "claude-sonnet-4-20250514"
}
```

**Response:**
```json
true
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:244-266`

---

### POST /session/{sessionID}/init

Initialize session (returns session info).

**Response:** Session object

**File Reference:** `go-opencode/internal/server/handlers_session.go:268-280`

---

### GET /session/{sessionID}/diff

Get file diffs for the session.

**Response:**
```json
[
  {
    "file": "/path/to/file.go",
    "additions": 10,
    "deletions": 5,
    "before": "original content...",
    "after": "new content..."
  }
]
```

**FileDiff Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `file` | string | Absolute file path |
| `additions` | number | Lines added |
| `deletions` | number | Lines deleted |
| `before` | string? | Original content (if available) |
| `after` | string? | New content (if available) |

**File Reference:** `go-opencode/internal/server/handlers_session.go:282-298`

---

### GET /session/{sessionID}/todo

Get todo items for the session.

**Response:**
```json
[
  {
    "id": "todo_xxxx",
    "content": "Task description",
    "status": "pending",
    "priority": "medium"
  }
]
```

**Todo Status Values:** `"pending"`, `"in_progress"`, `"completed"`, `"cancelled"`

**File Reference:** `go-opencode/internal/server/handlers_session.go:300-316`

---

### POST /session/{sessionID}/revert

Revert session to a previous state.

**Request Body:**
```json
{
  "messageID": "msg_xxxx",
  "partID": "prt_xxxx"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `messageID` | string | Yes | Message to revert to |
| `partID` | string | No | Specific part within message |

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:324-340`

---

### POST /session/{sessionID}/unrevert

Undo a revert operation.

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_session.go:342-352`

---

### POST /session/{sessionID}/command

Send a slash command.

**Request Body:**
```json
{
  "command": "/review uncommitted"
}
```

**Response:** Command result object

**File Reference:** `go-opencode/internal/server/handlers_session.go:359-376`

---

### POST /session/{sessionID}/shell

Run a shell command.

**Request Body:**
```json
{
  "command": "ls -la",
  "timeout": 30000
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | Yes | Shell command to run |
| `timeout` | number | No | Timeout in milliseconds |

**Response:** Shell execution result

**File Reference:** `go-opencode/internal/server/handlers_session.go:383-401`

---

### POST /session/{sessionID}/permissions/{permissionID}

Respond to a permission request.

**Request Body:**
```json
{
  "granted": true
}
```

**Response:**
```json
{"success": true}
```

**SSE Event:** `permission.replied` with:
```json
{
  "sessionID": "ses_xxxx",
  "permissionID": "per_xxxx",
  "response": "once"
}
```

**Response Values:** `"once"`, `"always"`, `"reject"`

**File Reference:** `go-opencode/internal/server/handlers_session.go:408-441`

---

## Message Endpoints

### GET /session/{sessionID}/message

Get all messages in a session.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | number | Maximum messages to return |

**Response:**
```json
[
  {
    "info": {
      "id": "msg_xxxx",
      "sessionID": "ses_xxxx",
      "role": "user",
      "time": {
        "created": 1764964062240
      },
      "agent": "build",
      "model": {
        "providerID": "anthropic",
        "modelID": "claude-sonnet-4-20250514"
      },
      "summary": {
        "title": "Task title",
        "diffs": []
      }
    },
    "parts": [
      {
        "id": "prt_xxxx",
        "sessionID": "ses_xxxx",
        "messageID": "msg_xxxx",
        "type": "text",
        "text": "Message content"
      }
    ]
  }
]
```

**Message Object Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Message ID (format: `msg_xxxx`) |
| `sessionID` | string | Parent session ID |
| `role` | string | `"user"` or `"assistant"` |
| `time.created` | number | Unix timestamp (ms) |
| `time.updated` | number? | Last update timestamp |
| `agent` | string? | Agent name (user messages) |
| `model` | ModelRef? | Model reference (user messages) |
| `summary` | object? | Summary with title/diffs (user) |
| `parentID` | string? | Parent message ID (assistant) |
| `modelID` | string? | Model ID used (assistant) |
| `providerID` | string? | Provider ID used (assistant) |
| `mode` | string? | Agent mode (assistant) |
| `path` | object? | `{cwd, root}` paths (assistant) |
| `finish` | string? | Finish reason (assistant) |
| `cost` | number | Cost in USD (assistant) |
| `tokens` | TokenUsage? | Token counts (assistant) |
| `error` | MessageError? | Error info if failed |

**TokenUsage Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `input` | number | Input tokens |
| `output` | number | Output tokens |
| `reasoning` | number | Reasoning tokens |
| `cache.read` | number | Cache read tokens |
| `cache.write` | number | Cache write tokens |

**File Reference:** `go-opencode/internal/server/handlers_message.go:245-271`

---

### POST /session/{sessionID}/message

Send a message and get streaming response.

**Request Body:**
```json
{
  "content": "Your prompt here",
  "parts": [
    {"type": "text", "text": "Your prompt here"}
  ],
  "agent": "build",
  "model": {
    "providerID": "anthropic",
    "modelID": "claude-sonnet-4-20250514"
  },
  "tools": {
    "read": true,
    "write": false
  },
  "files": []
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | string | Yes* | Message text (legacy format) |
| `parts` | array | Yes* | SDK format with text parts |
| `agent` | string | No | Agent to use |
| `model` | ModelRef | No | Model to use |
| `tools` | object | No | Tool enable/disable map |
| `files` | FilePart[] | No | Attached files |

*Either `content` or `parts` is required.

**Response Headers:**
```
Content-Type: application/json
Transfer-Encoding: chunked
Cache-Control: no-cache
Connection: keep-alive
```

**Response (final):**
```json
{
  "info": Message,
  "parts": Part[]
}
```

**SSE Events fired during processing:**
1. `message.updated` (user message)
2. `message.part.updated` (user text part)
3. `session.status` (type: "busy")
4. `session.updated`
5. `session.diff` (initial empty)
6. `message.created` (assistant message)
7. `message.part.updated` * N (streaming parts)
8. `message.updated` (assistant complete)
9. `session.status` (type: "idle")
10. `session.idle`

**File Reference:** `go-opencode/internal/server/handlers_message.go:52-243`

---

### GET /session/{sessionID}/message/{messageID}

Get a single message with its parts.

**Response:**
```json
{
  "info": Message,
  "parts": Part[]
}
```

**File Reference:** `go-opencode/internal/server/handlers_message.go:273-294`

---

## Configuration Endpoints

### GET /config

Get application configuration.

**Response:**
```json
{
  "model": "anthropic/claude-sonnet-4-20250514",
  "small_model": "anthropic/claude-3-5-haiku-20241022",
  "keybinds": {
    "leader": ",",
    "submit": "<C-j>",
    "abort": "<C-c>"
  },
  "lsp": {
    "disabled": false
  },
  "mcp": {},
  "provider": {},
  "agent": {}
}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:17-23`

---

### PATCH /config

Update configuration.

**Request Body:**
```json
{
  "model": "anthropic/claude-opus-4-20250514",
  "small_model": "anthropic/claude-3-5-haiku-20241022"
}
```

**Response:** Updated config object

**File Reference:** `go-opencode/internal/server/handlers_config.go:25-42`

---

### GET /config/providers

Get available providers with models.

**Response:**
```json
{
  "providers": [
    {
      "id": "anthropic",
      "name": "Anthropic",
      "env": ["ANTHROPIC_API_KEY"],
      "npm": "@ai-sdk/anthropic",
      "models": {
        "claude-sonnet-4-20250514": {
          "id": "claude-sonnet-4-20250514",
          "name": "Claude Sonnet 4",
          "release_date": "2025-05-14",
          "capabilities": {
            "temperature": true,
            "reasoning": false,
            "attachment": true,
            "toolcall": true,
            "input": {
              "text": true,
              "audio": false,
              "image": true,
              "video": false,
              "pdf": true
            },
            "output": {
              "text": true,
              "audio": false,
              "image": false,
              "video": false,
              "pdf": false
            }
          },
          "cost": {
            "input": 3.0,
            "output": 15.0,
            "cache_read": 0.3,
            "cache_write": 3.75
          },
          "limit": {
            "context": 200000,
            "output": 64000
          },
          "options": {}
        }
      }
    }
  ],
  "default": {
    "anthropic": "claude-sonnet-4-20250514"
  }
}
```

**ProviderModel Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Model ID |
| `name` | string | Display name |
| `release_date` | string | Release date (YYYY-MM-DD) |
| `capabilities` | object | Feature support flags |
| `cost.input` | number | $/M input tokens |
| `cost.output` | number | $/M output tokens |
| `cost.cache_read` | number | $/M cache read tokens |
| `cost.cache_write` | number | $/M cache write tokens |
| `limit.context` | number | Max context length |
| `limit.output` | number | Max output tokens |

**File Reference:** `go-opencode/internal/server/handlers_config.go:211-229`

---

## Provider Endpoints

### GET /provider

Get all providers with connection status.

**Response:**
```json
{
  "all": [ProviderInfo],
  "default": {
    "anthropic": "claude-sonnet-4-20250514"
  },
  "connected": ["anthropic"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `all` | ProviderInfo[] | All available providers |
| `default` | object | Default model per provider |
| `connected` | string[] | Provider IDs with API keys set |

**File Reference:** `go-opencode/internal/server/handlers_config.go:238-269`

---

### GET /provider/auth

Get authentication methods for providers.

**Response:**
```json
{
  "anthropic": [
    {"type": "api", "label": "Manually enter API Key"}
  ],
  "openai": [
    {"type": "api", "label": "Manually enter API Key"}
  ]
}
```

**AuthMethod Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `"oauth"` or `"api"` |
| `label` | string | Display label |

**File Reference:** `go-opencode/internal/server/handlers_config.go:282-296`

---

### PUT /auth/{providerID}

Set API key for a provider.

**Request Body:**
```json
{
  "apiKey": "sk-..."
}
```

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:309-330`

---

## File Endpoints

### GET /file

List files in a directory.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `path` | string | Directory path (uses context if omitted) |

**Response:**
```json
{
  "files": [
    {
      "name": "file.go",
      "isDirectory": false,
      "size": 1234
    }
  ]
}
```

**File Reference:** `go-opencode/internal/server/handlers_file.go:24-51`

---

### GET /file/content

Read file contents.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | File path |
| `offset` | number | No | Line offset |
| `limit` | number | No | Max lines (default 2000) |

**Response:**
```json
{
  "content": "file contents...",
  "lines": 150,
  "truncated": false
}
```

**File Reference:** `go-opencode/internal/server/handlers_file.go:53-94`

---

### GET /file/status

Get git status.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `directory` | string | Directory to check |

**Response:**
```json
{
  "branch": "main",
  "staged": ["file1.go"],
  "unstaged": ["file2.go"],
  "untracked": ["file3.go"]
}
```

**File Reference:** `go-opencode/internal/server/handlers_file.go:96-137`

---

### GET /vcs

Get VCS (git) branch info.

**Response:**
```json
{
  "branch": "main"
}
```

**File Reference:** `go-opencode/internal/server/handlers_file.go:262-279`

---

## Search Endpoints

### GET /find

Search text in files using ripgrep.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pattern` | string | Yes | Search regex pattern |
| `path` | string | No | Directory to search |
| `include` | string | No | Glob pattern filter |

**Response:**
```json
{
  "matches": [
    {
      "file": "/path/to/file.go",
      "line": 42,
      "content": "matching line content"
    }
  ],
  "count": 1,
  "truncated": false
}
```

**Note:** Results limited to 100 matches.

**File Reference:** `go-opencode/internal/server/handlers_file.go:139-208`

---

### GET /find/file

Search for files by pattern.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pattern` | string | Yes | Glob pattern |
| `path` | string | No | Directory to search |

**Response:**
```json
{
  "files": ["/path/to/file.go"],
  "count": 1
}
```

**Note:** Results limited to 100 files.

**File Reference:** `go-opencode/internal/server/handlers_file.go:210-247`

---

### GET /find/symbol

Search for code symbols via LSP.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Symbol search query |

**Response:**
```json
[
  {
    "name": "NewServer",
    "kind": 12,
    "containerName": "server",
    "location": {
      "uri": "file:///path/to/server.go",
      "range": {
        "start": {"line": 66, "character": 5},
        "end": {"line": 66, "character": 14}
      }
    }
  }
]
```

**Symbol Kind Values:**

| Value | Kind |
|-------|------|
| 5 | Class |
| 6 | Method |
| 10 | Enum |
| 11 | Interface |
| 12 | Function |
| 13 | Variable |
| 14 | Constant |
| 23 | Struct |

**Note:** Results limited to 10 symbols.

**File Reference:** `go-opencode/internal/server/handlers_file.go:281-317`

---

## MCP Endpoints

### GET /mcp

Get MCP server status.

**Response:**
```json
{
  "server-name": {
    "status": "connected",
    "error": null
  }
}
```

**Status Values:** `"connected"`, `"disabled"`, `"failed"`

**File Reference:** `go-opencode/internal/server/handlers_config.go:348-368`

---

### POST /mcp

Add an MCP server.

**Request Body:**
```json
{
  "name": "server-name",
  "type": "stdio",
  "command": ["node", "server.js"],
  "environment": {"KEY": "value"},
  "timeout": 30000
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Server name |
| `type` | string | Yes | `"stdio"` or `"sse"` |
| `url` | string | No | URL for SSE servers |
| `command` | string[] | No | Command for stdio servers |
| `headers` | object | No | HTTP headers for SSE |
| `environment` | object | No | Environment variables |
| `timeout` | number | No | Connection timeout (ms) |

**Response:** Server status object

**File Reference:** `go-opencode/internal/server/handlers_config.go:370-415`

---

### DELETE /mcp/{name}

Remove an MCP server.

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:417-436`

---

### GET /mcp/tools

Get all MCP tools.

**Response:**
```json
[
  {
    "name": "tool-name",
    "description": "Tool description",
    "inputSchema": {...}
  }
]
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:438-447`

---

### POST /mcp/tool/{name}

Execute an MCP tool.

**Request Body:** Tool arguments (JSON object)

**Response:**
```json
{
  "result": "tool output"
}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:449-475`

---

### GET /mcp/resources

List MCP resources.

**Response:** Array of MCP resources

**File Reference:** `go-opencode/internal/server/handlers_config.go:477-491`

---

### GET /mcp/resource

Read an MCP resource.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `uri` | string | Yes | Resource URI |

**Response:** Resource content

**File Reference:** `go-opencode/internal/server/handlers_config.go:493-513`

---

## Command Endpoints

### GET /command

List all commands.

**Response:**
```json
[
  {
    "name": "init",
    "description": "create/update AGENTS.md",
    "template": "Please analyze...",
    "agent": "",
    "model": "",
    "subtask": false
  }
]
```

**CommandInfo Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Command name (without /) |
| `description` | string | Short description |
| `template` | string | Prompt template with $ARGUMENTS |
| `agent` | string | Agent to use |
| `model` | string | Model override |
| `subtask` | boolean | Run as subtask |

**File Reference:** `go-opencode/internal/server/handlers_config.go:801-825`

---

### GET /command/{name}

Get a single command.

**Response:** CommandInfo object

**File Reference:** `go-opencode/internal/server/handlers_config.go:970-995`

---

### POST /command/{name}

Execute a command.

**Request Body:**
```json
{
  "args": "argument string",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx"
}
```

**Response:** Command result object

**SSE Event:** `command.executed` with:
```json
{
  "name": "command-name",
  "sessionID": "ses_xxxx",
  "arguments": "argument string",
  "messageID": "msg_xxxx"
}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:927-968`

---

## TUI Control Endpoints

### POST /tui/append-prompt

Append text to the TUI prompt.

**Request Body:**
```json
{
  "text": "text to append"
}
```

**Response:**
```json
{"success": true}
```

**SSE Event:** `tui.prompt.append` with `{text: string}`

**File Reference:** `go-opencode/internal/server/handlers_tui.go:14-31`

---

### POST /tui/execute-command

Execute a TUI command.

**Request Body:**
```json
{
  "command": "session.new"
}
```

**Response:**
```json
{"success": true}
```

**SSE Event:** `tui.command.execute` with `{command: string}`

**File Reference:** `go-opencode/internal/server/handlers_tui.go:33-50`

---

### POST /tui/show-toast

Show a toast notification.

**Request Body:**
```json
{
  "title": "Optional Title",
  "message": "Notification message",
  "variant": "info",
  "duration": 5000
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | No | Toast title |
| `message` | string | Yes | Toast message |
| `variant` | string | Yes | `"info"`, `"success"`, `"warning"`, `"error"` |
| `duration` | number | No | Display duration (ms) |

**Response:**
```json
{"success": true}
```

**SSE Event:** `tui.toast.show` with full request data

**File Reference:** `go-opencode/internal/server/handlers_tui.go:52-77`

---

### POST /tui/publish

Publish a generic TUI event.

**Request Body:**
```json
{
  "type": "tui.prompt.append",
  "properties": {
    "text": "appended text"
  }
}
```

**Supported Types:**
- `tui.prompt.append`
- `tui.command.execute`
- `tui.toast.show`

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_tui.go:79-121`

---

### POST /tui/open-help

Open the help dialog.

**Response:**
```json
{"success": true}
```

---

### POST /tui/open-sessions

Open the sessions dialog.

**Response:**
```json
{"success": true}
```

---

### POST /tui/open-themes

Open the themes dialog.

**Response:**
```json
{"success": true}
```

---

### POST /tui/open-models

Open the models dialog.

**Response:**
```json
{"success": true}
```

---

### POST /tui/submit-prompt

Submit the current prompt.

**Request Body:**
```json
{
  "text": "prompt text"
}
```

**Response:**
```json
{"success": true}
```

---

### POST /tui/clear-prompt

Clear the prompt.

**Response:**
```json
{"success": true}
```

---

### GET /tui/control/next

Get next pending TUI control request.

**Response:**
```json
{
  "path": "/some/path",
  "body": {...}
}
```

Returns empty path if nothing pending.

---

### POST /tui/control/response

Submit a response to a TUI control request.

**Request Body:** Response data (any JSON)

**Response:**
```json
{"success": true}
```

---

## Client Tools Endpoints

### POST /client-tools/register

Register client-provided tools.

**Request Body:**
```json
{
  "clientID": "client-xxx",
  "tools": [
    {
      "id": "my-tool",
      "description": "Tool description",
      "parameters": {
        "type": "object",
        "properties": {...}
      }
    }
  ]
}
```

**Response:**
```json
{
  "registered": ["my-tool"]
}
```

**File Reference:** `go-opencode/internal/server/handlers_tui.go:187-221`

---

### DELETE /client-tools/unregister

Unregister client tools.

**Request Body:**
```json
{
  "clientID": "client-xxx",
  "toolIDs": ["my-tool"]
}
```

**Response:**
```json
{
  "success": true,
  "unregistered": ["my-tool"]
}
```

**File Reference:** `go-opencode/internal/server/handlers_tui.go:223-244`

---

### POST /client-tools/execute

Execute a client tool.

**Request Body:**
```json
{
  "toolID": "my-tool",
  "requestID": "req-xxx",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx",
  "callID": "call_xxxx",
  "input": {...},
  "timeout": 30000
}
```

**Response:**
```json
{
  "status": "success",
  "output": "result..."
}
```

**File Reference:** `go-opencode/internal/server/handlers_tui.go:246-291`

---

### POST /client-tools/result

Submit a tool execution result.

**Request Body:**
```json
{
  "requestID": "req-xxx",
  "status": "success",
  "title": "Result Title",
  "output": "tool output",
  "metadata": {...},
  "error": null
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `requestID` | string | Yes | Original request ID |
| `status` | string | Yes | `"success"` or `"error"` |
| `title` | string | No | Display title |
| `output` | string | No | Tool output |
| `metadata` | object | No | Additional metadata |
| `error` | string | No | Error message (if status=error) |

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_tui.go:293-327`

---

### GET /client-tools/pending/{clientID}

Get pending tool requests for a client.

**Response:** Array of pending execution requests

---

### GET /client-tools/tools/{clientID}

Get tools registered by a specific client.

**Response:** Array of tool definitions

---

### GET /client-tools/tools

Get all registered client tools.

**Response:** Array of all tool definitions

---

## Utility Endpoints

### GET /agent

List available agents.

**Response:**
```json
[
  {
    "name": "build",
    "description": "",
    "mode": "primary",
    "builtIn": true,
    "prompt": "",
    "tools": {},
    "options": {},
    "permission": {
      "edit": "allow",
      "bash": {"*": "allow"},
      "webfetch": "allow",
      "external_directory": "ask",
      "doom_loop": "ask"
    },
    "temperature": 0,
    "topP": 0,
    "model": null,
    "color": ""
  }
]
```

**AgentInfo Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Agent identifier |
| `description` | string | Agent description |
| `mode` | string | `"primary"` or `"subagent"` |
| `builtIn` | boolean | Built-in vs custom |
| `prompt` | string | Custom system prompt |
| `tools` | object | Tool enable/disable map |
| `options` | object | Agent options |
| `permission` | object | Permission settings |
| `temperature` | number | Temperature override |
| `topP` | number | Top-p override |
| `model` | ModelRef? | Model override |
| `color` | string | UI color |

**Permission Fields:**

| Field | Type | Values |
|-------|------|--------|
| `edit` | string | `"allow"`, `"deny"`, `"ask"` |
| `bash` | object | Pattern → permission map |
| `webfetch` | string | `"allow"`, `"deny"`, `"ask"` |
| `external_directory` | string | `"allow"`, `"deny"`, `"ask"` |
| `doom_loop` | string | `"allow"`, `"deny"`, `"ask"` |

**File Reference:** `go-opencode/internal/server/handlers_config.go:547-624`

---

### GET /lsp

Get LSP server status.

**Response:**
```json
{
  "enabled": true,
  "servers": []
}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:332-339`

---

### GET /formatter

Get formatter status.

**Response:**
```json
{
  "enabled": true,
  "formatters": [...]
}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:743-752`

---

### POST /formatter/format

Format a file.

**Request Body:**
```json
{
  "path": "/path/to/file.go"
}
```

Or for multiple files:
```json
{
  "paths": ["/path/to/file1.go", "/path/to/file2.go"]
}
```

**Response:** Format result(s)

**File Reference:** `go-opencode/internal/server/handlers_config.go:754-788`

---

### GET /path

Get the current working directory.

**Response:**
```json
{
  "directory": "/path/to/project"
}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:997-1002`

---

### POST /log

Write to server log.

**Request Body:** Log message (any format)

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:1004-1008`

---

### POST /instance/dispose

Dispose of server instance resources.

**Response:**
```json
{"success": true}
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:1010-1014`

---

### GET /experimental/tool/ids

Get list of tool IDs.

**Response:**
```json
["read", "write", "edit", "bash", "glob", "grep", ...]
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:1016-1024`

---

### GET /experimental/tool

Get full tool definitions.

**Response:**
```json
[
  {
    "name": "read",
    "description": "Read file contents",
    "parameters": {...}
  }
]
```

**File Reference:** `go-opencode/internal/server/handlers_config.go:1026-1038`

---

### GET /doc

Get OpenAPI specification.

**Response:** OpenAPI 3.0.0 spec document

**File Reference:** `go-opencode/internal/server/handlers_tui.go:329-343`

---

### GET /project

List projects.

**Response:** Array of Project objects

---

### GET /project/current

Get current project.

**Response:** Project object

---

## Part Types Reference

Parts are the content blocks within messages.

### TextPart

```json
{
  "id": "prt_xxxx",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx",
  "type": "text",
  "text": "Text content...",
  "time": {
    "start": 1764964063910,
    "end": 1764964065196
  }
}
```

### ToolPart

```json
{
  "id": "prt_xxxx",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx",
  "type": "tool",
  "callID": "toolu_xxxx",
  "tool": "read",
  "state": {
    "status": "completed",
    "input": {"filePath": "/path/to/file"},
    "raw": "",
    "output": "file contents...",
    "error": null,
    "title": "go-opencode/internal/server.go",
    "metadata": {
      "lineCount": 150,
      "truncated": false
    },
    "time": {
      "start": 1764964069518,
      "end": 1764964072550
    }
  }
}
```

**Tool Status Values:** `"pending"`, `"running"`, `"completed"`, `"error"`

### FilePart

```json
{
  "id": "prt_xxxx",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx",
  "type": "file",
  "filename": "image.png",
  "mime": "image/png",
  "url": "data:image/png;base64,..."
}
```

### StepStartPart

```json
{
  "id": "prt_xxxx",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx",
  "type": "step-start",
  "snapshot": "git-hash"
}
```

### StepFinishPart

```json
{
  "id": "prt_xxxx",
  "sessionID": "ses_xxxx",
  "messageID": "msg_xxxx",
  "type": "step-finish",
  "reason": "tool-calls",
  "snapshot": "git-hash",
  "cost": 0.0,
  "tokens": {
    "input": 5,
    "output": 224,
    "reasoning": 0,
    "cache": {
      "read": 13510,
      "write": 248
    }
  }
}
```

---

## Permission Event Reference

### permission.updated

Sent when a tool requires user permission.

```json
{
  "type": "permission.updated",
  "properties": {
    "id": "per_xxxx",
    "type": "external_directory",
    "pattern": ["/path/to/dir", "/path/to/dir/*"],
    "sessionID": "ses_xxxx",
    "messageID": "msg_xxxx",
    "callID": "toolu_xxxx",
    "title": "Access file outside working directory",
    "metadata": {
      "filepath": "/path/to/file.go",
      "parentDir": "/path/to/dir"
    },
    "time": {
      "created": 1764964069517
    }
  }
}
```

**Permission Types:**
- `external_directory` - Access outside working directory
- `bash` - Execute shell command
- `edit` - Edit file

---

## Version History

- **1.0.0** (2025-12-05) - Initial comprehensive endpoint documentation

---

## References

- **Event Protocol:** See `docs/tui-event-specification.md`
- **TUI Protocol:** See `docs/tui-protocol-specification.md`
- **TypeScript Reference:** `packages/opencode/src/server/`
- **Go Implementation:** `go-opencode/internal/server/`
