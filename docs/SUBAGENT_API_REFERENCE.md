# OpenCode Subagent & Task Management API Reference

This document provides a comprehensive reference for all client and server APIs related to subagents and task management in OpenCode.

## Table of Contents

1. [Existing Clients & SDK](#existing-clients--sdk)
2. [Architecture Overview](#architecture-overview)
3. [Server-Side APIs](#server-side-apis)
4. [Client-Side APIs](#client-side-apis)
5. [Event System](#event-system)
6. [New Client Implementation Guide](#new-client-implementation-guide)

---

## Existing Clients & SDK

OpenCode already provides multiple client implementations and a generated SDK:

### Packages Overview

| Package | Type | Description |
|---------|------|-------------|
| `packages/opencode` | Core | Main OpenCode server and TUI client |
| `packages/desktop` | Web Client | SolidJS web client for browser/Electron |
| `packages/sdk/js` | SDK | Generated TypeScript SDK from OpenAPI |
| `packages/ui` | Components | Shared UI component library |
| `packages/console` | Console | Management console web app |
| `packages/tauri` | Desktop | Tauri-based desktop application |
| `packages/enterprise` | Enterprise | Enterprise features |
| `sdks/vscode` | IDE | VS Code extension |

### Generated SDK (`@opencode-ai/sdk`)

The SDK is auto-generated from OpenAPI specs using `@hey-api/openapi-ts`:

```typescript
import { createOpencodeClient } from "@opencode-ai/sdk/client"

const client = createOpencodeClient({
  baseUrl: "http://localhost:4096",
  directory: "/path/to/project",
})

// All methods are typed and available
const sessions = await client.session.list()
const session = await client.session.create()
const messages = await client.session.messages({ path: { id: session.id } })
```

**SDK Classes:**
- `Global` - Global events
- `Project` - Project management
- `Config` - Configuration
- `Tool` - Tool management
- `Instance` - Instance control
- `Path` - Path utilities
- `Session` - Session CRUD and messaging
- `Command` - Commands
- `Provider` - Model providers
- `Find` - Search functionality
- `File` - File operations
- `App` - App info and agents
- `Mcp` - MCP server management
- `Lsp` - LSP status
- `Formatter` - Formatter status
- `Tui` - TUI control
- `Auth` - Authentication
- `Event` - Event subscription

### Desktop Web Client (`packages/desktop`)

A full SolidJS web application with:

- **Session management** - Create, list, navigate sessions
- **Message rendering** - Real-time streaming messages with `<SessionTurn>`
- **File browser** - Open, view, and edit files
- **Diff review** - Side-by-side and unified diff views
- **Drag-and-drop tabs** - Reorderable file tabs
- **Progress tracking** - Context usage and token counts
- **Keyboard shortcuts** - Vim-style navigation

```typescript
// Desktop client uses the SDK
import { createOpencodeClient } from "@opencode-ai/sdk/client"
import { useSDK, SDKProvider } from "./context/sdk"

// Context provides SDK to all components
const { client, event } = useSDK()

// Make API calls
const session = await client.session.create()
await client.session.prompt({
  path: { id: session.id },
  body: { parts: [{ type: "text", text: "Hello" }] }
})
```

---

## Architecture Overview

OpenCode uses a parent-child session architecture for subagent management:

```
Parent Session (sessionID: "session_abc")
│
├─ User Message
├─ Assistant Response
│   └─ Task Tool Invocation
│       ├─ Child Session 1 (parentID: "session_abc")
│       ├─ Child Session 2 (parentID: "session_abc")
│       └─ Child Session 3 (parentID: "session_abc")
│
└─ Results aggregated back to parent
```

### Key Concepts

- **Session**: Container for a conversation with messages and parts
- **Message**: User or assistant turn in a session
- **Part**: Individual content blocks (text, tool calls, reasoning, etc.)
- **Agent**: Configuration for AI behavior (primary, subagent, or all modes)
- **Task Tool**: Mechanism for spawning child sessions

---

## Server-Side APIs

### Session Management

#### Session.create()

Creates a new session, optionally as a child of another session.

**Location:** `packages/opencode/src/session/index.ts:122-136`

```typescript
const create = fn(
  z.object({
    parentID: Identifier.schema("session").optional(),
    title: z.string().optional(),
  }).optional(),
  async (input) => Session.Info
)
```

**HTTP Endpoint:** `POST /session`

**Request Body:**
```json
{
  "parentID": "session_abc123",  // Optional: parent for child sessions
  "title": "My Session"          // Optional: custom title
}
```

**Response:** `Session.Info`

---

#### Session.get()

Retrieves a session by ID.

**Location:** `packages/opencode/src/session/index.ts:210-213`

```typescript
const get = fn(Identifier.schema("session"), async (id) => Session.Info)
```

**HTTP Endpoint:** `GET /session/:id`

---

#### Session.list()

Lists all sessions in the current project.

**Location:** `packages/opencode/src/session/index.ts:303-308`

```typescript
async function* list(): AsyncGenerator<Session.Info>
```

**HTTP Endpoint:** `GET /session`

---

#### Session.update()

Updates session properties.

**Location:** `packages/opencode/src/session/index.ts:270-280`

```typescript
async function update(
  id: string,
  editor: (session: Info) => void
): Promise<Session.Info>
```

**HTTP Endpoint:** `PATCH /session/:id`

---

#### Session.remove()

Deletes a session and all its children.

**Location:** `packages/opencode/src/session/index.ts:321-342`

```typescript
const remove = fn(Identifier.schema("session"), async (sessionID) => void)
```

**HTTP Endpoint:** `DELETE /session/:id`

---

#### Session.fork()

Creates a new session by copying messages up to a point.

**Location:** `packages/opencode/src/session/index.ts:138-167`

```typescript
const fork = fn(
  z.object({
    sessionID: Identifier.schema("session"),
    messageID: Identifier.schema("message").optional(),
  }),
  async (input) => Session.Info
)
```

**HTTP Endpoint:** `POST /session/:id/fork`

---

#### Session.children()

Gets all child sessions of a parent.

**HTTP Endpoint:** `GET /session/:id/children`

---

#### Session.messages()

Retrieves messages for a session.

**Location:** `packages/opencode/src/session/index.ts:287-301`

```typescript
const messages = fn(
  z.object({
    sessionID: Identifier.schema("session"),
    limit: z.number().optional(),
  }),
  async (input) => MessageV2.WithParts[]
)
```

**HTTP Endpoint:** `GET /session/:id/message?limit=<n>`

---

### Session.Info Schema

```typescript
const Info = z.object({
  id: Identifier.schema("session"),
  projectID: z.string(),
  directory: z.string(),
  parentID: Identifier.schema("session").optional(),
  summary: z.object({
    additions: z.number(),
    deletions: z.number(),
    files: z.number(),
    diffs: Snapshot.FileDiff.array().optional(),
  }).optional(),
  share: z.object({ url: z.string() }).optional(),
  title: z.string(),
  version: z.string(),
  time: z.object({
    created: z.number(),
    updated: z.number(),
    compacting: z.number().optional(),
  }),
  revert: z.object({
    messageID: z.string(),
    partID: z.string().optional(),
    snapshot: z.string().optional(),
    diff: z.string().optional(),
  }).optional(),
})
```

---

### Prompt Execution

#### SessionPrompt.prompt()

Creates a user message and starts the execution loop.

**Location:** `packages/opencode/src/session/prompt.ts:193-205`

```typescript
const PromptInput = z.object({
  sessionID: Identifier.schema("session"),
  messageID: Identifier.schema("message").optional(),
  model: z.object({
    providerID: z.string(),
    modelID: z.string(),
  }).optional(),
  agent: z.string().optional(),
  noReply: z.boolean().optional(),
  system: z.string().optional(),
  tools: z.record(z.string(), z.boolean()).optional(),
  parts: z.array(TextPart | FilePart | AgentPart | SubtaskPart),
})

const prompt = fn(PromptInput, async (input) => MessageV2.WithParts)
```

**HTTP Endpoint:** `POST /session/:id/message` (streams JSON)

---

#### SessionPrompt.loop()

Main execution loop for processing agent responses.

**Location:** `packages/opencode/src/session/prompt.ts:232-612`

```typescript
const loop = fn(Identifier.schema("session"), async (sessionID) => MessageV2.WithParts)
```

**Execution Flow:**
1. Fetch last user & assistant messages
2. Check for pending subtasks/compaction
3. Resolve system prompts & tools
4. Stream text from LLM
5. Process tool calls
6. Handle errors and retries
7. Continue until completion

---

#### SessionPrompt.command()

Executes a slash command.

**Location:** `packages/opencode/src/session/prompt.ts:1292-1396`

```typescript
const CommandInput = z.object({
  messageID: Identifier.schema("message").optional(),
  sessionID: Identifier.schema("session"),
  agent: z.string().optional(),
  model: z.string().optional(),
  arguments: z.string(),
  command: z.string(),
})

async function command(input: CommandInput): Promise<MessageV2.WithParts>
```

**HTTP Endpoint:** `POST /session/:id/command`

---

#### SessionPrompt.shell()

Executes a shell command and records output.

**Location:** `packages/opencode/src/session/prompt.ts:1106-1290`

```typescript
const ShellInput = z.object({
  sessionID: Identifier.schema("session"),
  agent: z.string(),
  model: z.object({
    providerID: z.string(),
    modelID: z.string(),
  }).optional(),
  command: z.string(),
})

async function shell(input: ShellInput): Promise<MessageV2.Assistant>
```

**HTTP Endpoint:** `POST /session/:id/shell`

---

### Task Tool API

The Task tool enables spawning subagent sessions.

**Location:** `packages/opencode/src/tool/task.ts:13-115`

#### Parameters

```typescript
z.object({
  description: z.string(),      // Short task description (3-5 words)
  prompt: z.string(),           // Full task prompt
  subagent_type: z.string(),    // Agent name (e.g., "general")
  session_id: z.string().optional(), // Continue existing session
})
```

#### Return Value

```typescript
{
  title: string,
  metadata: {
    summary: ToolPart[],
    sessionId: string,
  },
  output: string,
}
```

#### Execution Flow

1. Get subagent configuration by type
2. Create child session (or reuse existing)
3. Execute `SessionPrompt.prompt()` in child session
4. Monitor tool execution via Bus subscription
5. Return output with task metadata

---

### Agent APIs

#### Agent.get()

**Location:** `packages/opencode/src/agent/agent.ts:182-184`

```typescript
async function get(agent: string): Promise<Agent.Info | undefined>
```

---

#### Agent.list()

**Location:** `packages/opencode/src/agent/agent.ts:186-188`

```typescript
async function list(): Promise<Agent.Info[]>
```

**HTTP Endpoint:** `GET /agent`

---

#### Agent.Info Schema

```typescript
const Info = z.object({
  name: z.string(),
  description: z.string().optional(),
  mode: z.enum(["subagent", "primary", "all"]),
  builtIn: z.boolean(),
  topP: z.number().optional(),
  temperature: z.number().optional(),
  color: z.string().optional(),
  permission: z.object({
    edit: Config.Permission,
    bash: z.record(z.string(), Config.Permission),
    webfetch: Config.Permission.optional(),
    doom_loop: Config.Permission.optional(),
    external_directory: Config.Permission.optional(),
  }),
  model: z.object({
    modelID: z.string(),
    providerID: z.string(),
  }).optional(),
  prompt: z.string().optional(),
  tools: z.record(z.string(), z.boolean()),
  options: z.record(z.string(), z.any()),
})
```

**Agent Modes:**
- `primary` - User-selectable, initiates conversations
- `subagent` - Called by other agents for subtasks
- `all` - Can function as both

---

### Message APIs

#### MessageV2.Info Schema

```typescript
// User message
const User = Base.extend({
  role: z.literal("user"),
  time: z.object({ created: z.number() }),
  agent: z.string(),
  model: z.object({ providerID: z.string(), modelID: z.string() }),
})

// Assistant message
const Assistant = Base.extend({
  role: z.literal("assistant"),
  time: z.object({ created: z.number(), completed: z.number().optional() }),
  error: z.discriminatedUnion("name", [...]).optional(),
  parentID: z.string(),
  modelID: z.string(),
  providerID: z.string(),
  mode: z.string(),
  path: z.object({ cwd: z.string(), root: z.string() }),
  cost: z.number(),
  tokens: z.object({
    input: z.number(),
    output: z.number(),
    reasoning: z.number(),
    cache: z.object({ read: z.number(), write: z.number() }),
  }),
})
```

---

#### Message Part Types

| Type | Description | Key Fields |
|------|-------------|------------|
| `TextPart` | Plain text output | `text`, `synthetic` |
| `ReasoningPart` | Extended thinking | `text`, `time` |
| `FilePart` | File references | `filename`, `mime` |
| `ToolPart` | Tool invocations | `tool`, `state`, `callID` |
| `SnapshotPart` | Filesystem snapshots | `snapshot` |
| `PatchPart` | Diff patches | `hash`, `files` |
| `SubtaskPart` | Subtask references | `prompt`, `agent` |
| `StepStartPart` | Step markers | `snapshot` |
| `StepFinishPart` | Step completion | `cost`, `tokens` |

---

### HTTP Endpoints Summary

#### Session Endpoints

| Method | Path | Operation |
|--------|------|-----------|
| POST | `/session` | Create session |
| GET | `/session` | List sessions |
| GET | `/session/:id` | Get session |
| PATCH | `/session/:id` | Update session |
| DELETE | `/session/:id` | Delete session |
| GET | `/session/:id/children` | Get children |
| POST | `/session/:id/fork` | Fork session |
| POST | `/session/:id/share` | Share session |
| POST | `/session/:id/abort` | Abort execution |

#### Message Endpoints

| Method | Path | Operation |
|--------|------|-----------|
| GET | `/session/:id/message` | List messages |
| GET | `/session/:id/message/:msgID` | Get message |
| POST | `/session/:id/message` | Create & execute |
| POST | `/session/:id/command` | Execute command |
| POST | `/session/:id/shell` | Execute shell |
| POST | `/session/:id/revert` | Revert message |

#### Event Endpoints

| Method | Path | Operation |
|--------|------|-----------|
| GET | `/event` | Subscribe to events (SSE) |
| GET | `/global/event` | Global events (SSE) |
| GET | `/session/status` | Session status |

---

## Client-Side APIs

### Bus/Event System

The Bus system provides typed pub/sub messaging.

**Location:** `packages/opencode/src/bus/index.ts`

#### Bus.event()

Define a typed event.

```typescript
function event<Type extends string, Properties extends ZodType>(
  type: Type,
  properties: Properties
): EventDefinition
```

**Example:**
```typescript
const Created = Bus.event("session.created", z.object({ info: Session.Info }))
```

---

#### Bus.publish()

Broadcast an event to all subscribers.

```typescript
async function publish<Definition extends EventDefinition>(
  def: Definition,
  properties: z.output<Definition["properties"]>
): Promise<void[]>
```

**Example:**
```typescript
await Bus.publish(Session.Event.Created, { info: newSession })
```

---

#### Bus.subscribe()

Listen for specific events.

```typescript
function subscribe<Definition extends EventDefinition>(
  def: Definition,
  callback: (event: EventPayload) => void
): () => void // Returns unsubscribe function
```

**Example:**
```typescript
const unsubscribe = Bus.subscribe(Session.Event.Created, (event) => {
  console.log("Session created:", event.properties.info.id)
})
```

---

#### Bus.once()

One-time event listener.

```typescript
function once<Definition extends EventDefinition>(
  def: Definition,
  callback: (event: EventPayload) => "done" | undefined
): void
```

---

#### Bus.subscribeAll()

Listen to all events (wildcard).

```typescript
function subscribeAll(callback: (event: any) => void): () => void
```

---

### Defined Events

#### Session Events

```typescript
const Event = {
  Created: Bus.event("session.created", z.object({ info: Info })),
  Updated: Bus.event("session.updated", z.object({ info: Info })),
  Deleted: Bus.event("session.deleted", z.object({ info: Info })),
  Diff: Bus.event("session.diff", z.object({
    sessionID: z.string(),
    diff: Snapshot.FileDiff.array(),
  })),
  Error: Bus.event("session.error", z.object({
    sessionID: z.string().optional(),
    error: MessageV2.Assistant.shape.error,
  })),
}
```

#### Message Events

```typescript
const Event = {
  Updated: Bus.event("message.updated", z.object({ info: Info })),
  Removed: Bus.event("message.removed", z.object({
    sessionID: z.string(),
    messageID: z.string(),
  })),
  PartUpdated: Bus.event("message.part.updated", z.object({
    part: Part,
    delta: z.string().optional(),
  })),
  PartRemoved: Bus.event("message.part.removed", z.object({
    sessionID: z.string(),
    messageID: z.string(),
    partID: z.string(),
  })),
}
```

---

### Storage API

File-based JSON storage system.

**Location:** `packages/opencode/src/storage/storage.ts`

#### Storage.read()

```typescript
async function read<T>(key: string[]): Promise<T>
```

**Example:**
```typescript
const session = await Storage.read<Session.Info>(["session", projectID, sessionID])
```

---

#### Storage.write()

```typescript
async function write<T>(key: string[], content: T): Promise<void>
```

---

#### Storage.update()

Atomic read-modify-write.

```typescript
async function update<T>(
  key: string[],
  fn: (draft: T) => void
): Promise<T>
```

---

#### Storage.list()

List records by prefix.

```typescript
async function list(prefix: string[]): Promise<string[][]>
```

**Example:**
```typescript
const sessions = await Storage.list(["session", projectID])
// Returns: [["session", "proj_abc", "sess_123"], ...]
```

---

### Provider API

Model and provider management.

**Location:** `packages/opencode/src/provider/provider.ts`

#### Provider.getModel()

```typescript
async function getModel(
  providerID: string,
  modelID: string
): Promise<{
  modelID: string
  providerID: string
  info: ModelsDev.Model
  language: LanguageModel
  npm?: string
}>
```

---

#### Provider.list()

```typescript
async function list(): Promise<{
  [providerID: string]: {
    source: Source
    info: ModelsDev.Provider
    options: Record<string, any>
  }
}>
```

---

#### Provider.defaultModel()

```typescript
async function defaultModel(): Promise<{
  providerID: string
  modelID: string
}>
```

---

### Worker/RPC API

For multi-process communication.

**Location:** `packages/opencode/src/util/rpc.ts`

#### Rpc.listen()

Server-side RPC handler (in worker).

```typescript
function listen(rpc: Definition): void
```

**Example:**
```typescript
Rpc.listen({
  async server(input: { port: number }) {
    return { url: `http://localhost:${input.port}` }
  },
})
```

---

#### Rpc.client()

Client-side RPC caller (main thread).

```typescript
function client<T extends Definition>(target: Worker): {
  call<Method extends keyof T>(
    method: Method,
    input: Parameters<T[Method]>[0]
  ): Promise<ReturnType<T[Method]>>
}
```

**Example:**
```typescript
const client = Rpc.client<typeof rpc>(worker)
const result = await client.call("server", { port: 3000 })
```

---

### State Management Contexts

TUI state management using Solid.js contexts.

#### useSDK()

SDK client and event subscription.

**Location:** `packages/opencode/src/cli/cmd/tui/context/sdk.tsx`

```typescript
const { client, event } = useSDK()
// client: OpencodeClient - HTTP client for API calls
// event: EventEmitter - Batched event emissions
```

---

#### useSync()

Global state synchronization.

**Location:** `packages/opencode/src/cli/cmd/tui/context/sync.tsx`

```typescript
const sync = useSync()

// Access data
sync.data.session       // Session[]
sync.data.message       // { [sessionID]: Message[] }
sync.data.part          // { [messageID]: Part[] }
sync.data.agent         // Agent[]
sync.data.provider      // Provider[]
sync.data.permission    // { [sessionID]: Permission[] }

// Session utilities
sync.session.get(id)    // Get session by ID
sync.session.status(id) // "idle" | "working" | "compacting"
await sync.session.sync(id) // Fetch messages for session

// Bootstrap
await sync.bootstrap()  // Load initial data
```

---

#### useLocal()

Local preferences (model, agent).

**Location:** `packages/opencode/src/cli/cmd/tui/context/local.tsx`

```typescript
const local = useLocal()

// Model management
local.model.current()   // Current model
local.model.set(model)  // Set model
local.model.cycle(1)    // Cycle to next model

// Agent management
local.agent.current()   // Current agent
local.agent.set(name)   // Set agent
local.agent.list()      // Available agents
```

---

#### useRoute()

Navigation state.

**Location:** `packages/opencode/src/cli/cmd/tui/context/route.tsx`

```typescript
const route = useRoute()

route.data              // Current route
route.navigate({ type: "session", sessionID: "..." })
```

---

## Event System

### Event Flow

```
Tool Execution / State Change
         ↓
   Bus.publish()
         ↓
   GlobalBus.emit()  →  Other processes
         ↓
   Local subscribers
         ↓
   SSE to HTTP clients
```

### Subscribing via HTTP (SSE)

```typescript
const eventSource = new EventSource("/event")
eventSource.onmessage = (e) => {
  const event = JSON.parse(e.data)
  switch (event.type) {
    case "session.created":
      handleSessionCreated(event.properties.info)
      break
    case "message.part.updated":
      handlePartUpdated(event.properties.part)
      break
  }
}
```

### Event Types for Subagent Monitoring

| Event | Description | Payload |
|-------|-------------|---------|
| `session.created` | Child session created | `{ info: Session.Info }` |
| `message.updated` | Message state changed | `{ info: MessageV2.Info }` |
| `message.part.updated` | Part updated (streaming) | `{ part: Part, delta?: string }` |
| `session.diff` | File changes | `{ sessionID, diff: FileDiff[] }` |
| `session.error` | Error occurred | `{ sessionID?, error }` |

---

## New Client Implementation Guide

### Minimum Required APIs

To build a new client with full subagent support, implement these core integrations:

#### 1. Session Management

```typescript
interface SessionClient {
  create(input?: { parentID?: string; title?: string }): Promise<Session.Info>
  get(id: string): Promise<Session.Info>
  list(): Promise<Session.Info[]>
  children(id: string): Promise<Session.Info[]>
  remove(id: string): Promise<void>
}
```

#### 2. Message Execution

```typescript
interface MessageClient {
  prompt(input: {
    sessionID: string
    parts: Part[]
    agent?: string
    model?: { providerID: string; modelID: string }
  }): Promise<MessageV2.WithParts>

  messages(sessionID: string, limit?: number): Promise<MessageV2.WithParts[]>
  abort(sessionID: string): Promise<void>
}
```

#### 3. Event Subscription

```typescript
interface EventClient {
  subscribe(callback: (event: BusEvent) => void): () => void

  // Or via SSE
  connect(): EventSource
}
```

#### 4. Agent Configuration

```typescript
interface AgentClient {
  list(): Promise<Agent.Info[]>
  get(name: string): Promise<Agent.Info | undefined>
}
```

### Implementation Example

```typescript
class OpencodeClient {
  private baseUrl: string
  private eventSource?: EventSource

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
  }

  // Session APIs
  async createSession(parentID?: string): Promise<Session.Info> {
    const res = await fetch(`${this.baseUrl}/session`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ parentID }),
    })
    return res.json()
  }

  async getSession(id: string): Promise<Session.Info> {
    const res = await fetch(`${this.baseUrl}/session/${id}`)
    return res.json()
  }

  async listSessions(): Promise<Session.Info[]> {
    const res = await fetch(`${this.baseUrl}/session`)
    return res.json()
  }

  async getChildren(sessionID: string): Promise<Session.Info[]> {
    const res = await fetch(`${this.baseUrl}/session/${sessionID}/children`)
    return res.json()
  }

  // Message APIs
  async prompt(input: {
    sessionID: string
    parts: Part[]
    agent?: string
  }): Promise<MessageV2.WithParts> {
    const res = await fetch(`${this.baseUrl}/session/${input.sessionID}/message`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    })
    return res.json()
  }

  async getMessages(sessionID: string): Promise<MessageV2.WithParts[]> {
    const res = await fetch(`${this.baseUrl}/session/${sessionID}/message`)
    return res.json()
  }

  // Event subscription
  subscribeToEvents(callback: (event: any) => void): () => void {
    this.eventSource = new EventSource(`${this.baseUrl}/event`)

    this.eventSource.onmessage = (e) => {
      callback(JSON.parse(e.data))
    }

    return () => {
      this.eventSource?.close()
    }
  }

  // Agent APIs
  async listAgents(): Promise<Agent.Info[]> {
    const res = await fetch(`${this.baseUrl}/agent`)
    return res.json()
  }
}
```

### Subagent Monitoring

To monitor subagent execution in real-time:

```typescript
class SubagentMonitor {
  private client: OpencodeClient
  private parentSessionID: string

