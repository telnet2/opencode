# Event Historical Replay Analysis

This document analyzes whether OpenCode performs historical replay of events when a new client connects to a session.

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [SSE Event Endpoint Behavior](#2-sse-event-endpoint-behavior)
3. [Bus Subscription Mechanism](#3-bus-subscription-mechanism)
4. [How Clients Get Historical Data](#4-how-clients-get-historical-data)
5. [TUI Client Implementation](#5-tui-client-implementation)
6. [Architecture Pattern](#6-architecture-pattern)

---

## 1. Executive Summary

**OpenCode does NOT perform historical replay of events when a new client connects.**

Instead, it uses a **pull-based model** for historical data and **push-based model** for real-time updates:

| Data Type | Retrieval Method |
|-----------|------------------|
| Historical messages | `GET /session/:id/message` (pull) |
| Session state | `GET /session/:id` (pull) |
| Past diffs | `GET /session/:id/diff` (pull) |
| Future updates | SSE `/event` stream (push) |

---

## 2. SSE Event Endpoint Behavior

### What Happens on Connect

**File**: `packages/opencode/src/server/server.ts` (lines 1957-1996)

When a client connects to the `/event` endpoint, only a connection acknowledgment is sent:

```typescript
.get("/event", /* ... */, async (c) => {
  log.info("event connected")
  return streamSSE(c, async (stream) => {
    // Send only a connection acknowledgment - NO historical events
    stream.writeSSE({
      data: JSON.stringify({
        type: "server.connected",
        properties: {},
      }),
    })
    // Subscribe ONLY to future events
    const unsub = Bus.subscribeAll(async (event) => {
      await stream.writeSSE({
        data: JSON.stringify(event),
      })
    })
    await new Promise<void>((resolve) => {
      stream.onAbort(() => {
        unsub()
        resolve()
        log.info("event disconnected")
      })
    })
  })
})
```

**Key Finding**: Only a `server.connected` event is sent. No message history or session state is replayed.

### Global Event Endpoint

**File**: `packages/opencode/src/server/server.ts` (lines 127-170)

The `/global/event` endpoint uses `GlobalBus` with identical behavior - no historical replay:

```typescript
.get("/global/event", /* ... */, async (c) => {
  log.info("global event connected")
  return streamSSE(c, async (stream) => {
    GlobalBus.on("event", handler)
    // No historical events sent
    await new Promise<void>((resolve) => {
      stream.onAbort(() => {
        GlobalBus.off("event", handler)
        resolve()
      })
    })
  })
})
```

---

## 3. Bus Subscription Mechanism

### No Event History Storage

**File**: `packages/opencode/src/bus/index.ts`

The Bus implementation stores subscriptions in memory with NO event history:

```typescript
const state = Instance.state(() => {
  const subscriptions = new Map<any, Subscription[]>()
  return { subscriptions }  // No event history!
})

export function subscribeAll(callback: (event: any) => void) {
  return raw("*", callback)
}

function raw(type: string, callback: (event: any) => void) {
  const subscriptions = state().subscriptions
  let match = subscriptions.get(type) ?? []
  match.push(callback)
  subscriptions.set(type, match)
  // ... returns unsubscribe function
}
```

**Key Characteristics**:
- No event history maintained
- `Bus.subscribeAll()` only calls subscribers with NEW events going forward
- Events are NOT stored or cached
- Pure pub/sub pattern with zero replay capability

---

## 4. How Clients Get Historical Data

Clients retrieve historical data through explicit REST API calls, not through event streams.

### Message History API

**File**: `packages/opencode/src/server/server.ts` (lines 838-875)

```typescript
.get("/session/:id/message", /* ... */, async (c) => {
  const query = c.req.valid("query")
  const messages = await Session.messages({
    sessionID: c.req.valid("param").id,
    limit: query.limit,  // Supports pagination
  })
  return c.json(messages)
})
```

### Message Retrieval Implementation

**File**: `packages/opencode/src/session/index.ts` (line 287)

```typescript
export const messages = fn(
  z.object({
    sessionID: Identifier.schema("session"),
    limit: z.number().optional(),
  }),
  async (input) => {
    const result = [] as MessageV2.WithParts[]
    for await (const msg of MessageV2.stream(input.sessionID)) {
      if (input.limit && result.length >= input.limit) break
      result.push(msg)
    }
    result.reverse()
    return result
  },
)
```

**File**: `packages/opencode/src/session/message-v2.ts` (line 670)

```typescript
export const stream = fn(Identifier.schema("session"), async function* (sessionID) {
  const list = await Array.fromAsync(await Storage.list(["message", sessionID]))
  for (let i = list.length - 1; i >= 0; i--) {
    yield await get({
      sessionID,
      messageID: list[i][2],
    })
  }
})
```

Messages are fetched from persistent storage (file system), not from event streams.

---

## 5. TUI Client Implementation

### Explicit State Synchronization

**File**: `packages/opencode/src/cli/cmd/tui/context/sync.tsx`

The TUI client explicitly syncs session state on demand:

```typescript
session: {
  async sync(sessionID: string) {
    if (store.message[sessionID]) return  // Cache check

    // Fetch session data via explicit API calls
    const [session, messages, todo, diff] = await Promise.all([
      sdk.client.session.get({ path: { id: sessionID } }),
      sdk.client.session.messages({ path: { id: sessionID }, query: { limit: 100 } }),
      sdk.client.session.todo({ path: { id: sessionID } }),
      sdk.client.session.diff({ path: { id: sessionID } }),
    ])

    // Store in local state
    setStore(produce((draft) => {
      draft.message[sessionID] = messages.data!.map((x) => x.info)
      for (const message of messages.data!) {
        draft.part[message.info.id] = message.parts
      }
      // ...
    }))
  },
}

// Only future events are listened to via event stream
sdk.event.listen((e) => {
  const event = e.details
  // Handle message.updated, session.updated, etc. (NEW events only)
})
```

**Key Points**:
- Session history is NOT replayed via SSE
- Client explicitly calls `/session/:id/message` API
- Messages are fetched with optional limit (pagination support)
- Event stream is used ONLY for incremental updates

---

## 6. Architecture Pattern

### Event Types Published

**File**: `packages/opencode/src/session/index.ts` (lines 87-120)

Session publishes these events for future subscribers only:

```typescript
export const Event = {
  Created: Bus.event("session.created", z.object({ info: Info })),
  Updated: Bus.event("session.updated", z.object({ info: Info })),
  Deleted: Bus.event("session.deleted", z.object({ info: Info })),
  Diff: Bus.event("session.diff", z.object({ ... })),
  Error: Bus.event("session.error", z.object({ ... })),
}
```

**File**: `packages/opencode/src/session/message-v2.ts` (lines 373-399)

Message events:

```typescript
export const Event = {
  Updated: Bus.event("message.updated", z.object({ info: Info })),
  Removed: Bus.event("message.removed", z.object({ ... })),
  PartUpdated: Bus.event("message.part.updated", z.object({ ... })),
  PartRemoved: Bus.event("message.part.removed", z.object({ ... })),
}
```

Events are only for NEW state changes, not for delivering historical state.

### Design Rationale

This architecture provides:

1. **Scalability**: No need to store event history in memory
2. **Simplicity**: Clear separation between historical data and real-time updates
3. **Flexibility**: Clients can fetch exactly what they need via REST
4. **Efficiency**: SSE connections remain lightweight

---

## Summary

| Aspect | Finding |
|--------|---------|
| **Event History Storage** | None - events are not stored |
| **Historical Event Replay** | Not implemented |
| **SSE On Connect** | Sends only `server.connected` acknowledgment |
| **Future Events** | Streamed via `/event` or `/global/event` endpoints |
| **Session History Retrieval** | Explicit REST API calls (`GET /session/:id/message`) |
| **Message Pagination** | Supported via `limit` query parameter |
| **Client State Sync** | Lazy-loaded on demand via `session.sync()` |
| **Storage Backend** | File system based (via Storage abstraction) |

**Conclusion**: OpenCode uses a clean separation of concerns where the event bus handles real-time notifications while REST endpoints handle data retrieval. New clients must explicitly fetch past data through API calls.
