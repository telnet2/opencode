# Multi-Server Statefulness Analysis

This document provides a comprehensive analysis of whether OpenCode servers are stateless and whether sessions can be handled by multiple server instances.

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [State Storage Locations](#2-state-storage-locations)
3. [In-Memory State Analysis](#3-in-memory-state-analysis)
4. [File-Based Storage Analysis](#4-file-based-storage-analysis)
5. [Locking Across Processes](#5-locking-across-processes)
6. [Session Affinity Requirements](#6-session-affinity-requirements)
7. [Event Broadcasting Limitations](#7-event-broadcasting-limitations)
8. [What Breaks with Multiple Servers](#8-what-breaks-with-multiple-servers)
9. [Race Condition Scenarios](#9-race-condition-scenarios)
10. [Production Deployment Requirements](#10-production-deployment-requirements)

---

## 1. Executive Summary

**OpenCode servers are HIGHLY STATEFUL and are NOT designed for multi-server load balancing.**

| Question | Answer |
|----------|--------|
| **Are servers stateless?** | **No** - significant in-memory state |
| **Can any server handle any session?** | **No** - must use session affinity |
| **What if different server handles next message?** | Data corruption, lost callbacks, broken cancellation |

**Recommendation**: Use a single server per working directory, or implement sticky sessions if load balancing is required.

---

## 2. State Storage Locations

### File-Based Storage (Stateless/Shareable)

**Location**: XDG base directories (`~/.local/share/opencode/storage/`)

**Structure**: Hierarchical JSON files:
- `storage/session/{projectID}/{sessionID}.json` - Session metadata
- `storage/message/{sessionID}/{messageID}.json` - Individual messages
- `storage/part/{messageID}/{partID}.json` - Message parts
- `storage/session_diff/{sessionID}.json` - Diff data

**Shared by multiple servers**: YES (files are shared via filesystem)

### In-Memory State (Stateful/Process-Bound)

**Location**: Process memory only

**Shared between server instances**: **NO**

---

## 3. In-Memory State Analysis

### A. Session Prompt State Lock (CRITICAL)

**File**: `packages/opencode/src/session/prompt.ts` (lines 59-78)

```typescript
const state = Instance.state(
  () => {
    const data: Record<
      string,
      {
        abort: AbortController
        callbacks: {
          resolve(input: MessageV2.WithParts): void
          reject(): void
        }[]
      }
    > = {}
    return data
  },
  // cleanup on dispose
)
```

**Problem**: This state is entirely **process-local**. It contains:
- `AbortController` instances for each active session
- Callback queues for multiple concurrent requests to the same session
- Session busy/idle state tracking

**Multi-Server Impact**: If two servers try to handle the same session:
- Server A's abort signal won't affect Server B's processing
- Server B cannot see Server A's abort state
- Both servers will try to process the same session independently

### B. Locking Mechanism (PROCESS-LOCAL)

**File**: `packages/opencode/src/util/lock.ts`

```typescript
const locks = new Map<
  string,
  {
    readers: number
    writer: boolean
    waitingReaders: (() => void)[]
    waitingWriters: (() => void)[]
  }
>()
```

**Facts**:
- This is an **in-memory reader/writer lock**
- Keys are file paths
- Stored in a `Map` in process memory
- Used in `Storage.read()` and `Storage.write()` operations
- **NO file-level OS locks** (no `flock`, `fcntl`, or similar)

**Multi-Server Impact**:
- Server 1 acquires write lock on `session/{id}.json`
- Server 2 can acquire its own write lock on the same file (different lock object!)
- Both servers will write to the same file simultaneously
- **File corruption or lost writes possible**

### C. Session Busy State (PROCESS-LOCAL)

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

**How it's used**:
- `SessionRevert.revert()` calls `assertNotBusy()` before reverting
- `SessionRevert.unrevert()` calls `assertNotBusy()` before unreverting
- Only checked at operation start, not continuously

**Multi-Server Impact**:
- Only Server A knows session is busy (in its local state)
- Server B has no knowledge and will attempt the operation
- No prevention of concurrent modifications

### D. Session Status State (PROCESS-LOCAL)

**File**: `packages/opencode/src/session/status.ts` (lines 43-46)

```typescript
const state = Instance.state(() => {
  const data: Record<string, Info> = {}
  return data
})
```

States: `"idle"`, `"retry"`, `"busy"`

**Multi-Server Impact**:
- Server A has session status "busy"
- Server B sees status "idle" (different state maps)
- No coordination between servers

### E. Bus Subscriptions (PROCESS-LOCAL)

**File**: `packages/opencode/src/bus/index.ts` (lines 11-17)

```typescript
const state = Instance.state(() => {
  const subscriptions = new Map<any, Subscription[]>()
  return {
    subscriptions,
  }
})
```

Events are published to:
1. Local subscriptions (in-process only)
2. `GlobalBus` (EventEmitter in process memory)

**Multi-Server Impact**:
- Server A publishes event: `Bus.publish(Event.Created, { info: result })`
- Server B won't receive it (different subscription maps)
- Only clients connected to Server B can see events from Server B
- Clients connected to different servers see different event streams

---

## 4. File-Based Storage Analysis

**Storage Implementation**: `packages/opencode/src/storage/storage.ts`

```typescript
export async function read<T>(key: string[]) {
  const dir = await state().then((x) => x.dir)
  const target = path.join(dir, ...key) + ".json"
  return withErrorHandling(async () => {
    using _ = await Lock.read(target)  // In-memory lock!
    const result = await Bun.file(target).json()
    return result as T
  })
}

export async function update<T>(key: string[], fn: (draft: T) => void) {
  const dir = await state().then((x) => x.dir)
  const target = path.join(dir, ...key) + ".json"
  return withErrorHandling(async () => {
    using _ = await Lock.write(target)  // In-memory lock!
    const content = await Bun.file(target).json()
    fn(content)
    await Bun.write(target, JSON.stringify(content, null, 2))
    return content as T
  })
}
```

**Critical Issue**: The `Lock` mechanism protects within-process access but provides **zero protection** against concurrent writes from different server processes.

---

## 5. Locking Across Processes

### Distributed Locking Capability: NONE

The lock mechanism:
- Uses an in-memory `Map` as the lock store
- Each process has its own `Map` instance
- No OS-level file locks
- No persistent lock files
- No distributed lock service integration

### What Happens with Concurrent Multi-Server Writes

```
Server A                           Server B
──────────────────────────────────────────────
Lock.write("session/123.json")     Lock.write("session/123.json")
  ↓                                  ↓
locks.set(path, {writer: true})    locks.set(path, {writer: true})
  ↓                                  ↓
Read file                            Read file
  ↓                                  ↓
Modify content A                     Modify content B
  ↓                                  ↓
Write file                           Write file (overwrites A!)
  ↓                                  ↓
locks.delete(path)                   locks.delete(path)

Result: Server B's write overwrites Server A's changes!
```

---

## 6. Session Affinity Requirements

### Can a Different Server Pick Up a Session Mid-Conversation?

**NO - It would break in multiple ways**

### 1. Abort Signals Don't Work

The `AbortController` from Server A won't be available on Server B:

```typescript
// Server A started processing
const abort = start(sessionID)  // Creates abort in Server A's memory

// User requests cancel from Server B
SessionPrompt.cancel(sessionID) // Cancels Server B's memory state, not A's!
```

### 2. Callback Queues Are Lost

If multiple requests are queued waiting for response:

```typescript
// Server A is processing
state()[sessionID].callbacks = [resolve1, reject1]

// Server B tries to continue processing
state()[sessionID]  // undefined in Server B!
// callbacks array lost
```

### 3. Busy State Doesn't Transfer

```typescript
// Server A has session as "busy"
SessionStatus.set(sessionID, { type: "busy" })

// Server B doesn't see this
SessionStatus.get(sessionID)  // Returns { type: "idle" }
```

### 4. File Corruption Risk

Both servers could modify the same session state file concurrently without coordination.

---

## 7. Event Broadcasting Limitations

### Global Event Mechanism

**File**: `packages/opencode/src/bus/global.ts`

```typescript
export const GlobalBus = new EventEmitter<{
  event: [{ directory: string; payload: any }]
}>()
```

**Used in** `packages/opencode/src/bus/index.ts` (line 73):

```typescript
GlobalBus.emit("event", {
  directory: Instance.directory,
  payload,
})
```

### Capability

- Events can be broadcasted via `/global/event` endpoint
- Server A publishes event → Server B's clients receive via `/global/event` stream
- **HOWEVER**: This is unidirectional broadcast, not coordination
- Does NOT prevent concurrent modifications
- Does NOT provide distributed locking

---

## 8. What Breaks with Multiple Servers

| Component | Current Behavior | Multi-Server Impact |
|-----------|------------------|---------------------|
| **Lock Mechanism** | In-memory Map | Both servers acquire independent locks → race condition |
| **Session Prompt State** | Process-local with abort controller | Server B can't see/cancel Server A's processing |
| **Session Status** | Process-local state | Servers have inconsistent session status views |
| **Callback Queues** | In-memory queue per session | Queued callbacks lost when switching servers |
| **Bus Subscriptions** | Per-instance subscriptions | Different servers receive different events |
| **Abort Signals** | Process-local | Cancellation doesn't propagate across servers |
| **File Writes** | Protected by in-memory locks only | Concurrent writes cause data corruption |
| **Session Busy Check** | Local state check | No inter-server synchronization |

---

## 9. Race Condition Scenarios

### Scenario 1: Concurrent Message Processing

```
Time  Server A                          Server B
────  ──────────────────────────────    ───────────────────────────────
T0    POST /session/123/message
      → SessionPrompt.prompt()
      → state()[123] = {abort, []}

T1                                       POST /session/123/message
                                         → SessionPrompt.prompt()
                                         → start(123) returns controller
                                         → state()[123] = {abort, []}
                                         (Different state map!)

T2    Reading session messages
      Lock.read("message/123/*.json")
      Acquires in-memory lock on Server A

T3                                       Reading session messages
                                         Lock.read("message/123/*.json")
                                         Acquires DIFFERENT in-memory lock
                                         on Server B (same file!)

T4    Session.updateMessage(msg1)
      Lock.write(["message", 123, id])
      Writes file with lock on Server A

T5                                       Session.updateMessage(msg2)
                                         Lock.write(["message", 123, id])
                                         Writes SAME FILE with lock on B
                                         msg2 overwrites msg1!
```

### Scenario 2: Cancellation Failure

```
Time  Server A                          Server B
────  ──────────────────────────────    ───────────────────────────────
T0    Processing message for session 123
      state()[123].abort = controller_A

T1                                       User cancels session 123
                                         cancel(123)
                                         state()[123] = undefined
                                         (No effect on Server A!)

T2    Server A continues processing
      (Doesn't know about cancellation)

T3    Server A completes and writes
      (User expected it to be cancelled)
```

### Scenario 3: Lost Queued Requests

```
Time  Server A                          Server B
────  ──────────────────────────────    ───────────────────────────────
T0    Processing session 123
      state()[123].callbacks = []

T1    New request arrives at Server A
      Queued: callbacks = [resolve1]

T2                                       Session 123 processing completes
                                         Server B handles completion
                                         state()[123] = undefined
                                         (No callbacks to resolve!)

T3    resolve1 never called
      Client hangs forever
```

---

## 10. Production Deployment Requirements

To support multiple servers handling the same sessions, OpenCode would need:

### 1. Distributed File Locking

Replace in-memory `Lock` with external coordination:
- Redis-based distributed locks (Redlock algorithm)
- ZooKeeper/etcd for distributed coordination
- File-level OS locks (flock/fcntl) for single-host deployments

### 2. Shared State Store

Move process-local state to shared storage:
- Redis for session state, callbacks, abort signals
- Database for persistent state
- Distributed cache for performance

### 3. Global Event Pub/Sub

Replace in-memory Bus with distributed messaging:
- Redis Pub/Sub
- NATS
- Apache Kafka
- RabbitMQ

### 4. Session Affinity Routing

If not implementing the above, use load balancer sticky sessions:
- Cookie-based affinity
- IP-based affinity
- Session ID hashing

### Example Architecture for Multi-Server

```
┌─────────────────────────────────────────────────┐
│              Load Balancer                       │
│         (with session affinity)                  │
└──────────┬──────────────┬──────────────┬────────┘
           │              │              │
     ┌─────▼────┐   ┌─────▼────┐   ┌─────▼────┐
     │ Server 1 │   │ Server 2 │   │ Server 3 │
     └─────┬────┘   └─────┬────┘   └─────┬────┘
           │              │              │
           └──────────┬───┴──────────────┘
                      │
              ┌───────▼───────┐
              │    Redis      │
              │ - Locks       │
              │ - Session     │
              │ - Events      │
              └───────┬───────┘
                      │
              ┌───────▼───────┐
              │  Shared FS    │
              │ - Files       │
              │ - Storage     │
              └───────────────┘
```

---

## Summary

### Stateless Components (Can Be Shared)

- Session data files (on disk)
- Session metadata (on disk)
- Configuration (on disk)
- Message files (on disk)

### Stateful Components (Blocking Multi-Server)

- Session prompt state with abort controllers
- In-memory locking mechanism
- Session busy state tracking
- Bus event subscriptions
- Session status tracking
- Callback queue management

### Deployment Options

| Option | Complexity | Guarantee |
|--------|------------|-----------|
| **Single server** | Low | Full consistency |
| **Session affinity** | Medium | Consistency per session |
| **Full distribution** | High | Full horizontal scaling |

---

## Conclusion

OpenCode servers are designed for **single-server deployments** or **session-affinity-based load balancing**. The extensive use of in-memory state for session management, locking, and event broadcasting means that:

1. **Sessions must be handled by the same server** that started processing them
2. **No coordination exists** between multiple server instances
3. **File writes can be corrupted** if multiple servers access the same session
4. **Events are not distributed** across server instances
5. **Cancellation and abort signals** don't propagate between servers

For production deployments requiring multiple servers, implement sticky sessions at the load balancer level, or undertake significant architectural changes to move state management to distributed systems.
