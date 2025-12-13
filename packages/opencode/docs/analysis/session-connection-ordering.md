# Session Multiple Connections and Message Ordering

This document provides a comprehensive analysis of how the OpenCode server handles multiple client connections for the same session and guarantees message ordering.

## Table of Contents

1. [Multiple Client Support](#1-multiple-client-support)
2. [Message Ordering Guarantee](#2-message-ordering-guarantee)
3. [Concurrent Request Handling](#3-concurrent-request-handling)
4. [File-Level Concurrency Control](#4-file-level-concurrency-control)
5. [Event Broadcasting](#5-event-broadcasting)
6. [Race Condition Prevention](#6-race-condition-prevention)
7. [Key Files Summary](#7-key-files-summary)

---

## 1. Multiple Client Support

### Does OpenCode Allow Multiple Connections for the Same Session?

**YES** - The OpenCode server uses a Bus-based pub/sub event system that allows multiple clients to connect to the same session and receive updates.

### Event Stream Endpoints

**File**: `packages/opencode/src/server/server.ts` (lines 1957-1995)

```typescript
GET /event         - Session-scoped Server-Sent Events (SSE) stream
GET /global/event  - Global event stream
```

### Connection Implementation

**Code Location**: `packages/opencode/src/server/server.ts:1973-1995`

```typescript
async (c) => {
  log.info("event connected")
  return streamSSE(c, async (stream) => {
    stream.writeSSE({
      data: JSON.stringify({
        type: "server.connected",
        properties: {},
      }),
    })
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
}
```

Each client maintains a separate HTTP connection with an SSE stream. Multiple connections are supported because events are published to all subscribers.

### Connection Tracking

**File**: `packages/opencode/src/bus/index.ts` (lines 1-119)

- Each connection uses the Bus subscription system to track active listeners
- Subscriptions are stored in a `Map<any, Subscription[]>()` structure (line 12)
- When a client connects via SSE, it registers a callback handler
- When a client disconnects, the `stream.onAbort()` callback unsubscribes the handler

### Global Event Broadcasting

**File**: `packages/opencode/src/bus/global.ts` (lines 1-10)

```typescript
export const GlobalBus = new EventEmitter<{
  event: [{
    directory: string
    payload: any
  }]
}>()
```

- Uses Node.js EventEmitter for cross-directory event propagation
- Each event published to Bus is also emitted to GlobalBus for multi-client notification

---

## 2. Message Ordering Guarantee

### How Are Messages Processed?

Messages are processed **sequentially per session**, using a sophisticated queueing mechanism that ensures only one message is processed at a time for any given session.

### Session Prompt State Lock

**File**: `packages/opencode/src/session/prompt.ts` (lines 55-238)

```typescript
const state = Instance.state(
  () => {
    const data: Record<string, {
      abort: AbortController
      callbacks: {
        resolve(input: MessageV2.WithParts): void
        reject(): void
      }[]
    }> = {}
    return data
  }
)

function start(sessionID: string) {
  const s = state()
  if (s[sessionID]) return  // Session already busy - return undefined
  const controller = new AbortController()
  s[sessionID] = {
    abort: controller,
    callbacks: [],
  }
  return controller.signal
}

export const loop = fn(Identifier.schema("session"), async (sessionID) => {
  const abort = start(sessionID)
  if (!abort) {
    // Session is busy - queue this request
    return new Promise<MessageV2.WithParts>((resolve, reject) => {
      const callbacks = state()[sessionID].callbacks
      callbacks.push({ resolve, reject })
    })
  }
  // Process message...
})
```

### How the Queue Works

1. **First client** to call `prompt()` acquires the lock (sets `state()[sessionID]`)
2. **Subsequent clients** push their resolve/reject callbacks into the queue (line 237)
3. When the first message completes, **all queued callbacks are resolved** in order (line 607)

### Callback Resolution

**File**: `packages/opencode/src/session/prompt.ts` (lines 603-610)

```typescript
SessionCompaction.prune({ sessionID })
for await (const item of MessageV2.stream(sessionID)) {
  if (item.info.role === "user") continue
  const queued = state()[sessionID]?.callbacks ?? []
  for (const q of queued) {
    q.resolve(item)  // Resolve all queued callbacks with the result
  }
  return item
}
```

---

## 3. Concurrent Request Handling

### What Happens When Two Clients Send Messages?

#### Scenario A: Client B sends while Client A's message is processing

1. Client A's message starts processing (acquires lock)
2. Client B's request arrives and calls `start(sessionID)`
3. `start()` returns `undefined` because session is busy
4. Client B's promise is queued in `callbacks[]`
5. When Client A's message completes, Client B receives the result
6. Client B's message is then processed next

#### Scenario B: Concurrent sends arrive at exact same time

- Only **one** client acquires the lock (first to call `start()`)
- Others are queued and resolved in order
- No race condition due to JavaScript's single-threaded event loop

### BusyError Prevention

**File**: `packages/opencode/src/session/prompt.ts` (lines 80-83)

```typescript
export function assertNotBusy(sessionID: string) {
  const match = state()[sessionID]
  if (match) throw new Session.BusyError(sessionID)
}
```

**File**: `packages/opencode/src/session/index.ts` (lines 443-446)

```typescript
export class BusyError extends Error {
  constructor(public readonly sessionID: string) {
    super(`Session ${sessionID} is busy`)
  }
}
```

This check is used by operations like `SessionRevert.revert()` and `SessionRevert.unrevert()` to prevent concurrent modifications during processing.

### Session Status Tracking

**File**: `packages/opencode/src/session/status.ts` (lines 43-46)

```typescript
const state = Instance.state(() => {
  const data: Record<string, Info> = {}
  return data
})
```

States: `"idle"`, `"retry"`, `"busy"`

---

## 4. File-Level Concurrency Control

### Reader-Writer Lock Pattern

**File**: `packages/opencode/src/util/lock.ts` (lines 1-98)

The server implements a classic read-write lock with writer starvation prevention:

```typescript
export namespace Lock {
  const locks = new Map<
    string,
    {
      readers: number
      writer: boolean
      waitingReaders: (() => void)[]
      waitingWriters: (() => void)[]
    }
  >()

  function process(key: string) {
    const lock = locks.get(key)
    if (!lock || lock.writer || lock.readers > 0) return

    // Prioritize writers to prevent starvation
    if (lock.waitingWriters.length > 0) {
      const nextWriter = lock.waitingWriters.shift()!
      nextWriter()
      return
    }

    // Wake up all waiting readers
    while (lock.waitingReaders.length > 0) {
      const nextReader = lock.waitingReaders.shift()!
      nextReader()
    }
  }

  export async function read(key: string): Promise<Disposable> {
    // Multiple concurrent readers allowed
    // ...
  }

  export async function write(key: string): Promise<Disposable> {
    // Exclusive write access
    // ...
  }
}
```

### Lock Characteristics

- **Line 28-32**: Writers have priority over readers (prevents starvation)
- **Line 51-52**: Multiple readers can hold lock simultaneously
- **Line 77-78**: Only one writer can hold lock exclusively
- Uses Promise-based async/await locking with disposal pattern (`Symbol.dispose`)

### Storage Lock Usage

**File**: `packages/opencode/src/storage/storage.ts` (lines 168-196)

```typescript
export async function read<T>(key: string[]) {
  const dir = await state().then((x) => x.dir)
  const target = path.join(dir, ...key) + ".json"
  return withErrorHandling(async () => {
    using _ = await Lock.read(target)   // Read lock
    const result = await Bun.file(target).json()
    return result as T
  })
}

export async function update<T>(key: string[], fn: (draft: T) => void) {
  const dir = await state().then((x) => x.dir)
  const target = path.join(dir, ...key) + ".json"
  return withErrorHandling(async () => {
    using _ = await Lock.write(target)  // Write lock
    const content = await Bun.file(target).json()
    fn(content)
    await Bun.write(target, JSON.stringify(content, null, 2))
    return content as T
  })
}

export async function write<T>(key: string[], content: T) {
  const dir = await state().then((x) => x.dir)
  const target = path.join(dir, ...key) + ".json"
  return withErrorHandling(async () => {
    using _ = await Lock.write(target)  // Write lock
    await Bun.write(target, JSON.stringify(content, null, 2))
  })
}
```

### Concurrency Control Strategy

- **Reads**: Multiple concurrent reads on the same file (lock-free for reads)
- **Updates**: Exclusive write lock (read-modify-write transaction)
- **Writes**: Exclusive write lock (atomic writes)

---

## 5. Event Broadcasting

### Multi-Client Event Distribution

**File**: `packages/opencode/src/bus/index.ts` (lines 55-78)

```typescript
export async function publish<Definition extends EventDefinition>(
  def: Definition,
  properties: z.output<Definition["properties"]>
) {
  const payload = {
    type: def.type,
    properties,
  }
  log.info("publishing", { type: def.type })

  const pending = []
  for (const key of [def.type, "*"]) {
    const match = state().subscriptions.get(key)
    for (const sub of match ?? []) {
      pending.push(sub(payload))      // Call all subscribers
    }
  }

  GlobalBus.emit("event", {           // Broadcast globally
    directory: Instance.directory,
    payload,
  })

  return Promise.all(pending)          // Wait for all handlers
}
```

### Event Types Published

**File**: `packages/opencode/src/session/index.ts` (lines 87-120)

```typescript
export const Event = {
  Created: Bus.event("session.created", z.object({ info: Info })),
  Updated: Bus.event("session.updated", z.object({ info: Info })),
  Deleted: Bus.event("session.deleted", z.object({ info: Info })),
  Diff: Bus.event("session.diff", z.object({ sessionID, diff })),
  Error: Bus.event("session.error", z.object({ sessionID, error })),
}
```

### Message Update Events

**File**: `packages/opencode/src/session/index.ts` (lines 344-388)

```typescript
export const updateMessage = fn(MessageV2.Info, async (msg) => {
  await Storage.write(["message", msg.sessionID, msg.id], msg)
  Bus.publish(MessageV2.Event.Updated, {  // Broadcast message update
    info: msg,
  })
  return msg
})

export const updatePart = fn(UpdatePartInput, async (input) => {
  const part = "delta" in input ? input.part : input
  const delta = "delta" in input ? input.delta : undefined
  await Storage.write(["part", part.messageID, part.id], part)
  Bus.publish(MessageV2.Event.PartUpdated, {  // Broadcast part update
    part,
    delta,
  })
  return part
})
```

### Dual Response Mechanism

1. **Direct Response**: Message response streamed back on the same HTTP connection
2. **Event Broadcasting**: Message updates also published to all SSE subscribers on `/event` endpoint

### Message Streaming Endpoint

**File**: `packages/opencode/src/server/server.ts` (lines 942-980)

```typescript
.post("/session/:id/message", async (c) => {
  c.status(200)
  c.header("Content-Type", "application/json")
  return stream(c, async (stream) => {
    const sessionID = c.req.valid("param").id
    const body = c.req.valid("json")
    const msg = await SessionPrompt.prompt({ ...body, sessionID })
    stream.write(JSON.stringify(msg))
  })
})
```

---

## 6. Race Condition Prevention

### Key Mechanisms Summary

| Concern | Mechanism | File | Lines |
|---------|-----------|------|-------|
| **Session-level message conflicts** | `start()` function returns falsy if session busy; queues requests | prompt.ts | 207-238 |
| **File-level concurrent access** | Reader-Writer Lock with writer priority | lock.ts | 24-45 |
| **State disposal races** | Timeout-protected disposal with Promise.all | state.ts | 31-64 |
| **Event ordering** | Bus.publish waits for all subscribers (Promise.all) | index.ts | 77 |

### Cleanup on Session Completion

**File**: `packages/opencode/src/session/prompt.ts` (lines 218-230)

```typescript
export function cancel(sessionID: string) {
  log.info("cancel", { sessionID })
  const s = state()
  const match = s[sessionID]
  if (!match) return
  match.abort.abort()                   // Abort ongoing processing
  for (const item of match.callbacks) {
    item.reject()                       // Reject queued requests
  }
  delete s[sessionID]                   // Remove session from state
  SessionStatus.set(sessionID, { type: "idle" })
  return
}
```

### Async Queue Utility

**File**: `packages/opencode/src/util/queue.ts` (lines 1-19)

```typescript
export class AsyncQueue<T> implements AsyncIterable<T> {
  private queue: T[] = []
  private resolvers: ((value: T) => void)[] = []

  push(item: T) {
    const resolve = this.resolvers.shift()
    if (resolve) resolve(item)
    else this.queue.push(item)
  }

  async next(): Promise<T> {
    if (this.queue.length > 0) return this.queue.shift()!
    return new Promise((resolve) => this.resolvers.push(resolve))
  }

  async *[Symbol.asyncIterator]() {
    while (true) yield await this.next()
  }
}
```

This enables async iteration patterns where consumers can wait for items that haven't been pushed yet.

---

## 7. Key Files Summary

| File | Role |
|------|------|
| `src/session/prompt.ts:207-238` | Session lock and callback queue for message ordering |
| `src/util/lock.ts:1-98` | Reader-writer lock implementation for file access |
| `src/bus/index.ts:55-78` | Event broadcasting to multiple clients |
| `src/server/server.ts:1973-1995` | SSE event streaming endpoints |
| `src/storage/storage.ts:168-196` | Locked file operations |
| `src/session/status.ts:43-46` | Session status tracking |
| `src/util/queue.ts:1-19` | Async queue for event processing |

---

## Summary Table: Concurrency Control

| Layer | Mechanism | Scope | Guarantees |
|-------|-----------|-------|-----------|
| **Session Message** | Single-threaded loop with callback queue | Per session | Sequential processing, queued requests |
| **File I/O** | Reader-Writer Lock | Per file | Concurrent reads, exclusive writes |
| **Event Publishing** | Bus pub/sub + GlobalBus EventEmitter | Global | All subscribers notified atomically |
| **State Storage** | Directory-scoped Instance.state | Per project | Singleton per init function |
| **HTTP Connections** | SSE streams with individual subscriptions | Per connection | Independent event delivery |

---

## Conclusion

The OpenCode server supports **multiple concurrent client connections** to the same session through:

- **Isolated SSE streams** for each client connection
- **Bus-based pub/sub** for event broadcasting
- **Sequential message processing** per session using callback queues
- **File-level locking** with reader-writer semantics
- **Atomic storage operations** with automatic timestamps and event publishing

This design ensures **message ordering is preserved per session** while allowing **concurrent message processing across different sessions** and **concurrent client connections** to receive real-time updates.

When two clients send messages to the same session:
1. One gets processed immediately (acquires the lock)
2. The other waits in the queue
3. Both receive the result when processing completes
4. The queued message then processes next
