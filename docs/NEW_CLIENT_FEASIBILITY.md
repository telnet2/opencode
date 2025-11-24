# New Client Feasibility Analysis

This document analyzes the feasibility of building a new client for OpenCode with full subagent and task management support.

## Executive Summary

**Verdict: Highly Feasible**

OpenCode's architecture is well-suited for alternative client implementations. The HTTP API is comprehensive, events are streamed via SSE, and all schemas are well-defined with Zod.

**Important:** OpenCode already has:
- A **generated TypeScript SDK** (`@opencode-ai/sdk`) with all API methods
- A **SolidJS web client** (`packages/desktop`) with full subagent support
- A **TUI client** in the core package

New clients can either use the existing SDK (for TypeScript/JavaScript) or implement their own HTTP client based on the OpenAPI spec.

---

## Current Architecture

### Communication Patterns

```
┌─────────────┐         HTTP/SSE          ┌──────────────┐
│   Client    │ ◄─────────────────────►   │    Server    │
│  (TUI/Web)  │                           │   (Hono)     │
└─────────────┘                           └──────────────┘
                                                  │
                                          ┌───────┴───────┐
                                          │               │
                                    ┌─────▼─────┐   ┌─────▼─────┐
                                    │  Session  │   │   Agent   │
                                    │  Manager  │   │  Executor │
                                    └───────────┘   └───────────┘
```

### Key Components

| Component | Role | Client Access |
|-----------|------|---------------|
| Server (Hono) | HTTP API gateway | Direct HTTP |
| Bus | Event pub/sub | SSE streaming |
| Storage | Persistence | Via API only |
| Session Manager | Session CRUD | HTTP endpoints |
| Prompt Executor | LLM execution | POST /message |
| Agent Registry | Agent config | GET /agent |

---

## API Completeness Analysis

### Session Management: Complete

| Operation | Endpoint | Status |
|-----------|----------|--------|
| Create | POST /session | Available |
| Read | GET /session/:id | Available |
| List | GET /session | Available |
| Update | PATCH /session/:id | Available |
| Delete | DELETE /session/:id | Available |
| Fork | POST /session/:id/fork | Available |
| Children | GET /session/:id/children | Available |
| Share | POST /session/:id/share | Available |

### Message Execution: Complete

| Operation | Endpoint | Status |
|-----------|----------|--------|
| Create & Execute | POST /session/:id/message | Available (streams) |
| List Messages | GET /session/:id/message | Available |
| Get Message | GET /session/:id/message/:msgID | Available |
| Execute Command | POST /session/:id/command | Available |
| Execute Shell | POST /session/:id/shell | Available |
| Abort | POST /session/:id/abort | Available |
| Revert | POST /session/:id/revert | Available |

### Event Streaming: Complete

| Operation | Endpoint | Status |
|-----------|----------|--------|
| Session Events | GET /event | SSE stream |
| Global Events | GET /global/event | SSE stream |
| Status Polling | GET /session/status | Available |

### Agent Configuration: Complete

| Operation | Endpoint | Status |
|-----------|----------|--------|
| List Agents | GET /agent | Available |
| Permissions | POST /session/:id/permissions/:id | Available |

---

## New Client Capabilities

### Tier 1: Basic Client (1-2 weeks)

**Features:**
- Session CRUD
- Message sending/receiving
- Basic streaming output
- Agent selection

**APIs Required:**
- POST/GET/DELETE /session
- POST /session/:id/message
- GET /session/:id/message

**Complexity:** Low

---

### Tier 2: Full-Featured Client (3-4 weeks)

**Additional Features:**
- Real-time event streaming
- Subagent monitoring
- File diff visualization
- Permission handling
- Session forking

**APIs Required:**
- All Tier 1 APIs
- GET /event (SSE)
- GET /session/:id/children
- POST /session/:id/fork
- POST /session/:id/permissions/:id

**Complexity:** Medium

---

### Tier 3: Advanced Client (5-8 weeks)

**Additional Features:**
- Custom agent creation
- Model management
- Cost analytics
- Session sharing
- Compaction handling

**APIs Required:**
- All Tier 2 APIs
- Full event handling
- Share endpoints
- Usage aggregation logic

**Complexity:** High

---

## Implementation Approaches

### Approach 1: Use Existing SDK (TypeScript/JavaScript)

For TypeScript/JavaScript projects, use the existing generated SDK:

```typescript
import { createOpencodeClient } from "@opencode-ai/sdk/client"

const client = createOpencodeClient({
  baseUrl: "http://localhost:4096",
  directory: "/path/to/project",
})

// Full type safety and all methods available
const session = await client.session.create()
const response = await client.session.prompt({
  path: { id: session.id },
  body: { parts: [{ type: "text", text: "Hello" }] }
})
```

**Pros:**
- Pre-built, tested, and maintained
- Full TypeScript types
- Generated from OpenAPI spec
- Handles authentication and headers

**Cons:**
- TypeScript/JavaScript only

**Recommended for:** Web apps, Electron apps, Node.js tools, VS Code extensions

