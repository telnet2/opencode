# Code Complexity Comparison: kiana vs kiana-v6

## Summary

| Metric | kiana | kiana-v6 |
|--------|-------|----------|
| **Total LOC** | ~7,060 | ~7,267 |
| **Core File** | session.ts (1,119 LOC) | agent.ts (922 LOC) |
| **Architecture** | Functional (factory) + EventBus | Class-based (OOP) + StreamPart |
| **AI SDK** | v6 (streamText/generateText) | v6 (ToolLoopAgent) |
| **Event Protocol** | Custom EventBus (13 event types) | AI SDK UI StreamPart (19 types) |

---

## Code Readability Scores (1-10 scale)

### 1. Naming & Semantics
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **8/10** | Clear names: `createSession`, `sendMessage`, `runAgentLoop`. State flows are explicit. |
| kiana-v6 | **9/10** | Class methods are self-documenting: `generate()`, `stream()`, `buildAgent()`. OOP encapsulation improves discoverability. |

### 2. Code Organization
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **7/10** | Session logic in one 1,119-line file. The `runAgentLoop` function handles streaming inline with complex switch-case blocks (~200 lines for stream chunk handling). |
| kiana-v6 | **8/10** | `CodingAgent` class (919 lines) is more modular. Separate `buildAgent()`, `buildAITools()`, and `buildAIMessages()` methods improve separation of concerns. |

### 3. Abstraction Level
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **7/10** | Manual streaming loop with fine-grained chunk handling. More verbose but explicit control over each event type. |
| kiana-v6 | **9/10** | `ToolLoopAgent` abstracts away the agent loop. Callbacks (`onStepFinish`, `onFinish`) are declarative. Less boilerplate. |

### 4. Type Safety
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **9/10** | Comprehensive Zod schemas with discriminated unions for parts, events. |
| kiana-v6 | **9/10** | Identical type definitions (shared patterns). |

### 5. Tool System
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **9/10** | Clean `defineTool` pattern with validation wrapper. |
| kiana-v6 | **9/10** | Identical implementation. |

### 6. Error Handling
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **8/10** | Try-catch at tool/stream level. Defensive JSON parsing for malformed inputs. |
| kiana-v6 | **8/10** | Same patterns. Slightly cleaner with class encapsulation. |

### 7. Streaming Implementation
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **6/10** | 200+ lines of switch-case handling for stream chunks. Uses EventBus with nested event structure `{ type, properties: { part, delta } }`. |
| kiana-v6 | **9/10** | Uses AI SDK UI-compatible StreamPart protocol. Flat structure `{ type, id, delta }`. SSE-ready with `formatSSE`/`parseSSE` utilities. |

### 8. Cognitive Load
| Package | Score | Notes |
|---------|-------|-------|
| kiana | **6/10** | Functional style with closures requires tracking state through closure scope. `runAgentLoop` while-loop is complex. |
| kiana-v6 | **8/10** | Class properties (`this.messages`, `this.config`) are explicit. State is encapsulated, easier to follow. |

---

## Overall Readability Scores

| Package | Score | Summary |
|---------|-------|---------|
| **kiana** | **7.4/10** | Explicit, manual control over streaming/events. Custom EventBus with nested event structure. Good for understanding internals but verbose. |
| **kiana-v6** | **8.7/10** | AI SDK UI-compatible StreamPart protocol. Class-based with flat event structure. SSE-ready for web integration. Trade-off: more event types to learn. |

---

## Key Architectural Differences

| Aspect | kiana | kiana-v6 |
|--------|-------|----------|
| **Session API** | `createSession()` → returns object with methods | `new CodingAgent()` → class instance |
| **Event System** | `EventBus` with `onEvent()` callback | `StreamPart` with `onStream()` callback |
| **Event Structure** | Nested: `{ type, properties: { part, delta } }` | Flat: `{ type, id, delta }` |
| **Event Count** | 13 event types (session.*, message.*, todo.*) | 19 stream part types (text-*, tool-*, data-*) |
| **Wire Format** | Custom JSON | AI SDK UI SSE protocol |
| **Web Integration** | Needs adapter | SSE-ready (`formatSSE`, `createSSEHeaders`) |
| **Message Loop** | Manual `while(true)` loop in `runAgentLoop()` | `ToolLoopAgent` handles internally |
| **Config** | External `Config` type loaded from file | `CodingAgentConfig` interface passed to constructor |

---

## Code Structure Comparison

