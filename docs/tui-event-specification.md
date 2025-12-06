# OpenCode TUI Event Protocol Specification

**Version:** 2.0.0
**Last Updated:** 2025-12-05

## Table of Contents

- [Overview](#overview)
- [Comparison with AI SDK UI Message Stream Protocol](#comparison-with-ai-sdk-ui-message-stream-protocol)
- [Transport Layer](#transport-layer)
- [Event Envelope Format](#event-envelope-format)
- [Event Categories](#event-categories)
- [Complete Event Reference](#complete-event-reference)
- [Event Temporal Sequences](#event-temporal-sequences)
- [Metadata Usage](#metadata-usage)
- [Validation Against Existing Documentation](#validation-against-existing-documentation)

---

## Overview

The OpenCode TUI Event Protocol defines how the server communicates real-time state changes to TUI clients via Server-Sent Events (SSE). Unlike request-response patterns, events flow unidirectionally from server to client, enabling real-time UI updates during AI interactions.

### Key Characteristics

| Characteristic | OpenCode TUI Protocol | AI SDK UI Message Stream |
|---------------|----------------------|--------------------------|
| **Transport** | SSE (Server-Sent Events) | SSE (Server-Sent Events) |
| **Envelope** | `{type, properties}` | `{type, ...fields}` |
| **Scope** | Session-centric (stateful) | Message-centric (stateless) |
| **Event Count** | 31+ event types | 24 event types |
| **Streaming** | Accumulated + Delta | Delta only |
| **Tool State** | Full state machine | Input/Output streaming |

### File References

- **Event Type Constants**: `go-opencode/internal/event/bus.go:40-89`
- **Event Data Structures**: `go-opencode/internal/event/types.go`
- **SSE Handler**: `go-opencode/internal/server/sse.go`
- **TypeScript Reference**: `packages/opencode/src/bus/index.ts`

---

## Comparison with AI SDK UI Message Stream Protocol

### Architectural Differences

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AI SDK UI Message Stream                              │
│                                                                              │
│   Request ──► [start] ──► [text-start] ──► [text-delta]* ──► [text-end]    │
│                      ──► [tool-input-start] ──► [tool-output] ──► [finish]  │
│                                                                              │
│   • Stateless: Each message is independent                                   │
│   • Fine-grained: Separate start/delta/end for each content type            │
│   • Message-scoped: Events tied to single message generation                │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                        OpenCode TUI Protocol                                 │
│                                                                              │
│   [session.status:busy] ──► [message.created] ──► [message.part.updated]*   │
│                         ──► [session.diff] ──► [session.status:idle]        │
│                                                                              │
│   • Stateful: Events update persistent session state                        │
│   • Coarse-grained: Part updates carry full state + optional delta          │
│   • Session-scoped: Events include sessionID for multi-session support      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Event Type Mapping

| Purpose | AI SDK Event | OpenCode Event |
|---------|-------------|----------------|
| Message start | `start` | `message.created` |
| Message end | `finish` | `message.updated` (with finish reason) |
| Text streaming | `text-start`, `text-delta`, `text-end` | `message.part.updated` (type=text, delta field) |
| Reasoning | `reasoning-start`, `reasoning-delta`, `reasoning-end` | `message.part.updated` (type=reasoning) |
| Tool input | `tool-input-start`, `tool-input-delta`, `tool-input-available` | `message.part.updated` (type=tool, state.status) |
| Tool output | `tool-output-available` | `message.part.updated` (type=tool, state.output) |
| Error | `error` | `session.error` or tool state.error |
| File reference | `file` | `message.part.updated` (type=file) |
| Step boundaries | `start-step`, `finish-step` | `message.part.updated` (type=step-start/step-finish) |
| Stream end | `[DONE]` | `session.idle` |

### Key Differences in Detail

#### 1. Delta Handling

**AI SDK**: Sends only the delta (incremental text)
```json
{"type": "text-delta", "delta": "Hello"}
{"type": "text-delta", "delta": " world"}
```

**OpenCode**: Sends accumulated state + optional delta
```json
{
  "type": "message.part.updated",
  "properties": {
    "part": {"type": "text", "text": "Hello"},
    "delta": "Hello"
  }
}
{
  "type": "message.part.updated",
  "properties": {
    "part": {"type": "text", "text": "Hello world"},
    "delta": " world"
  }
}
```

**Rationale**: OpenCode's approach allows late-joining clients to get current state without replay.

#### 2. Tool State Machine

**AI SDK**: Streaming tool input then output
```json
{"type": "tool-input-start", "toolCallId": "call_1", "toolName": "read"}
{"type": "tool-input-delta", "toolCallId": "call_1", "delta": "{\"path\":"}
{"type": "tool-input-available", "toolCallId": "call_1", "input": {"path": "/file.txt"}}
{"type": "tool-output-available", "toolCallId": "call_1", "output": "file contents"}
```

**OpenCode**: Full state machine in single event type
```json
{"type": "message.part.updated", "properties": {"part": {"type": "tool", "state": {"status": "pending"}}}}
{"type": "message.part.updated", "properties": {"part": {"type": "tool", "state": {"status": "running", "input": {...}}}}}
{"type": "message.part.updated", "properties": {"part": {"type": "tool", "state": {"status": "completed", "output": "..."}}}}
```

#### 3. Session vs Message Scope

**AI SDK**: Events scoped to single message generation
- No cross-message state
- No persistent session concept

**OpenCode**: Events scoped to session with message context
- `sessionID` in every event
- Session lifecycle events (created, deleted, idle)
- Diff tracking across messages

---

## Transport Layer

### SSE Connection

**Endpoint**: `GET /event`

**Headers**:
```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

**Query Parameters**:
- `directory` (optional): Working directory path
- `sessionID` (optional): Filter events to specific session

### Wire Format

```
event: message
data: {"type":"session.created","properties":{"info":{...}}}

event: message
data: {"type":"message.part.updated","properties":{"part":{...},"delta":"text"}}

: heartbeat

```

**File Reference**: `go-opencode/internal/server/sse.go:45-78`

---

## Event Envelope Format

All events follow the SDK-compatible envelope format:

```typescript
interface Event {
  type: string;        // Event type identifier (e.g., "session.created")
  properties: object;  // Event-specific data payload
}
```

**File Reference**: `go-opencode/internal/server/sse.go:20-24`

---

## Event Categories

### Session Events (8 types)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `session.created` | New session created | POST /session |
| `session.updated` | Session metadata changed | PATCH /session, title generation |
| `session.deleted` | Session removed | DELETE /session |
| `session.status` | Processing state changed | Message processing start/end |
| `session.idle` | Session became idle | Message processing complete |
| `session.diff` | File changes tracked | After tool execution |
| `session.error` | Error occurred | Processing failure |
| `session.compacted` | History compressed | Manual/auto compaction |

### Message Events (5 types)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `message.created` | New message added | User/assistant message created |
| `message.updated` | Message changed | Status change, completion |
| `message.removed` | Message deleted | Message removal |
| `message.part.updated` | Part content changed | Streaming, tool updates |
| `message.part.removed` | Part deleted | Part removal |

### Permission Events (2 types)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `permission.updated` | Permission request created | Tool needs approval |
| `permission.replied` | User responded | User grants/denies |

### TUI Events (3 types)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `tui.prompt.append` | Append text to prompt | API call |
| `tui.command.execute` | Execute TUI command | API call |
| `tui.toast.show` | Show notification | API call |

### VCS Events (1 type)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `vcs.branch.updated` | Git branch changed | .git/HEAD file change |

### PTY Events (4 types)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `pty.created` | Terminal session created | POST /pty |
| `pty.updated` | Terminal updated | Resize, title change |
| `pty.exited` | Process exited | Terminal process exit |
| `pty.deleted` | Terminal removed | DELETE /pty |

### Command Events (1 type)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `command.executed` | Slash command run | POST /command/:name |

### Client Tool Events (6 types)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `client-tool.request` | Execution requested | AI calls client tool |
| `client-tool.registered` | Tools registered | POST /client-tools/register |
| `client-tool.unregistered` | Tools removed | DELETE /client-tools/unregister |
| `client-tool.executing` | Execution started | Tool execution begins |
| `client-tool.completed` | Execution succeeded | Tool returns success |
| `client-tool.failed` | Execution failed | Tool returns error |

### File Events (1 type)

| Event Type | Description | When Fired |
|-----------|-------------|------------|
| `file.edited` | File was modified | Write/Edit tool execution |

---

## Complete Event Reference

### session.created

Fired when a new session is created.

```typescript
{
  type: "session.created",
  properties: {
    info: {
      id: string,                    // Session ID (ULID format)
      projectID: string,             // Project identifier
      directory: string,             // Working directory path
      parentID?: string,             // Parent session for forks
      title: string,                 // Session title
      version: string,               // Schema version
      summary: {
        title?: string,              // AI-generated title
        body?: string,               // Summary text
        diffs?: FileDiff[]           // File changes
      },
      share?: {
        url: string                  // Sharing URL
      },
      time: {
        created: number,             // Unix timestamp (ms)
        updated?: number             // Last update timestamp
      }
    }
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:6-9`

---

### session.updated

Fired when session metadata changes.

```typescript
{
  type: "session.updated",
  properties: {
    info: Session                    // Full session object (same as session.created)
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:11-15`

---

### session.status

Fired when session processing state changes.

```typescript
{
  type: "session.status",
  properties: {
    sessionID: string,
    status: {
      type: "busy" | "idle"          // Current processing state
    }
  }
}
```

**Temporal Position**:
- `busy`: First event when message processing starts
- `idle`: Fired after `session.idle` at processing completion

**File Reference**: `go-opencode/internal/event/types.go:28-38`

---

### session.diff

Fired when file changes are recorded.

```typescript
{
  type: "session.diff",
  properties: {
    sessionID: string,
    diff: Array<{
      file: string,                  // File path
      additions: number,             // Lines added
      deletions: number,             // Lines removed
      before?: string,               // Original content
      after?: string                 // New content
    }>
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:40-45`

---

### session.error

Fired when an error occurs during processing.

```typescript
{
  type: "session.error",
  properties: {
    sessionID?: string,
    error?: {
      name: string,                  // Error type
      data: {
        message: string,             // Error message
        providerID?: string          // Provider that caused error
      }
    }
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:47-51`

---

### session.idle

Fired when session becomes idle after processing.

```typescript
{
  type: "session.idle",
  properties: {
    sessionID: string
  }
}
```

**Temporal Position**: Final event in message processing sequence.

**File Reference**: `go-opencode/internal/event/types.go:24-26`

---

### message.created

Fired when a new message is added to the session.

```typescript
{
  type: "message.created",
  properties: {
    info: {
      id: string,                    // Message ID
      sessionID: string,             // Parent session
      role: "user" | "assistant",    // Message role
      time: {
        created: number,
        updated?: number
      },
      // User message fields
      agent?: string,                // Agent name
      model?: {
        providerID: string,
        modelID: string
      },
      // Assistant message fields
      parentID?: string,             // Parent user message
      modelID?: string,
      providerID?: string,
      mode?: string,                 // Execution mode
      finish?: string,               // Finish reason
      cost?: number,                 // Cost in USD
      tokens?: {
        input: number,
        output: number,
        reasoning: number,
        cache: {
          read: number,
          write: number
        }
      },
      error?: MessageError
    }
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:58-62`, `go-opencode/pkg/types/message.go`

---

### message.updated

Fired when message metadata changes.

```typescript
{
  type: "message.updated",
  properties: {
    info: Message                    // Full message object
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:64-68`

---

### message.part.updated

**The most frequently fired event during streaming.** Contains part state and optional delta.

```typescript
{
  type: "message.part.updated",
  properties: {
    part: Part,                      // Full part state (see Part Types below)
    delta?: string                   // Incremental text (for streaming only)
  }
}
```

#### Part Types

**TextPart**:
```typescript
{
  id: string,
  sessionID: string,
  messageID: string,
  type: "text",
  text: string,                      // Accumulated text
  time?: { start?: number, end?: number },
  metadata?: Record<string, any>
}
```

**ReasoningPart**:
```typescript
{
  id: string,
  sessionID: string,
  messageID: string,
  type: "reasoning",
  text: string,                      // Reasoning content
  time?: { start?: number, end?: number }
}
```

**ToolPart** (most complex):
```typescript
{
  id: string,
  sessionID: string,
  messageID: string,
  type: "tool",
  callID: string,                    // Tool call identifier
  tool: string,                      // Tool name
  state: {
    status: "pending" | "running" | "completed" | "error",
    input: Record<string, any>,      // Tool parameters
    raw?: string,                    // Raw input string (for streaming)
    output?: string,                 // Tool output
    error?: string,                  // Error message
    title?: string,                  // Display title
    metadata?: Record<string, any>,  // Tool-specific data
    time?: {
      start: number,
      end?: number,
      compacted?: number
    },
    attachments?: FilePart[]
  },
  metadata?: Record<string, any>
}
```

**FilePart**:
```typescript
{
  id: string,
  sessionID: string,
  messageID: string,
  type: "file",
  filename?: string,
  mime: string,
  url: string
}
```

**StepStartPart**:
```typescript
{
  id: string,
  sessionID: string,
  messageID: string,
  type: "step-start",
  snapshot?: string                  // State snapshot
}
```

**StepFinishPart**:
```typescript
{
  id: string,
  sessionID: string,
  messageID: string,
  type: "step-finish",
  reason: string,                    // Finish reason
  snapshot?: string,
  cost: number,
  tokens?: TokenUsage
}
```

**File Reference**: `go-opencode/internal/event/types.go:76-91`, `go-opencode/pkg/types/parts.go`

---

### permission.updated

Fired when a permission request is created.

```typescript
{
  type: "permission.updated",
  properties: {
    id: string,                      // Permission request ID
    sessionID: string,
    permissionType: "bash" | "edit" | "external_directory",
    pattern: string[],               // What's being requested
    title: string                    // Display title
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:98-106`

---

### permission.replied

Fired when user responds to permission request.

```typescript
{
  type: "permission.replied",
  properties: {
    permissionID: string,
    sessionID: string,
    response: "once" | "always" | "reject"
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:111-116`

---

### tui.prompt.append

Fired to append text to the TUI prompt.

```typescript
{
  type: "tui.prompt.append",
  properties: {
    text: string                     // Text to append
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:156-159`

---

### tui.command.execute

Fired to execute a TUI command.

```typescript
{
  type: "tui.command.execute",
  properties: {
    command: string                  // Command name (e.g., "session.new")
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:161-164`

---

### tui.toast.show

Fired to display a toast notification.

```typescript
{
  type: "tui.toast.show",
  properties: {
    title?: string,                  // Optional title
    message: string,                 // Notification message
    variant: "info" | "success" | "warning" | "error",
    duration?: number                // Display duration (ms)
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:166-172`

---

### vcs.branch.updated

Fired when git branch changes.

```typescript
{
  type: "vcs.branch.updated",
  properties: {
    branch?: string                  // New branch name
  }
}
```

**Trigger**: fsnotify watch on `.git/HEAD` file.

**File Reference**: `go-opencode/internal/event/types.go:176-179`, `go-opencode/internal/vcs/watcher.go`

---

### pty.created / pty.updated

Fired when PTY session is created or updated.

```typescript
{
  type: "pty.created" | "pty.updated",
  properties: {
    info: {
      id: string,
      title: string,
      command: string,
      args: string[],
      cwd: string,
      status: "running" | "exited",
      pid: number
    }
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:183-202`

---

### pty.exited

Fired when PTY process exits.

```typescript
{
  type: "pty.exited",
  properties: {
    id: string,
    exitCode: number
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:204-208`

---

### pty.deleted

Fired when PTY session is removed.

```typescript
{
  type: "pty.deleted",
  properties: {
    id: string
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:210-213`

---

### command.executed

Fired when a slash command is executed.

```typescript
{
  type: "command.executed",
  properties: {
    name: string,                    // Command name
    sessionID: string,
    arguments: string,               // Command arguments
    messageID: string
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:217-223`

---

### client-tool.* Events

See the [Client Tools Protocol](#client-tools-protocol) section in the main specification.

**File Reference**: `go-opencode/internal/event/types.go:125-152`

---

### file.edited

Fired when a file is modified.

```typescript
{
  type: "file.edited",
  properties: {
    file: string                     // File path
  }
}
```

**File Reference**: `go-opencode/internal/event/types.go:93-96`

---

## Event Temporal Sequences

### Session Creation Flow

```
POST /session
    │
    ▼
┌─────────────────────┐
│  session.created    │  ◄── Single event
└─────────────────────┘
```

**Simplest flow**: Single event upon successful session creation.

---

### Message Sending Flow (Complete)

```
POST /session/{id}/message
    │
    ▼
┌─────────────────────┐
│  message.updated    │  ◄── User message stored (status: completed)
│  (user message)     │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ message.part.updated│  ◄── User text part
│  (type: text)       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  session.status     │  ◄── status.type = "busy"
│  (busy)             │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  session.updated    │  ◄── Session state updated
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  session.diff       │  ◄── Empty diffs at start
│  (initial)          │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  message.created    │  ◄── Assistant message created
│  (assistant)        │
└──────────┬──────────┘
           │
           ▼
    ╔═══════════════════════════════════════╗
    ║     STREAMING PHASE (repeated)         ║
    ╠═══════════════════════════════════════╣
    ║ message.part.updated (step-start)      ║
    ║ message.part.updated (text, delta)  *N ║
    ║ message.part.updated (reasoning)    *N ║
    ║ message.part.updated (tool)         *N ║
    ║ message.part.updated (step-finish)     ║
    ╚═══════════════════════════════════════╝
           │
           ▼
┌─────────────────────┐
│  message.updated    │  ◄── Final message state (finish reason)
│  (complete)         │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  session.status     │  ◄── status.type = "idle"
│  (idle)             │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  session.idle       │  ◄── Final event
└─────────────────────┘
```

---

### Tool Execution Flow (within streaming)

```
LLM returns tool_use
    │
    ▼
┌─────────────────────────────────────┐
│ message.part.updated               │  ◄── status: "pending"
│   type: "tool"                     │
│   state.status: "pending"          │
│   state.input: {} (accumulating)   │
└──────────────────┬──────────────────┘
                   │  (argument streaming)
                   ▼
┌─────────────────────────────────────┐
│ message.part.updated               │  ◄── status: "running"
│   state.status: "running"          │
│   state.input: {complete}          │
└──────────────────┬──────────────────┘
                   │  (execution)
                   ▼
┌─────────────────────────────────────┐
│ message.part.updated               │  ◄── Metadata updates (optional)
│   state.metadata: {...}            │
└──────────────────┬──────────────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
┌───────────────────┐  ┌───────────────────┐
│ SUCCESS           │  │ FAILURE           │
├───────────────────┤  ├───────────────────┤
│ message.part.     │  │ message.part.     │
│   updated         │  │   updated         │
│ status: completed │  │ status: error     │
│ output: "..."     │  │ error: "..."      │
└───────────────────┘  └───────────────────┘
        │                     │
        └──────────┬──────────┘
                   ▼
┌─────────────────────────────────────┐
│ session.diff                        │  ◄── If file was edited
│   diff: [{file, additions, ...}]    │
└─────────────────────────────────────┘
```

---

### Permission Request Flow

```
Tool requires permission
    │
    ▼
┌─────────────────────────────────────┐
│ permission.updated                  │  ◄── Request created
│   id: "perm_xxx"                    │
│   permissionType: "bash"            │
│   pattern: ["rm -rf *"]             │
│   title: "Delete all files"         │
└──────────────────┬──────────────────┘
                   │
    ╔══════════════╧══════════════╗
    ║   BLOCKING: Waits for user   ║
    ║   response via HTTP POST     ║
    ╚══════════════╤══════════════╝
                   │
                   ▼
┌─────────────────────────────────────┐
│ permission.replied                  │  ◄── User response
│   permissionID: "perm_xxx"          │
│   response: "once" | "always" |     │
│             "reject"                │
└──────────────────┬──────────────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
   [response: once/always]   [response: reject]
        │                     │
        ▼                     ▼
   Tool executes         Tool fails with
                         RejectedError
```

---

## Metadata Usage

### Tool Part Metadata

The `metadata` field in ToolPart is used for tool-specific data that doesn't fit the standard schema:

| Tool | Metadata Fields | Purpose |
|------|----------------|---------|
| `Read` | `lineCount`, `truncated` | File reading stats |
| `Write` | `created`, `backup` | File creation info |
| `Edit` | `matches`, `replaced` | Edit operation stats |
| `Bash` | `exitCode`, `signal` | Process exit info |
| `Glob` | `matchCount` | Search results count |
| `Grep` | `matchCount`, `files` | Search results |
| `Task` | `subagentType`, `agentId` | Subagent info |

**Example**:
```json
{
  "type": "message.part.updated",
  "properties": {
    "part": {
      "type": "tool",
      "tool": "Read",
      "state": {
        "status": "completed",
        "output": "file contents...",
        "metadata": {
          "lineCount": 150,
          "truncated": true,
          "truncatedAt": 100
        }
      }
    }
  }
}
```

### Text Part Metadata

Used for rendering hints:

| Field | Purpose |
|-------|---------|
| `language` | Code block language for syntax highlighting |
| `title` | Code block title |

---

## Validation Against Existing Documentation

### Comparison with `docs/tui-protocol-specification.md`

| Aspect | Existing Doc (v1.1.0) | This Spec (v2.0.0) | Status |
|--------|----------------------|-------------------|--------|
| Event envelope | `{type, properties}` | `{type, properties}` | **Correct** |
| session.created | `{sessionID}` | `{info: Session}` | **Updated** - SDK format |
| session.updated | `{sessionID, title}` | `{info: Session}` | **Updated** - Full object |
| session.status | `status: "pending"\|"running"\|...` | `status.type: "busy"\|"idle"` | **Updated** - Simpler states |
| message.part.updated | `{part, delta}` | `{part, delta}` | **Correct** |
| message.updated | `{sessionID, messageID, status}` | `{info: Message}` | **Updated** - Full object |
| permission.updated | `{sessionID, permissionID, tool, status}` | `{id, sessionID, permissionType, pattern, title}` | **Updated** - SDK format |
| permission.replied | `{response: "allow"\|"deny"\|...}` | `{response: "once"\|"always"\|"reject"}` | **Updated** - Correct values |
| file.edited | `{sessionID, messageID, path}` | `{file}` | **Updated** - Simpler format |
| VCS events | Not documented | `vcs.branch.updated` | **Added** |
| PTY events | Not documented | `pty.*` | **Added** |
| Command events | Partial | `command.executed` | **Expanded** |

### Breaking Changes from v1.1.0

1. **Session events now use `info` wrapper** for SDK compatibility
2. **Permission events renamed**: `permissionID` field name changes
3. **Status values simplified**: "pending/running/completed/error" → "busy/idle"
4. **File events simplified**: Removed session/message context

### Additions in v2.0.0

1. VCS events (`vcs.branch.updated`)
2. PTY events (`pty.created`, `pty.updated`, `pty.exited`, `pty.deleted`)
3. Command events (`command.executed`)
4. Complete temporal sequence documentation
5. AI SDK comparison

---

## Version History

- **2.0.0** (2025-12-05) - Complete rewrite with temporal sequences, AI SDK comparison, metadata documentation
- **1.1.0** (2025-11-25) - Added Client Tools Protocol
- **1.0.0** (2025-11-24) - Initial specification

---

## References

- **AI SDK UI Message Stream Protocol**: https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol
- **Server-Sent Events Specification**: https://html.spec.whatwg.org/multipage/server-sent-events.html
- **OpenCode Repository**: https://github.com/sst/opencode
- **go-opencode Implementation**: `go-opencode/internal/event/`
- **TypeScript Implementation**: `packages/opencode/src/bus/`