  constructor(client: OpencodeClient, parentSessionID: string) {
    this.client = client
    this.parentSessionID = parentSessionID
  }

  async watchSubagents(callback: (event: SubagentEvent) => void): Promise<() => void> {
    const children = new Set<string>()

    // Get existing children
    const existing = await this.client.getChildren(this.parentSessionID)
    existing.forEach(s => children.add(s.id))

    // Subscribe to events
    return this.client.subscribeToEvents((event) => {
      switch (event.type) {
        case "session.created":
          if (event.properties.info.parentID === this.parentSessionID) {
            children.add(event.properties.info.id)
            callback({
              type: "child_created",
              session: event.properties.info,
            })
          }
          break

        case "message.part.updated":
          if (children.has(event.properties.part.sessionID)) {
            callback({
              type: "child_progress",
              sessionID: event.properties.part.sessionID,
              part: event.properties.part,
            })
          }
          break

        case "session.error":
          if (children.has(event.properties.sessionID)) {
            callback({
              type: "child_error",
              sessionID: event.properties.sessionID,
              error: event.properties.error,
            })
          }
          break
      }
    })
  }
}
```

### Key Considerations for New Clients

1. **Streaming Support**: Handle streaming responses for real-time output
2. **Event Batching**: Batch rapid events to avoid UI thrashing
3. **Session Tree Navigation**: Support parent-child relationships
4. **Permission Handling**: Respond to permission requests via `/session/:id/permissions/:permissionID`
5. **Error Recovery**: Handle network errors, retries, and reconnection
6. **Cost Tracking**: Aggregate costs across parent and child sessions

### Feature Matrix

| Feature | API Required | Complexity |
|---------|-------------|------------|
| Basic sessions | Session CRUD | Low |
| Message execution | POST /message | Medium |
| Real-time updates | SSE /event | Medium |
| Subagent spawning | Task tool | High |
| Permission handling | Permission endpoints | Medium |
| File diffs | Session.Diff events | Medium |
| Cost tracking | Message tokens | Low |
| Session sharing | Share endpoints | Low |

---

## Conclusion

OpenCode provides a comprehensive API surface for building clients with full subagent support:

- **15+ HTTP endpoints** for session and message management
- **10+ event types** for real-time monitoring
- **Typed schemas** with Zod validation
- **Parent-child session** architecture
- **Flexible agent configuration**

A new client can leverage these APIs to implement:
- Multi-session management
- Real-time streaming output
- Subagent progress monitoring
- Cost aggregation
- File change tracking
- Custom UI experiences

The modular architecture makes it straightforward to implement clients in any language or framework that supports HTTP and Server-Sent Events.
