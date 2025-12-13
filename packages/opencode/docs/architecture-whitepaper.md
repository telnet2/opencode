# OpenCode Architecture Whitepaper

**Version**: 1.0
**Date**: November 2024
**Status**: Technical Analysis

---

## Executive Summary

OpenCode is a sophisticated AI-powered coding assistant that integrates Language Server Protocol (LSP) capabilities, Model Context Protocol (MCP) servers, and large language models to provide intelligent code assistance. This whitepaper provides a comprehensive analysis of OpenCode's architecture, design decisions, and operational characteristics.

**Key Characteristics**:
- **Stateful architecture** requiring session affinity
- **Event-driven** real-time updates with SSE
- **LSP integration** with 19+ language servers
- **MCP support** for extensible tool integration
- **File-based storage** with in-memory locking
- **Multi-client support** with sequential message processing

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Core Architecture](#2-core-architecture)
3. [Session Management](#3-session-management)
4. [MCP Server Integration](#4-mcp-server-integration)
5. [LSP Integration](#5-lsp-integration)
6. [System Prompt Construction](#6-system-prompt-construction)
7. [Event System](#7-event-system)
8. [Storage Layer](#8-storage-layer)
9. [Concurrency Control](#9-concurrency-control)
10. [Multi-Server Considerations](#10-multi-server-considerations)
11. [Security Model](#11-security-model)
12. [Performance Characteristics](#12-performance-characteristics)
13. [Design Decisions](#13-design-decisions)
14. [Future Considerations](#14-future-considerations)

---

## 1. System Overview

### 1.1 Architecture Style

OpenCode employs a **monolithic stateful architecture** with the following characteristics:

- **Single-process execution** per project instance
- **File-based persistence** for session data
- **In-memory state management** for active sessions
- **Event-driven communication** via Server-Sent Events (SSE)
- **Plugin-based extensibility** via MCP and LSP

### 1.2 Technology Stack

| Component | Technology |
|-----------|------------|
| **Runtime** | Bun (JavaScript runtime) |
| **Transport** | HTTP/1.1 with SSE |
| **Storage** | JSON files (XDG base directories) |
| **Locking** | In-memory reader-writer locks |
| **LSP Communication** | JSON-RPC over stdio |
| **MCP Communication** | HTTP/SSE or stdio |
| **Event Bus** | In-memory pub/sub |

### 1.3 Key Components

```
┌─────────────────────────────────────────────────────┐
│                  OpenCode Server                    │
├─────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │   Session    │  │     LSP      │  │   MCP    │ │
│  │  Management  │  │  Integration │  │  Servers │ │
│  └──────────────┘  └──────────────┘  └──────────┘ │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │   Storage    │  │  Event Bus   │  │  Prompt  │ │
│  │    Layer     │  │   (Pub/Sub)  │  │  System  │ │
│  └──────────────┘  └──────────────┘  └──────────┘ │
│  ┌──────────────┐  ┌──────────────┐               │
│  │   Locking    │  │     Tool     │               │
│  │  Mechanism   │  │   Registry   │               │
│  └──────────────┘  └──────────────┘               │
└─────────────────────────────────────────────────────┘
         │                    │                │
         ▼                    ▼                ▼
    ┌────────┐          ┌─────────┐     ┌──────────┐
    │  File  │          │   LSP   │     │   MCP    │
    │ System │          │ Servers │     │  Servers │
    └────────┘          └─────────┘     └──────────┘
```

---

## 2. Core Architecture

### 2.1 Project Instance Model

**File**: `packages/opencode/src/project/instance.ts`

OpenCode uses a **per-directory instance model**:

- Each working directory has its own `Instance`
- Instance maintains isolated state via `Instance.state()`
- State is scoped by initialization function (singleton per init)
- Cleanup via `Instance.dispose()` on process exit

**State Hierarchy**:
```
Instance (per directory)
├── SessionPrompt state (session locks, callbacks)
├── MCP state (clients, status)
├── LSP state (servers, broken tracking)
├── Bus state (subscriptions)
└── Storage state (directory path)
```

### 2.2 HTTP API Surface

**File**: `packages/opencode/src/server/server.ts`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/session` | GET | List sessions |
| `/session/:id` | GET | Get session info |
| `/session/:id/message` | GET | Get messages (paginated) |
| `/session/:id/message` | POST | Send message (streaming) |
| `/session/:id/diff` | GET | Get session diffs |
| `/session/:id/todo` | GET | Get session todos |
| `/event` | GET | SSE event stream |
| `/global/event` | GET | Global SSE stream |
| `/mcp` | GET | MCP server status |
| `/mcp` | POST | Add MCP server |

### 2.3 Message Flow

```
User Request
    ↓
POST /session/:id/message
    ↓
SessionPrompt.prompt()
    ↓
┌─────────────────────┐
│ Lock Acquisition    │ (start() function)
│ - Check busy state  │
│ - Queue if busy     │
└─────────────────────┘
    ↓
┌─────────────────────┐
│ Prompt Construction │
│ - System prompt     │
│ - Tool resolution   │
│ - Message history   │
└─────────────────────┘
    ↓
┌─────────────────────┐
│ LLM API Call        │
│ - Stream response   │
│ - Handle tool calls │
└─────────────────────┘
    ↓
┌─────────────────────┐
│ Storage & Events    │
│ - Write to disk     │
│ - Publish events    │
│ - Resolve callbacks │
└─────────────────────┘
    ↓
Response to Client
```

---

## 3. Session Management

### 3.1 Session Lifecycle

**File**: `packages/opencode/src/session/index.ts`

**Phases**:
1. **Creation**: `Session.create()` → writes JSON to storage
2. **Active**: Messages processed via `SessionPrompt.prompt()`
3. **Idle**: No active processing, can receive new messages
4. **Archived**: Historical data retained

**Storage Structure**:
```
~/.local/share/opencode/storage/
├── session/
│   └── {projectID}/
│       └── {sessionID}.json
├── message/
│   └── {sessionID}/
│       └── {messageID}.json
└── part/
    └── {messageID}/
        └── {partID}.json
```

### 3.2 Sequential Message Processing

**File**: `packages/opencode/src/session/prompt.ts` (lines 207-238)

**Key Mechanism**: Session-level lock with callback queue

```typescript
const state = Record<sessionID, {
  abort: AbortController,
  callbacks: Array<{resolve, reject}>
}>

function start(sessionID: string) {
  if (state[sessionID]) return undefined  // Already busy
  state[sessionID] = { abort: new AbortController(), callbacks: [] }
  return controller.signal
}
```

**Behavior**:
- First client acquires lock
- Subsequent clients queued in `callbacks[]`
- When processing completes, all queued callbacks resolved
- Guarantees sequential processing per session

### 3.3 Multi-Client Support

**Multiple connections allowed** via:
- Separate SSE connections per client
- Bus pub/sub broadcasts events to all
- Shared file storage for persistence

**Historical data retrieval**:
- NO automatic replay on connect
- Clients must fetch via REST APIs
- Pull-based for history, push-based for updates

---

## 4. MCP Server Integration

### 4.1 MCP Architecture

**File**: `packages/opencode/src/mcp/index.ts`

**MCP (Model Context Protocol)** enables external tool providers:

**Configuration Types**:

```typescript
// Local subprocess
{
  type: "local",
  command: ["npx", "mcp-server"],
  environment: { ... },
  timeout: 5000
}

// Remote HTTP/SSE
{
  type: "remote",
  url: "https://example.com/mcp",
  headers: { "Authorization": "..." },
  timeout: 5000
}
```

### 4.2 Connection Lifecycle

**Initialization** (on first tool access):
```
1. Load config from opencode.jsonc
2. For each MCP server:
   a. Validate configuration
   b. Create transport (HTTP/SSE/Stdio)
   c. Create MCP client via @ai-sdk/mcp
   d. Fetch tools with timeout
   e. Store client + status
```

**Transport Selection**:

| Server Type | Transport 1 | Transport 2 |
|-------------|-------------|-------------|
| **Remote** | StreamableHTTPClientTransport | SSEClientTransport (fallback) |
| **Local** | StdioClientTransport | - |

### 4.3 Tool Registration

**File**: `packages/opencode/src/session/prompt.ts` (lines 727-789)

**Tool Naming**: `{sanitized_client_name}_{sanitized_tool_name}`

**Integration Flow**:
```
MCP.tools()
    ↓
For each client:
    ↓
client.tools()
    ↓
Sanitize names (replace non-alphanumeric)
    ↓
resolveTools()
    ↓
Wrap with plugin hooks
    ↓
Available to LLM
```

**Plugin Hooks**:
- `tool.execute.before` - Pre-execution hook
- `tool.execute.after` - Post-execution hook

### 4.4 Error Handling

**Status Tracking**:
```typescript
Status =
  | { status: "connected" }
  | { status: "disabled" }
  | { status: "failed", error: string }
```

**Failure Modes**:
- Connection timeout (5s default)
- Tool fetch timeout (configurable)
- Transport failures (both transports tried)
- Subprocess spawn failures

---

## 5. LSP Integration

### 5.1 LSP Architecture

**File**: `packages/opencode/src/lsp/index.ts`

**Supported Features**:
- Diagnostics (errors, warnings)
- Hover information (type inspection)
- Workspace symbols (cross-file search)
- Document symbols (file outline)

### 5.2 Language Server Matrix

| Language | Server | Extensions | Auto-Install |
|----------|--------|------------|--------------|
| **TypeScript** | typescript-language-server | .ts, .tsx, .js, .jsx | Yes (npm) |
| **Go** | gopls | .go | Yes (`go install`) |
| **Python** | pyright | .py, .pyi | Yes (npm) |
| **Rust** | rust-analyzer | .rs | No (expects installed) |
| **C/C++** | clangd | .c, .cpp, .h, .hpp | Yes (GitHub) |
| **Java** | jdtls | .java | Yes (Eclipse) |
| **Ruby** | ruby-lsp | .rb | Yes (`gem install`) |

**19 total language servers** supported.

### 5.3 Server Selection Algorithm

**File**: `packages/opencode/src/lsp/index.ts` (lines 156-240)

```
getClients(file) {
  extension = extract_extension(file)

  for server in configured_servers:
    if extension not in server.extensions:
      continue

    root = server.root(file)  // Project root detection
    if not root:
      continue

    if broken.has(root + server.id):
      continue  // Previously failed

    if cached_client exists:
      return cached_client

    if spawn_inflight:
      wait for spawn
    else:
      spawn new server

    return client
}
```

**Root Detection**: Searches up directory tree for:
- Go: `go.work`, `go.mod`, `go.sum`
- TypeScript: `package-lock.json`, lockfiles
- Rust: `Cargo.toml` (with workspace detection)
- Python: `pyproject.toml`, `requirements.txt`

### 5.4 LSP Data Usage

**In Edit Tool** (`packages/opencode/src/tool/edit.ts`):
```typescript
await LSP.touchFile(filePath, true)  // Wait for diagnostics
const diagnostics = await LSP.diagnostics()
const errors = diagnostics.filter(d => d.severity === 1)
// Errors automatically shown to LLM
```

**In Prompt Generation** (`packages/opencode/src/session/prompt.ts`):
- Document symbols used for range refinement
- Workspace symbols for code navigation
- Range data for Read tool offset calculation

### 5.5 Transport

**JSON-RPC over stdio**:
```typescript
createMessageConnection(
  new StreamMessageReader(process.stdout),
  new StreamMessageWriter(process.stdin)
)
```

**Notification Handling**:
- `textDocument/publishDiagnostics` → tracked by file
- `window/workDoneProgress/create` → ignored
- `workspace/configuration` → returns init options

---

## 6. System Prompt Construction

### 6.1 Prompt Assembly Pipeline

**File**: `packages/opencode/src/session/prompt.ts` (lines 621-641)

```
resolveSystemPrompt() {
  messages = []

  // Step 1: Provider header
  messages.push(SystemPrompt.header(providerID))

  // Step 2: Base prompt (priority order)
  if (custom_system_override):
    messages.push(custom_system)
  else if (agent.prompt):
    messages.push(agent.prompt)
  else:
    messages.push(SystemPrompt.provider(modelID))

  // Step 3: Environment context
  messages.push(SystemPrompt.environment())

  // Step 4: Custom instructions
  messages.push(SystemPrompt.custom())

  // Optimization: Combine into 2 messages for caching
  return [messages[0], messages.slice(1).join("\n")]
}
```

### 6.2 Model-Specific Prompts

| Model | Header | Base Prompt | Focus |
|-------|--------|-------------|-------|
| **Claude** | "Claude Code" | anthropic.txt (106 lines) | TodoWrite, parallelism, code refs |
| **GPT-4/o1/o3** | None | beast.txt | Autonomous, research-heavy |
| **GPT-5** | None | codex.txt (319 lines) | Structured workflows |
| **Gemini** | None | gemini.txt (156 lines) | Gemini-specific |
| **Others** | None | qwen.txt | Concise (1-3 sentences) |

### 6.3 Environment Context

**File**: `packages/opencode/src/session/system.ts` (lines 36-59)

**Variables Substituted**:
- `${Instance.directory}` → Working directory
- `${project.vcs}` → Git repository status
- `${process.platform}` → OS platform
- `${new Date().toDateString()}` → Current date
- File tree via Ripgrep (limit: 200 files)

### 6.4 Custom Instructions

**Search Order**:

**Local** (project-specific):
1. `AGENTS.md`
2. `CLAUDE.md`
3. `CONTEXT.md` (deprecated)

**Global** (user-level):
1. `~/.opencode/AGENTS.md`
2. `~/.claude/CLAUDE.md`

**Format**: Each file prefixed with `"Instructions from: {path}\n{content}"`

### 6.5 Anthropic Prompt Content

**Key Sections** (from `anthropic.txt`):

1. **Identity**: "OpenCode, the best coding agent on the planet"
2. **Tone & Style**: Concise, no emojis, markdown
3. **Professional Objectivity**: Facts over validation
4. **Task Management**: Heavy TodoWrite usage
5. **Tool Policy**:
   - Parallel calls for independent operations
   - Task tool for codebase exploration
   - Specialized tools over bash
6. **Code References**: `file_path:line_number` format

---

## 7. Event System

### 7.1 Event Architecture

**File**: `packages/opencode/src/bus/index.ts`

**Components**:
- **Bus**: Local pub/sub within process
- **GlobalBus**: Cross-directory EventEmitter
- **SSE**: Server-Sent Events for clients

### 7.2 Event Flow

```
Event Source (Session.updateMessage)
    ↓
Bus.publish(MessageV2.Event.Updated, { info })
    ↓
┌────────────────────┬────────────────────┐
│                    │                    │
▼                    ▼                    ▼
Local Subscribers    GlobalBus.emit       Store to subscriptions map
    ↓                    ↓
Plugin Hooks         Cross-directory broadcast
    ↓                    ↓
Processing           Other instances
```

### 7.3 Event Types

**Session Events**:
- `session.created`
- `session.updated`
- `session.deleted`
- `session.diff`
- `session.error`

**Message Events**:
- `message.updated`
- `message.removed`
- `message.part.updated`
- `message.part.removed`

**LSP Events**:
- `lsp.updated`
- `lsp.client.diagnostics`

### 7.4 Client Subscription

**File**: `packages/opencode/src/server/server.ts` (lines 1973-1995)

```typescript
GET /event → streamSSE(async (stream) => {
  // Send connection ack
  stream.writeSSE({
    data: JSON.stringify({ type: "server.connected" })
  })

  // Subscribe to all events
  const unsub = Bus.subscribeAll(async (event) => {
    await stream.writeSSE({ data: JSON.stringify(event) })
  })

  // Cleanup on disconnect
  stream.onAbort(() => {
    unsub()
  })
})
```

**Key Behavior**: No historical replay, only future events.

---

## 8. Storage Layer

### 8.1 Storage Architecture

**File**: `packages/opencode/src/storage/storage.ts`

**Storage Location**: XDG base directories
- `~/.local/share/opencode/storage/` (Linux)
- `~/Library/Application Support/opencode/storage/` (macOS)

**Format**: JSON files with hierarchical structure

### 8.2 Storage Operations

| Operation | Lock Type | Atomicity |
|-----------|-----------|-----------|
| `Storage.read()` | Read lock (shared) | Read-only |
| `Storage.update()` | Write lock (exclusive) | Read-modify-write |
| `Storage.write()` | Write lock (exclusive) | Atomic write |

### 8.3 Lock Implementation

**File**: `packages/opencode/src/util/lock.ts`

**Reader-Writer Lock**:
```typescript
Lock = {
  readers: number,
  writer: boolean,
  waitingReaders: (() => void)[],
  waitingWriters: (() => void)[]
}
```

**Characteristics**:
- **Multiple concurrent readers** allowed
- **Single exclusive writer** (blocks all)
- **Writer priority** (prevents starvation)
- **In-memory only** (no cross-process protection)

### 8.4 Critical Limitation

**No distributed locking** - locks are process-local:
- Multiple server processes can corrupt data
- No file-level OS locks (`flock`/`fcntl`)
- No distributed coordination (Redis, etc.)

---

## 9. Concurrency Control

### 9.1 Concurrency Layers

| Layer | Mechanism | Scope | Guarantees |
|-------|-----------|-------|-----------|
| **Session Message** | Single-threaded loop + callback queue | Per session | Sequential processing |
| **File I/O** | Reader-Writer Lock | Per file | Concurrent reads, exclusive writes |
| **Event Publishing** | Bus pub/sub + Promise.all | Global | Atomic notification |
| **State Storage** | Directory-scoped Instance.state | Per project | Singleton per init function |
| **HTTP Connections** | SSE streams + individual subscriptions | Per connection | Independent delivery |

### 9.2 Race Condition Prevention

**Session Level**:
```typescript
// Only one message processed at a time
if (state[sessionID]) {
  // Queue this request
  return new Promise((resolve, reject) => {
    state[sessionID].callbacks.push({ resolve, reject })
  })
}
```

**File Level**:
```typescript
using _ = await Lock.write(target)  // Exclusive access
const content = await Bun.file(target).json()
fn(content)  // Modify
await Bun.write(target, JSON.stringify(content))
```

**Event Level**:
```typescript
const pending = subscribers.map(sub => sub(event))
await Promise.all(pending)  // Wait for all handlers
```

### 9.3 Cancellation

**AbortController per session**:
```typescript
state[sessionID] = {
  abort: new AbortController(),
  callbacks: []
}

// User cancels
SessionPrompt.cancel(sessionID)
state[sessionID].abort.abort()  // Propagates to LLM call
```

---

## 10. Multi-Server Considerations

### 10.1 Statefulness Analysis

**OpenCode is HIGHLY STATEFUL** due to:

| Component | Storage | Cross-Process |
|-----------|---------|---------------|
| **Session locks** | In-memory Map | ❌ No |
| **File locks** | In-memory Map | ❌ No |
| **Callback queues** | In-memory Map | ❌ No |
| **Session status** | In-memory Map | ❌ No |
| **Bus subscriptions** | In-memory Map | ❌ No |
| **AbortControllers** | In-memory objects | ❌ No |
| **Session data** | File system | ✅ Yes |
| **Message data** | File system | ✅ Yes |

### 10.2 Multi-Server Problems

**Scenario: Two servers handle same session**

| Time | Server A | Server B | Problem |
|------|----------|----------|---------|
| T0 | Acquires session lock | - | - |
| T1 | Processing message | Acquires session lock | **Concurrent processing** |
| T2 | Writes session.json | Writes session.json | **Last write wins** |
| T3 | - | Server A's changes lost | **Data corruption** |

**Additional Issues**:
- Cancellation doesn't propagate
- Callback queues lost
- Events not distributed
- Diagnostics inconsistent

### 10.3 Deployment Requirements

**Option 1: Single Server** (Recommended)
- Simple, no coordination needed
- All guarantees preserved

**Option 2: Session Affinity**
- Load balancer sticky sessions
- Cookie or IP-based routing
- Same guarantees within session

**Option 3: Full Distribution** (Not Supported)
Would require:
- Distributed file locking (Redis, ZooKeeper)
- Shared state store (Redis, database)
- Global event pub/sub (NATS, Kafka)
- Session migration protocol

---

## 11. Security Model

### 11.1 Permission System

**Agent-level permissions**:
```typescript
agent.permission = {
  edit: boolean,     // Edit files
  bash: boolean,     // Run bash commands
  webfetch: boolean, // Fetch web content
}
```

**Tool inheritance**: MCP tools inherit agent permissions.

### 11.2 MCP Security

**Limitations**:
- No built-in authentication
- Custom headers for auth (user-provided)
- Subprocess isolation for local servers
- No sandboxing beyond process boundaries

**Warnings**:
- MCP servers run with full process privileges
- No capability-based security
- Trust model: User configures, system executes

### 11.3 LSP Security

**Isolation**:
- Language servers run as subprocesses
- Stderr suppressed (line 190 in lsp/client.ts)
- Environment variables controllable
- No network access restrictions

### 11.4 File Access

**No access control** beyond filesystem permissions:
- Read/Write tools access any file
- No chroot or jail
- No path traversal protection
- Relies on filesystem permissions

---

## 12. Performance Characteristics

### 12.1 Bottlenecks

| Component | Bottleneck | Impact |
|-----------|------------|--------|
| **Session Processing** | Sequential per session | One message at a time |
| **File I/O** | Writer blocks all readers | Lock contention |
| **LSP Startup** | Server spawn + initialization | 1-5s delay |
| **MCP Tool Fetch** | 5s timeout default | Startup latency |
| **File Tree** | 200 file limit | Incomplete context |

### 12.2 Optimizations

**Prompt Caching**:
- System prompts limited to 2 messages
- First message: Header + base prompt
- Second message: Environment + custom instructions
- Enables provider-level caching

**LSP Reuse**:
- Clients cached by (root, serverID)
- Inflight spawns deduplicated
- Broken servers tracked to avoid retry

**MCP Reuse**:
- Clients cached in Instance.state
- Tools fetched once per server
- Cleanup on Instance.dispose()

**Parallel Tool Calls**:
- LLM can invoke multiple tools simultaneously
- Independent operations execute in parallel
- Results aggregated before response

### 12.3 Scalability

**Vertical Scaling**:
- Single process per directory
- Concurrent sessions across directories
- Memory grows with active sessions

**Horizontal Scaling**:
- Not supported (stateful architecture)
- Requires session affinity or refactoring
- See [Section 10.3](#103-deployment-requirements)

---

## 13. Design Decisions

### 13.1 Language-Agnostic Prompts

**Decision**: No language-specific instructions in system prompts

**Rationale**:
- Models have inherent language knowledge
- Project structure provides context
- Users can add custom instructions
- Reduces prompt complexity
- Enables universal workflows

**Trade-off**: May miss language-specific best practices

### 13.2 File-Based Storage

**Decision**: JSON files instead of database

**Benefits**:
- Simple deployment (no DB setup)
- Human-readable format
- Easy backup/sync
- Version control friendly

**Trade-offs**:
- No ACID transactions
- No complex queries
- Manual indexing
- Lock limitations

### 13.3 In-Memory Locking

**Decision**: Process-local locks instead of OS locks

**Rationale**:
- Simpler implementation
- Faster (no syscalls)
- Sufficient for single-server

**Trade-off**: Prevents horizontal scaling

### 13.4 Sequential Session Processing

**Decision**: One message at a time per session

**Rationale**:
- Prevents context confusion
- Simpler state management
- Natural conversation flow
- Easier error recovery

**Trade-off**: Lower throughput per session

### 13.5 No Historical Replay

**Decision**: Pull-based history, push-based updates

**Rationale**:
- No event storage overhead
- Clients control what they fetch
- Reduces server memory
- Simplifies event bus

**Trade-off**: Requires explicit sync on connect

### 13.6 Two-Message System Prompt

**Decision**: Combine environment + custom into one message

**Rationale**:
- Enables prompt caching
- Most providers cache by message prefix
- 2-message structure maximizes cache hits

**Trade-off**: Less granular caching control

---

## 14. Future Considerations

### 14.1 Potential Enhancements

**Horizontal Scaling**:
- Distributed locking (Redis/etcd)
- Shared state store
- Event streaming (Kafka/NATS)
- Session migration

**Storage Improvements**:
- SQLite for indexed queries
- Compression for old sessions
- Configurable retention policies
- Backup/restore tools

**Security Enhancements**:
- MCP authentication framework
- Capability-based security
- Path allowlist/denylist
- Audit logging

**Performance Optimizations**:
- Incremental file tree updates
- Lazy LSP server loading
- Tool result streaming
- Parallel session processing

### 14.2 Architectural Evolution

**Phase 1: Current** (Single-server stateful)
- ✅ Simple deployment
- ✅ Strong consistency
- ❌ No horizontal scaling

**Phase 2: Stateless with sticky sessions**
- ✅ Multiple servers
- ✅ Session affinity
- ⚠️ Requires load balancer

**Phase 3: Fully distributed**
- ✅ True horizontal scaling
- ✅ High availability
- ❌ Significant complexity increase
- ❌ Eventual consistency challenges

**Recommendation**: Phase 1 sufficient for most deployments.

---

## Conclusion

OpenCode demonstrates a pragmatic architecture that prioritizes:

1. **Simplicity**: File-based storage, in-memory state
2. **Correctness**: Sequential processing, explicit locking
3. **Extensibility**: MCP/LSP plugin systems
4. **Developer Experience**: Rich tooling, comprehensive prompts

The stateful design trades horizontal scalability for implementation simplicity and operational correctness. For most deployment scenarios (individual developers, small teams), this is an appropriate trade-off.

The architecture's main strength is its **comprehensive integration**: LSP for code intelligence, MCP for extensible tools, and sophisticated prompt engineering for model guidance. These combine to create a powerful AI coding assistant that understands project context deeply.

**Key Takeaway**: OpenCode is designed for single-server deployments with session affinity, not for large-scale multi-tenancy. Its architecture excels at providing rich, context-aware assistance with strong consistency guarantees.

---

## Appendix A: File Reference Index

| Component | Primary Files |
|-----------|---------------|
| **Session Management** | `src/session/index.ts`, `src/session/prompt.ts` |
| **MCP Integration** | `src/mcp/index.ts`, `src/mcp/client.ts` |
| **LSP Integration** | `src/lsp/index.ts`, `src/lsp/client.ts`, `src/lsp/server.ts` |
| **Storage Layer** | `src/storage/storage.ts` |
| **Locking** | `src/util/lock.ts` |
| **Event System** | `src/bus/index.ts`, `src/bus/global.ts` |
| **System Prompt** | `src/session/system.ts`, `src/session/prompt/*.txt` |
| **HTTP Server** | `src/server/server.ts` |
| **Configuration** | `src/config/config.ts` |
| **Project Instance** | `src/project/instance.ts`, `src/project/state.ts` |

---

## Appendix B: Event Type Reference

```typescript
// Session Events
"session.created" → { info: Session.Info }
"session.updated" → { info: Session.Info }
"session.deleted" → { info: Session.Info }
"session.diff" → { sessionID, diff }
"session.error" → { sessionID, error }

// Message Events
"message.updated" → { info: MessageV2.Info }
"message.removed" → { sessionID, messageID }
"message.part.updated" → { part, delta }
"message.part.removed" → { messageID, partID }

// LSP Events
"lsp.updated" → {}
"lsp.client.diagnostics" → { serverID, path }

// Server Events (SSE)
"server.connected" → {}
```

---

## Appendix C: Configuration Schema

```jsonc
{
  // MCP Server Configuration
  "mcp": {
    "server-name": {
      "type": "local" | "remote",
      "command": ["cmd", "args"],        // local only
      "url": "https://...",              // remote only
      "headers": {},                     // remote only
      "environment": {},                 // local only
      "enabled": true,
      "timeout": 5000
    }
  },

  // LSP Server Configuration
  "lsp": {
    "server-id": {
      "disabled": false,
      "command": ["lsp-server"],
      "extensions": [".ext"],
      "env": {},
      "initialization": {}
    }
  },

  // Custom Instructions
  "instructions": [
    "~/global-instructions.md",
    "project-specific.md"
  ]
}
```

---

**Document Version**: 1.0
**Last Updated**: November 2024
**Based on OpenCode**: Latest main branch analysis
