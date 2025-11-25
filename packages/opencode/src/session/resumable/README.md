# Resumable Sessions with Async Tool Tracking

## Overview

This module implements resumable agent sessions that can survive agent shutdowns during long-running asynchronous tool executions. When a tool call takes a long time (e.g., external API calls, background jobs, human-in-the-loop operations), the agent can shut down and resume later when the result is available.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Resumable Session Flow                             │
└─────────────────────────────────────────────────────────────────────────────┘

                    ┌──────────────────────┐
                    │    User Request      │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │  Session Processing  │
                    │  (SessionPrompt)     │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │   Tool Execution     │
                    │   (sync or async?)   │
                    └──────────┬───────────┘
                               │
           ┌───────────────────┴───────────────────┐
           │                                       │
   ┌───────▼───────┐                      ┌───────▼───────┐
   │  Sync Tool    │                      │  Async Tool   │
   │  (immediate)  │                      │  (long-run)   │
   └───────┬───────┘                      └───────┬───────┘
           │                                       │
           │                              ┌────────▼────────┐
           │                              │ PendingToolCall │
           │                              │   persisted     │
           │                              └────────┬────────┘
           │                                       │
           │                              ┌────────▼────────┐
           │                              │ Agent can       │
           │                              │ shut down       │
           │                              └────────┬────────┘
           │                                       │
           │                              ┌────────▼────────┐
           │                              │ External system │
           │                              │ completes work  │
           │                              └────────┬────────┘
           │                                       │
           │                              ┌────────▼────────┐
           │                              │ POST /async-    │
           │                              │ tool/result     │
           │                              └────────┬────────┘
           │                                       │
           │                              ┌────────▼────────┐
           │                              │ SessionResumer  │
           │                              │ loads state     │
           │                              └────────┬────────┘
           │                                       │
           └───────────────────┬───────────────────┘
                               │
                    ┌──────────▼───────────┐
                    │  Continue Session    │
                    │  Processing          │
                    └──────────────────────┘
```

## Components

### 1. PendingToolCall Storage (`pending-tool.ts`)

Persists tool call state that survives agent shutdown:

```typescript
interface PendingToolCall {
  id: string                    // Unique ID for this pending call
  sessionID: string             // Parent session
  messageID: string             // Parent message
  callID: string                // Tool call ID (from LLM)
  tool: string                  // Tool name
  input: Record<string, any>    // Tool parameters
  status: 'waiting' | 'processing' | 'completed' | 'failed' | 'expired'
  webhookURL?: string           // Optional callback URL
  externalRef?: string          // External system reference (job ID, etc.)
  timeout?: number              // Expiration time (ms since epoch)
  time: {
    created: number
    started?: number
    completed?: number
  }
  result?: {
    title: string
    output: string
    metadata: Record<string, any>
  }
  error?: string
}
```

Storage path: `~/.local/share/opencode/storage/pending_tool/{id}.json`

### 2. AsyncToolRegistry (`async-tool-registry.ts`)

Manages async tool definitions and execution:

```typescript
interface AsyncToolDefinition {
  id: string
  description: string
  parameters: z.ZodType
  // Returns immediately with a pending reference
  execute: (input, ctx) => Promise<{
    pendingID: string          // ID to track this execution
    externalRef?: string       // External system reference
    estimatedDuration?: number // Hint for timeout
  }>
  // Optional: validate result before accepting
  validateResult?: (result) => boolean
}
```

### 3. SessionResumer (`session-resumer.ts`)

Handles session resumption when tool results arrive:

```typescript
namespace SessionResumer {
  // Submit result and resume session
  async function submitResult(pendingID: string, result: ToolResult): Promise<void>

  // Mark as failed and resume session
  async function submitError(pendingID: string, error: string): Promise<void>

  // Find sessions waiting for tool results (startup recovery)
  async function findWaitingSessions(): Promise<PendingToolCall[]>

  // Resume a session from a pending tool call
  async function resume(pendingToolCall: PendingToolCall): Promise<void>
}
```

### 4. API Endpoints

```
POST /async-tool/result
  Body: { pendingID, result: { title, output, metadata } }
  → Submits result and resumes session

POST /async-tool/error
  Body: { pendingID, error: string }
  → Submits error and resumes session

GET /async-tool/pending
  → Lists all pending async tool calls

GET /async-tool/pending/:id
  → Gets status of specific pending call