---

### Approach 2: HTTP Client Only (Other Languages)

**Pros:**
- Simplest implementation
- Works in any language
- No special dependencies

**Cons:**
- Must poll for some operations
- No direct storage access

**Recommended for:** Python, Go, Rust clients, mobile apps, integrations

---

### Approach 3: WebSocket Enhancement

Currently OpenCode uses SSE for events. A WebSocket client could be built:

**Implementation:**
```typescript
// Wrap SSE in WebSocket adapter
class WebSocketAdapter {
  private sse: EventSource
  private ws: WebSocket

  connect() {
    this.sse = new EventSource("/event")
    this.sse.onmessage = (e) => {
      this.ws.send(e.data)
    }
  }
}
```

**Pros:**
- Bidirectional communication
- Better mobile support

**Cons:**
- Additional server changes needed

---

### Approach 4: Direct Integration

Import OpenCode modules directly:

```typescript
import { Session, SessionPrompt, Bus } from "@opencode/core"

// Direct access to all internals
const session = await Session.create({ title: "My Session" })
Bus.subscribe(Session.Event.Created, handleCreated)
```

**Pros:**
- Full access to internals
- Best performance
- No network overhead

**Cons:**
- Node.js/Bun only
- Tight coupling to internals

**Recommended for:** CLI tools, IDE plugins

---

## Language-Specific Implementations

### TypeScript/JavaScript

```typescript
import { OpencodeClient } from "@opencode/sdk"

const client = new OpencodeClient("http://localhost:3000")
const session = await client.session.create()
const message = await client.message.prompt({
  sessionID: session.id,
  parts: [{ type: "text", text: "Hello" }],
})
```

**Advantages:** Native types, existing SDK patterns

---

### Python

```python
import opencode

client = opencode.Client("http://localhost:3000")
session = client.sessions.create()
message = client.messages.prompt(
    session_id=session.id,
    parts=[{"type": "text", "text": "Hello"}]
)

# Event streaming
for event in client.events.stream():
    if event.type == "message.part.updated":
        print(event.properties.part.text)
```

**Advantages:** Large AI/ML ecosystem

---

### Go

```go
client := opencode.NewClient("http://localhost:3000")
session, _ := client.Sessions.Create(nil)
message, _ := client.Messages.Prompt(opencode.PromptInput{
    SessionID: session.ID,
    Parts: []opencode.Part{
        {Type: "text", Text: "Hello"},
    },
})

// Event streaming
events := client.Events.Subscribe()
for event := range events {
    switch e := event.(type) {
    case *opencode.MessagePartUpdated:
        fmt.Println(e.Part.Text)
    }
}
```

**Advantages:** Performance, concurrency

---

### Rust

```rust
let client = OpenCodeClient::new("http://localhost:3000");
let session = client.sessions().create(None).await?;
let message = client.messages().prompt(PromptInput {
    session_id: session.id,
    parts: vec![Part::Text { text: "Hello".into() }],
}).await?;

// Event streaming
let mut events = client.events().subscribe().await?;
while let Some(event) = events.next().await {
    match event {
        Event::MessagePartUpdated { part, .. } => {
            println!("{}", part.text);
        }
        _ => {}
    }
}
```

**Advantages:** Performance, safety, WASM support

---

## Client Architecture Patterns

### Pattern 1: Thin Client

```
┌─────────────────┐
│   Thin Client   │
│  (just HTTP)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  OpenCode API   │
└─────────────────┘
```

All logic in server. Client only renders.

**Use case:** Web dashboards, monitoring tools

---

### Pattern 2: Smart Client

```
┌─────────────────┐
│   Smart Client  │
│ ┌─────────────┐ │
│ │ Local State │ │
│ │   Cache     │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  OpenCode API   │
└─────────────────┘
```

Local state, caching, optimistic updates.

**Use case:** TUI, IDE plugins

---

### Pattern 3: Offline-First Client

```
┌─────────────────┐
│ Offline Client  │
│ ┌─────────────┐ │
│ │ Local Store │ │
│ │  (SQLite)   │ │
│ └─────────────┘ │
└────────┬────────┘
         │ Sync
         ▼
┌─────────────────┐
│  OpenCode API   │
└─────────────────┘
```

Full offline support with sync.

**Use case:** Mobile apps, distributed teams

---

## Feature Parity Matrix

| Feature | Current TUI | New Client Possible |
|---------|-------------|---------------------|
| Session management | Yes | Yes |
| Real-time streaming | Yes | Yes |
| Subagent monitoring | Yes | Yes |
| File diff view | Yes | Yes |
| Cost tracking | Yes | Yes |
| Permission dialogs | Yes | Yes |
| Vim keybindings | Yes | Implementation choice |
| Markdown rendering | Yes | Implementation choice |
| Syntax highlighting | Yes | Implementation choice |
| Theme customization | Yes | Implementation choice |
| Session navigation | Yes | Yes |

---

## Challenges and Solutions