### kiana (Functional Factory + EventBus)
```
kiana/src/
├── session.ts        (1,119 LOC) - Core agent loop, manual streaming
├── cli.ts            (409 LOC)   - CLI interface
├── config.ts         (121 LOC)   - Config loading
├── event.ts          (111 LOC)   - EventBus class
├── provider.ts       (59 LOC)    - LLM provider factory
├── types/
│   ├── event.ts      (184 LOC)   - 13 event type schemas (Zod)
│   ├── part.ts       (230 LOC)   - Message part schemas
│   └── ...
├── tool/             (2,100 LOC) - 14 tool implementations
└── util/             (220 LOC)   - Ripgrep wrapper
```

### kiana-v6 (Class-Based + StreamPart Protocol)
```
kiana-v6/src/
├── agent.ts          (922 LOC)   - CodingAgent class with ToolLoopAgent
├── stream.ts         (288 LOC)   - AI SDK UI StreamPart types + SSE utilities
├── cli.ts            (443 LOC)   - CLI interface
├── config.ts         (120 LOC)   - Config loading
├── provider.ts       (58 LOC)    - LLM provider factory
├── types/
│   ├── part.ts       (230 LOC)   - Message part schemas
│   └── ...           (no event.ts - replaced by stream.ts)
├── tool/             (2,702 LOC) - 14 tool implementations
└── util/             (220 LOC)   - Ripgrep wrapper
```

---

## Shared Components (Identical)

Both packages share identical implementations for:

1. **Type System** (`types/part.ts`, `types/message.ts`, `types/event.ts`, `types/session.ts`)
   - Same Zod schemas with discriminated unions
   - 12 part types: Text, Reasoning, Tool, File, StepStart, StepFinish, Snapshot, Patch, Agent, Retry, Compaction, Subtask

2. **Tool Definition** (`tool/tool.ts`)
   - Same `defineTool()` wrapper with validation
   - Same `ToolContext` and `ToolResult` interfaces
   - Defensive JSON parsing for malformed inputs

3. **Event Bus** (`event.ts`)
   - Same typed pub/sub implementation
   - Same event type discriminated union

4. **Most Tool Implementations**
   - bash, read, write, glob, grep, list, webfetch, websearch, codesearch, todo, task, invalid

---

## Event/Stream Protocol Deep Dive

### kiana - EventBus with Nested Events
```typescript
// Event structure (13 types)
interface Event {
  type: "message.part.updated"
  properties: {
    part: TextPart | ToolPart | ...
    delta?: string
  }
  context?: SubagentContext  // For subagent events
}

// Usage
eventBus.emit({
  type: "message.part.updated",
  properties: { part: textPart, delta: chunk.text }
})

// Subscription
session.onEvent((event) => {
  if (event.type === "message.part.updated") {
    // Handle nested properties
    console.log(event.properties.part)
  }
})
```

### kiana-v6 - AI SDK UI StreamPart Protocol
```typescript
// StreamPart structure (19 types, SSE-compatible)
type StreamPart =
  | { type: "text-start"; id: string }
  | { type: "text-delta"; id: string; delta: string }
  | { type: "text-end"; id: string }
  | { type: "tool-input-available"; toolCallId: string; toolName: string; input: object }
  | { type: "tool-output-available"; toolCallId: string; output: unknown }
  | { type: "data-session"; data: SessionInfo }
  | ...

// Usage - flat structure
this.emit({ type: "text-delta", id: textPartId, delta: chunk })

// Subscription
agent.onStream((part) => {
  if (part.type === "text-delta") {
    // Direct access to fields
    console.log(part.delta)
  }
})

// SSE utilities included
formatSSE(part)  // → "data: {"type":"text-delta",...}\n\n"
parseSSE(line)   // → StreamPart | null
createSSEHeaders() // → { "Content-Type": "text/event-stream", ... }
```

---

## Recommendations

### Use kiana when:
- Building custom TUI/desktop applications with own event handling
- Need Zod-validated event schemas for type safety
- Want nested event structure with rich Part objects
- Prefer functional factory pattern

### Use kiana-v6 when:
- Building web applications with AI SDK UI components (`useChat`)
- Need SSE streaming out of the box
- Want flat, simple event structure
- Prefer class-based OOP pattern
- Need direct compatibility with Vercel AI SDK ecosystem

### Protocol Comparison

| Aspect | kiana (EventBus) | kiana-v6 (StreamPart) |
|--------|------------------|----------------------|
| **Event structure** | Nested `{ type, properties }` | Flat `{ type, ...fields }` |
| **Type safety** | Zod schemas | TypeScript interfaces |
| **SSE support** | Manual | Built-in (`formatSSE`, `parseSSE`) |
| **Web ready** | Needs adapter | Direct `useChat` compatibility |
| **Event granularity** | Coarse (13 types) | Fine (19 types) |
| **Subagent context** | In event `context` field | Via `data-subagent-context` part |

### For Extension:
Both packages share identical tool/type systems, making them interchangeable at the tool layer. Tools written for one package work in the other without modification.