DELETE /async-tool/pending/:id
  → Cancels pending call
```

### 5. Webhook Support (Optional)

For external systems that prefer push-based notification:

```typescript
// When creating async tool call
const pending = await AsyncToolRegistry.execute(tool, input, {
  webhookURL: 'https://external-system.com/callback',
  webhookSecret: 'shared-secret'
})

// External system calls back
POST {webhookURL}
  Headers: X-Webhook-Signature: HMAC-SHA256(body, secret)
  Body: { pendingID, result/error }
```

## Storage Schema

### File-Based Storage (Primary)

```
~/.local/share/opencode/storage/
├── pending_tool/
│   ├── {pendingID}.json         # Individual pending calls
│   └── index.json               # Index for fast lookups
├── session/
│   └── {projectID}/
│       └── {sessionID}.json     # Session with resume state
└── ...
```

### Index Structure

```typescript
// pending_tool/index.json
interface PendingToolIndex {
  bySession: Record<sessionID, pendingID[]>
  byStatus: Record<status, pendingID[]>
  byExpiration: Array<{ pendingID: string, expiresAt: number }>
}
```

## Implementation Flow

### 1. Tool Marked as Async

```typescript
// In tool definition
export const LongRunningTool: Tool.Info = {
  id: 'long-running',
  async: true,  // Mark as async
  init: () => ({
    description: 'A tool that takes a long time',
    parameters: z.object({ ... }),
    async execute(input, ctx) {
      // Start external process
      const jobID = await externalSystem.startJob(input)

      // Return immediately with pending reference
      return {
        async: true,
        pendingID: ctx.pendingID,
        externalRef: jobID,
        output: `Job ${jobID} started, will complete asynchronously`
      }
    }
  })
}
```

### 2. Session Processing Handles Async

```typescript
// In SessionProcessor
case 'tool-result': {
  const match = toolcalls[value.toolCallId]
  if (match && value.output.async) {
    // Store pending state
    await PendingToolCall.create({
      id: value.output.pendingID,
      sessionID: input.sessionID,
      messageID: input.assistantMessage.id,
      callID: value.toolCallId,
      tool: value.toolName,
      input: value.input,
      status: 'waiting',
      externalRef: value.output.externalRef,
      time: { created: Date.now() }
    })

    // Update tool part as waiting
    await Session.updatePart({
      ...match,
      state: {
        status: 'waiting_async',
        input: value.input,
        pendingID: value.output.pendingID,
        time: { start: Date.now() }
      }
    })

    // Session can now safely stop
    return 'waiting_async'
  }
  // ... normal flow
}
```

### 3. Result Submission Triggers Resume

```typescript
// POST /async-tool/result handler
async function handleAsyncResult(req) {
  const { pendingID, result } = req.body

  // Load and validate pending call
  const pending = await PendingToolCall.get(pendingID)
  if (!pending) throw new Error('Unknown pending ID')
  if (pending.status !== 'waiting') throw new Error('Not waiting')

  // Update pending state
  await PendingToolCall.complete(pendingID, result)

  // Update the tool part in the message
  await Session.updatePart({
    id: pending.partID,
    messageID: pending.messageID,
    sessionID: pending.sessionID,
    type: 'tool',
    callID: pending.callID,
    tool: pending.tool,
    state: {
      status: 'completed',
      input: pending.input,
      output: result.output,
      title: result.title,
      metadata: result.metadata,
      time: {
        start: pending.time.started || pending.time.created,
        end: Date.now()
      }
    }
  })

  // Resume session processing
  await SessionResumer.resume(pending.sessionID)
}
```

### 4. Startup Recovery

```typescript
// On server startup
async function recoverPendingSessions() {
  const pending = await PendingToolCall.listByStatus('waiting')

  for (const call of pending) {
    // Check for expiration
    if (call.timeout && Date.now() > call.timeout) {
      await PendingToolCall.expire(call.id)
      await SessionResumer.resumeWithError(call, 'Tool execution timed out')
      continue
    }

    // Mark session as waiting for async result
    await Session.setStatus(call.sessionID, {
      type: 'waiting_async',
      pendingID: call.id,
      tool: call.tool
    })
  }
}
```

## Alternative Storage Options

### Redis (Optional Fast Index)

If Redis is available, use it for fast lookups while keeping JSON files as source of truth:

```typescript
// Redis keys
pending:by-session:{sessionID} = Set<pendingID>
pending:by-status:{status} = Set<pendingID>
pending:expiration = SortedSet<pendingID, expiresAt>