### Challenge 1: Streaming Response Parsing

**Problem:** POST /message returns streaming JSON chunks.

**Solution:**
```typescript
async function* streamPrompt(input: PromptInput) {
  const response = await fetch(url, {
    method: "POST",
    body: JSON.stringify(input),
  })

  const reader = response.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split("\n")
    buffer = lines.pop()!

    for (const line of lines) {
      if (line.trim()) {
        yield JSON.parse(line)
      }
    }
  }
}
```

---

### Challenge 2: Event Reconnection

**Problem:** SSE connections can drop.

**Solution:**
```typescript
class ResilientEventSource {
  private url: string
  private eventSource?: EventSource
  private retryDelay = 1000

  connect() {
    this.eventSource = new EventSource(this.url)

    this.eventSource.onerror = () => {
      this.eventSource?.close()
      setTimeout(() => this.connect(), this.retryDelay)
      this.retryDelay = Math.min(this.retryDelay * 2, 30000)
    }

    this.eventSource.onopen = () => {
      this.retryDelay = 1000
    }
  }
}
```

---

### Challenge 3: Parent-Child Session Tracking

**Problem:** Need to track relationships for subagent monitoring.

**Solution:**
```typescript
class SessionTree {
  private sessions: Map<string, Session.Info> = new Map()
  private children: Map<string, Set<string>> = new Map()

  add(session: Session.Info) {
    this.sessions.set(session.id, session)
    if (session.parentID) {
      if (!this.children.has(session.parentID)) {
        this.children.set(session.parentID, new Set())
      }
      this.children.get(session.parentID)!.add(session.id)
    }
  }

  getChildren(id: string): Session.Info[] {
    const childIds = this.children.get(id) || new Set()
    return [...childIds].map(id => this.sessions.get(id)!)
  }

  getAncestors(id: string): Session.Info[] {
    const result: Session.Info[] = []
    let current = this.sessions.get(id)
    while (current?.parentID) {
      current = this.sessions.get(current.parentID)
      if (current) result.push(current)
    }
    return result
  }
}
```

---

### Challenge 4: Permission Handling

**Problem:** Server may pause execution for permission requests.

**Solution:**
```typescript
class PermissionHandler {
  private pending: Map<string, {
    resolve: (approved: boolean) => void
    permission: Permission
  }> = new Map()

  async handle(event: PermissionEvent) {
    const permission = event.properties.permission

    // Show UI dialog
    const approved = await this.showDialog(permission)

    // Send response
    await fetch(`/session/${permission.sessionID}/permissions/${permission.id}`, {
      method: "POST",
      body: JSON.stringify({ approved }),
    })
  }

  private async showDialog(permission: Permission): Promise<boolean> {
    // Implementation depends on UI framework
  }
}
```

---

## Estimated Development Effort

### TypeScript Web Client

| Component | Effort | Priority |
|-----------|--------|----------|
| HTTP client wrapper | 2-3 days | P0 |
| SSE event handling | 1-2 days | P0 |
| Session state management | 2-3 days | P0 |
| Message rendering | 3-5 days | P0 |
| Subagent monitoring | 2-3 days | P1 |
| Permission dialogs | 1-2 days | P1 |
| File diff viewer | 3-5 days | P1 |
| Cost dashboard | 1-2 days | P2 |
| Session sharing | 1 day | P2 |

**Total: 2-4 weeks** for full-featured client

---

### Python SDK

| Component | Effort | Priority |
|-----------|--------|----------|
| HTTP client | 2-3 days | P0 |
| Async streaming | 2-3 days | P0 |
| Type definitions | 1-2 days | P0 |
| Event handling | 1-2 days | P0 |
| Documentation | 2-3 days | P1 |

**Total: 1-2 weeks** for SDK

---

## Recommendations

### For Web Client

1. Use React/Vue/Svelte with reactive state
2. Implement SSE event batching for performance
3. Use virtual scrolling for message lists
4. Consider Monaco editor for code blocks

### For CLI Client

1. Use Ink (React for CLI) or Bubble Tea (Go)
2. Implement local caching
3. Support pipe/redirect for automation
4. Consider TUI framework like Ratatui (Rust)

### For IDE Plugin

1. Use direct module import for performance
2. Integrate with IDE's existing event loop
3. Leverage IDE's UI components
4. Support multiple concurrent sessions

---

## Conclusion

Building a new OpenCode client is highly feasible due to:

1. **Complete HTTP API** - All operations exposed via REST
2. **Real-time Events** - SSE provides live updates
3. **Well-Defined Schemas** - Zod schemas can generate types
4. **Clear Architecture** - Parent-child session model is straightforward
5. **Flexible Permission System** - Async permission handling

**Recommended starting point:**
1. Implement basic session/message CRUD
2. Add SSE event streaming
3. Build subagent monitoring
4. Add permission handling
5. Enhance with file diffs, costs, sharing

The modular API design ensures any client can achieve feature parity with the existing TUI while potentially adding new capabilities like web UIs, mobile apps, or IDE integrations.