// On write
await Storage.write(['pending_tool', id], data)
await redis.sadd(`pending:by-session:${sessionID}`, id)
await redis.sadd(`pending:by-status:${status}`, id)
if (timeout) await redis.zadd('pending:expiration', timeout, id)

// On lookup
const ids = await redis.smembers(`pending:by-session:${sessionID}`)
return Promise.all(ids.map(id => Storage.read(['pending_tool', id])))
```

### SQLite (Alternative)

For more complex queries, SQLite can be used:

```sql
CREATE TABLE pending_tool_calls (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  call_id TEXT NOT NULL,
  tool TEXT NOT NULL,
  input TEXT NOT NULL,  -- JSON
  status TEXT NOT NULL,
  external_ref TEXT,
  timeout INTEGER,
  created_at INTEGER NOT NULL,
  started_at INTEGER,
  completed_at INTEGER,
  result TEXT,  -- JSON
  error TEXT
);

CREATE INDEX idx_pending_session ON pending_tool_calls(session_id);
CREATE INDEX idx_pending_status ON pending_tool_calls(status);
CREATE INDEX idx_pending_timeout ON pending_tool_calls(timeout) WHERE timeout IS NOT NULL;
```

## Session State Extensions

Add new session status for async waiting:

```typescript
// In SessionStatus
export const Status = z.discriminatedUnion('type', [
  z.object({ type: z.literal('idle') }),
  z.object({ type: z.literal('busy') }),
  z.object({ type: z.literal('retry'), attempt: z.number(), message: z.string(), next: z.number() }),
  // New: waiting for async tool
  z.object({
    type: z.literal('waiting_async'),
    pendingID: z.string(),
    tool: z.string(),
    since: z.number(),
    timeout: z.number().optional()
  })
])
```

## Error Handling

### Tool Execution Timeout

```typescript
// Configurable per-tool timeout
const DEFAULT_ASYNC_TIMEOUT = 24 * 60 * 60 * 1000  // 24 hours

// Background job checks for expired calls
async function checkExpiredCalls() {
  const expired = await PendingToolCall.findExpired()
  for (const call of expired) {
    await PendingToolCall.expire(call.id)
    await SessionResumer.resumeWithError(call, 'Tool execution timed out')
  }
}
```

### Agent Crash During Processing

The system is designed to handle crashes:

1. Pending calls are persisted immediately when created
2. On startup, recover all `waiting` status calls
3. Update any stale `processing` status to `waiting`

## Security Considerations

1. **Pending ID is cryptographically random** - prevents guessing
2. **Webhook signatures** - HMAC verification for callbacks
3. **Session validation** - verify session exists before accepting results
4. **Rate limiting** - prevent abuse of result submission endpoint
5. **Expiration** - automatic cleanup of stale pending calls

## Usage Example

```typescript
// Register an async tool
ToolRegistry.register({
  id: 'deploy-preview',
  async: true,
  init: () => ({
    description: 'Deploy a preview environment',
    parameters: z.object({
      branch: z.string(),
      config: z.record(z.string())
    }),
    async execute(input, ctx) {
      // Trigger deployment
      const deployment = await deploymentService.create({
        branch: input.branch,
        config: input.config,
        callbackURL: `${serverURL}/async-tool/webhook`,
        callbackID: ctx.pendingID
      })

      return {
        async: true,
        pendingID: ctx.pendingID,
        externalRef: deployment.id,
        estimatedDuration: 5 * 60 * 1000,  // 5 minutes
        output: `Deployment ${deployment.id} started`
      }
    }
  })
})

// External system calls back when done
POST /async-tool/webhook
{
  "pendingID": "01HX...",
  "result": {
    "title": "Deployment Complete",
    "output": "Preview deployed to https://preview-123.example.com",
    "metadata": {
      "url": "https://preview-123.example.com",
      "deploymentID": "deploy-123"
    }
  }
}
```

## Files

- `pending-tool.ts` - PendingToolCall storage and management
- `async-tool-registry.ts` - Async tool definition and execution
- `session-resumer.ts` - Session resumption logic
- `recovery.ts` - Startup recovery routines
- `index.ts` - Public API exports
