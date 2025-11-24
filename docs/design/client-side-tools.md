# Client-Side Tools Design Document

## Overview

This document describes the design for client-side tools in OpenCode, where clients can register tool definitions with the server, and the server delegates tool execution back to the client.

### Goals

1. **Client Tool Registration**: Allow SDK clients to define and register tools with the server
2. **Server Delegation**: Enable the server to delegate tool execution to the originating client
3. **Bidirectional Communication**: Support real-time communication for tool execution requests/responses
4. **Seamless Integration**: Integrate with existing tool infrastructure (permissions, hooks, streaming)
5. **Multi-Client Support**: Handle multiple clients with different tool sets

### Non-Goals

- Replacing existing server-side tools
- Cross-client tool sharing (tools are scoped to their registering client)
- Persistent tool registration (tools exist only for session lifetime)

---

## Architecture Overview

```
┌─────────────────┐                    ┌─────────────────┐
│   SDK Client    │                    │  OpenCode       │
│                 │                    │  Server         │
│ ┌─────────────┐ │   Register Tools   │                 │
│ │ Tool Defs   │─┼───────────────────►│ ┌─────────────┐ │
│ └─────────────┘ │                    │ │Client Tool  │ │
│                 │                    │ │Registry     │ │
│ ┌─────────────┐ │   Execute Request  │ └─────────────┘ │
│ │ Tool        │◄├────────────────────┤                 │
│ │ Handlers    │ │                    │ ┌─────────────┐ │
│ └──────┬──────┘ │   Execute Result   │ │Session      │ │
│        │        ├───────────────────►│ │Processor    │ │
│        ▼        │                    │ └─────────────┘ │
│ ┌─────────────┐ │                    │                 │
│ │ Local       │ │                    │ ┌─────────────┐ │
│ │ Execution   │ │      Stream        │ │AI Model     │ │
│ └─────────────┘ │◄───────────────────┤ └─────────────┘ │
└─────────────────┘                    └─────────────────┘
```

---

## Protocol Design

### New Message Types

Add to `/packages/opencode/src/session/message-v2.ts`:

```typescript
// Client tool definition sent during registration
export type ClientToolDefinition = {
  id: string
  description: string
  parameters: JsonSchema7  // JSON Schema for tool parameters
}

// Request sent from server to client for tool execution
export type ClientToolExecutionRequest = {
  type: "client-tool-request"
  requestID: string
  sessionID: string
  messageID: string
  callID: string
  tool: string
  input: Record<string, unknown>
}

// Response sent from client to server after execution
export type ClientToolExecutionResponse = {
  type: "client-tool-response"
  requestID: string
  result: ClientToolResult | ClientToolError
}

export type ClientToolResult = {
  status: "success"
  title: string
  output: string
  metadata?: Record<string, unknown>
  attachments?: FilePart[]
}

export type ClientToolError = {
  status: "error"
  error: string
}
```

### New API Endpoints

Add to server API (in `/packages/opencode/src/server/`):

```typescript
// POST /client-tools/register
// Register client tools for a session
interface RegisterClientToolsRequest {
  sessionID: string
  clientID: string
  tools: ClientToolDefinition[]
}

interface RegisterClientToolsResponse {
  registered: string[]  // Tool IDs that were registered
}

// POST /client-tools/result
// Submit tool execution result
interface SubmitToolResultRequest {
  requestID: string
  result: ClientToolResult | ClientToolError
}

// GET /client-tools/pending/:clientID (SSE endpoint)
// Stream pending tool execution requests to client
// Returns: Server-Sent Events stream of ClientToolExecutionRequest

// DELETE /client-tools/unregister
// Unregister client tools
interface UnregisterClientToolsRequest {
  sessionID: string
  clientID: string
  toolIDs?: string[]  // If omitted, unregister all
}
```

### WebSocket Alternative

For lower latency, support WebSocket connections:

```typescript
// WS /client-tools/ws/:clientID
// Bidirectional WebSocket for tool requests/responses

// Client -> Server messages:
type WSClientMessage =
  | { type: "register"; tools: ClientToolDefinition[] }
  | { type: "result"; requestID: string; result: ClientToolResult | ClientToolError }
  | { type: "unregister"; toolIDs?: string[] }

// Server -> Client messages:
type WSServerMessage =
  | { type: "registered"; toolIDs: string[] }
  | { type: "request"; request: ClientToolExecutionRequest }
  | { type: "error"; error: string }
```

---

## Server-Side Implementation

### 1. Client Tool Registry

Create `/packages/opencode/src/tool/client-registry.ts`:

```typescript
import { z } from "zod"
import { Bus } from "../bus"
import { Tool } from "./tool"
import type { ClientToolDefinition, ClientToolExecutionRequest } from "../session/message-v2"

export namespace ClientToolRegistry {
  // Store client tools by clientID -> toolID -> definition
  const registry = new Map<string, Map<string, ClientToolDefinition>>()

  // Pending execution requests by requestID
  const pendingRequests = new Map<string, {
    request: ClientToolExecutionRequest
    resolve: (result: any) => void
    reject: (error: Error) => void
    timeout: Timer
  }>()

  // Event emitter for tool execution requests
  export const Event = {
    ToolRequest: Bus.event(
      "client-tool.request",
      z.object({
        clientID: z.string(),
        request: z.custom<ClientToolExecutionRequest>(),
      })
    ),
  }

  /**
   * Register tools for a client
   */
  export function register(
    clientID: string,
    tools: ClientToolDefinition[]
  ): string[] {
    if (!registry.has(clientID)) {
      registry.set(clientID, new Map())
    }

    const clientTools = registry.get(clientID)!
    const registered: string[] = []

    for (const tool of tools) {
      // Prefix with client ID to avoid collisions
      const toolID = `client_${clientID}_${tool.id}`
      clientTools.set(toolID, {
        ...tool,
        id: toolID,
      })
      registered.push(toolID)
    }

    return registered
  }

  /**
   * Unregister tools for a client
   */
  export function unregister(clientID: string, toolIDs?: string[]): void {
    const clientTools = registry.get(clientID)
    if (!clientTools) return

    if (toolIDs) {
      for (const id of toolIDs) {
        clientTools.delete(id)
      }
    } else {
      registry.delete(clientID)
    }
  }

  /**
   * Get all tools for a client
   */
  export function getTools(clientID: string): ClientToolDefinition[] {
    const clientTools = registry.get(clientID)
    if (!clientTools) return []
    return Array.from(clientTools.values())
  }

  /**
   * Get all client tools across all clients
   */
  export function getAllTools(): Map<string, ClientToolDefinition> {
    const all = new Map<string, ClientToolDefinition>()
    for (const [_, clientTools] of registry) {
      for (const [toolID, tool] of clientTools) {
        all.set(toolID, tool)
      }
    }
    return all
  }

  /**
   * Find which client owns a tool
   */
  export function findClientForTool(toolID: string): string | undefined {
    for (const [clientID, clientTools] of registry) {
      if (clientTools.has(toolID)) {
        return clientID
      }
    }
    return undefined
  }

  /**
   * Execute a client tool
   * Sends request to client and waits for response
   */
  export async function execute(
    clientID: string,
    request: Omit<ClientToolExecutionRequest, "type">,
    timeoutMs: number = 30000
  ): Promise<ClientToolResult> {
    const fullRequest: ClientToolExecutionRequest = {
      type: "client-tool-request",
      ...request,
    }

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        pendingRequests.delete(request.requestID)
        reject(new Error(`Client tool execution timed out after ${timeoutMs}ms`))
      }, timeoutMs)

      pendingRequests.set(request.requestID, {
        request: fullRequest,
        resolve,
        reject,
        timeout,
      })

      // Emit event for client to receive
      Event.ToolRequest.publish({
        clientID,
        request: fullRequest,
      })
    })
  }

  /**
   * Submit result from client
   */
  export function submitResult(
    requestID: string,
    result: ClientToolResult | ClientToolError
  ): boolean {
    const pending = pendingRequests.get(requestID)
    if (!pending) return false

    clearTimeout(pending.timeout)
    pendingRequests.delete(requestID)

    if (result.status === "error") {
      pending.reject(new Error(result.error))
    } else {
      pending.resolve(result)
    }

    return true
  }

  /**
   * Clean up all tools for a client (on disconnect)
   */
  export function cleanup(clientID: string): void {
    // Cancel all pending requests for this client
    for (const [requestID, pending] of pendingRequests) {
      if (pending.request.requestID.startsWith(clientID)) {
        clearTimeout(pending.timeout)
        pending.reject(new Error("Client disconnected"))
        pendingRequests.delete(requestID)
      }
    }

    // Remove all tools
    registry.delete(clientID)
  }
}
```

### 2. Integration with Tool Registry

Modify `/packages/opencode/src/tool/registry.ts`:

```typescript
import { ClientToolRegistry } from "./client-registry"

export namespace ToolRegistry {
  // ... existing code ...

  /**
   * Get all tools including client tools
   */
  export async function tools(
    providerID: string,
    modelID: string,
    clientID?: string
  ) {
    const serverTools = await all()
    const result = await Promise.all(
      serverTools.map(async (t) => ({
        id: t.id,
        ...(await t.init()),
      })),
    )

    // Add client tools if clientID provided
    if (clientID) {
      const clientTools = ClientToolRegistry.getTools(clientID)
      for (const tool of clientTools) {
        result.push({
          id: tool.id,
          description: tool.description,
          parameters: tool.parameters as any,
          execute: createClientToolExecutor(clientID, tool.id),
        })
      }
    }

    return result
  }

  /**
   * Create executor function for client tool
   */
  function createClientToolExecutor(clientID: string, toolID: string) {
    return async (
      args: Record<string, unknown>,
      ctx: Tool.Context
    ): Promise<Tool.Result> => {
      const requestID = `${clientID}_${ctx.callID}_${Date.now()}`

      const result = await ClientToolRegistry.execute(clientID, {
        requestID,
        sessionID: ctx.sessionID,
        messageID: ctx.messageID,
        callID: ctx.callID!,
        tool: toolID,
        input: args,
      })

      return {
        title: result.title,
        metadata: result.metadata ?? {},
        output: result.output,
        attachments: result.attachments,
      }
    }
  }
}
```

### 3. API Routes

Create `/packages/opencode/src/server/routes/client-tools.ts`:

```typescript
import { Hono } from "hono"
import { streamSSE } from "hono/streaming"
import { ClientToolRegistry } from "../../tool/client-registry"
import { Identifier } from "../../util/identifier"

export const clientToolsRouter = new Hono()

// Register client tools
clientToolsRouter.post("/register", async (c) => {
  const body = await c.req.json()
  const { sessionID, clientID, tools } = body

  const registered = ClientToolRegistry.register(clientID, tools)

  return c.json({ registered })
})

// Unregister client tools
clientToolsRouter.delete("/unregister", async (c) => {
  const body = await c.req.json()
  const { sessionID, clientID, toolIDs } = body

  ClientToolRegistry.unregister(clientID, toolIDs)

  return c.json({ success: true })
})

// Submit tool execution result
clientToolsRouter.post("/result", async (c) => {
  const body = await c.req.json()
  const { requestID, result } = body

  const success = ClientToolRegistry.submitResult(requestID, result)

  if (!success) {
    return c.json({ error: "Unknown request ID" }, 404)
  }

  return c.json({ success: true })
})

// SSE endpoint for tool execution requests
clientToolsRouter.get("/pending/:clientID", async (c) => {
  const clientID = c.req.param("clientID")

  return streamSSE(c, async (stream) => {
    // Subscribe to tool request events
    const unsubscribe = ClientToolRegistry.Event.ToolRequest.subscribe(
      async (event) => {
        if (event.clientID === clientID) {
          await stream.writeSSE({
            event: "tool-request",
            data: JSON.stringify(event.request),
          })
        }
      }
    )

    // Keep connection alive
    const keepAlive = setInterval(async () => {
      await stream.writeSSE({
        event: "ping",
        data: "",
      })
    }, 30000)

    // Cleanup on disconnect
    c.req.raw.signal.addEventListener("abort", () => {
      unsubscribe()
      clearInterval(keepAlive)
      ClientToolRegistry.cleanup(clientID)
    })

    // Block until client disconnects
    await new Promise(() => {})
  })
})
```

### 4. WebSocket Handler

Create `/packages/opencode/src/server/routes/client-tools-ws.ts`:

```typescript
import { Hono } from "hono"
import { upgradeWebSocket } from "hono/cloudflare-workers"
import { ClientToolRegistry } from "../../tool/client-registry"

export const clientToolsWSRouter = new Hono()

clientToolsWSRouter.get(
  "/ws/:clientID",
  upgradeWebSocket((c) => {
    const clientID = c.req.param("clientID")
    let unsubscribe: (() => void) | undefined

    return {
      onOpen(event, ws) {
        // Subscribe to tool requests for this client
        unsubscribe = ClientToolRegistry.Event.ToolRequest.subscribe(
          (evt) => {
            if (evt.clientID === clientID) {
              ws.send(JSON.stringify({
                type: "request",
                request: evt.request,
              }))
            }
          }
        )
      },

      onMessage(event, ws) {
        try {
          const message = JSON.parse(event.data as string)

          switch (message.type) {
            case "register": {
              const registered = ClientToolRegistry.register(
                clientID,
                message.tools
              )
              ws.send(JSON.stringify({
                type: "registered",
                toolIDs: registered,
              }))
              break
            }

            case "result": {
              ClientToolRegistry.submitResult(
                message.requestID,
                message.result
              )
              break
            }

            case "unregister": {
              ClientToolRegistry.unregister(clientID, message.toolIDs)
              break
            }
          }
        } catch (error) {
          ws.send(JSON.stringify({
            type: "error",
            error: String(error),
          }))
        }
      },

      onClose() {
        unsubscribe?.()
        ClientToolRegistry.cleanup(clientID)
      },

      onError(event) {
        unsubscribe?.()
        ClientToolRegistry.cleanup(clientID)
      },
    }
  })
)
```

---

## Client SDK Implementation

### 1. Types

Add to `/packages/sdk/js/src/types.ts`:

```typescript
export interface ClientToolDefinition {
  id: string
  description: string
  parameters: Record<string, unknown>  // JSON Schema
}

export interface ClientToolHandler {
  (input: Record<string, unknown>, context: ClientToolContext): Promise<ClientToolResult>
}

export interface ClientToolContext {
  sessionID: string
  messageID: string
  callID: string
  signal: AbortSignal
}

export interface ClientToolResult {
  title: string
  output: string
  metadata?: Record<string, unknown>
}

export interface ClientTool {
  definition: ClientToolDefinition
  handler: ClientToolHandler
}

export interface ClientToolsConfig {
  /** Timeout for tool execution in ms (default: 30000) */
  timeout?: number
  /** Use WebSocket instead of SSE (default: false) */
  useWebSocket?: boolean
}
```

### 2. Client Tools Manager

Create `/packages/sdk/js/src/client-tools.ts`:

```typescript
import type {
  ClientTool,
  ClientToolDefinition,
  ClientToolHandler,
  ClientToolResult,
  ClientToolsConfig,
} from "./types"

export class ClientToolsManager {
  private clientID: string
  private baseUrl: string
  private tools = new Map<string, ClientTool>()
  private eventSource?: EventSource
  private ws?: WebSocket
  private config: Required<ClientToolsConfig>
  private abortController = new AbortController()

  constructor(
    clientID: string,
    baseUrl: string,
    config?: ClientToolsConfig
  ) {
    this.clientID = clientID
    this.baseUrl = baseUrl
    this.config = {
      timeout: config?.timeout ?? 30000,
      useWebSocket: config?.useWebSocket ?? false,
    }
  }

  /**
   * Register a tool with the server
   */
  async register(
    id: string,
    definition: Omit<ClientToolDefinition, "id">,
    handler: ClientToolHandler
  ): Promise<void> {
    const tool: ClientTool = {
      definition: { id, ...definition },
      handler,
    }
    this.tools.set(id, tool)

    // If already connected, register immediately
    if (this.eventSource || this.ws) {
      await this.syncTools()
    }
  }

  /**
   * Unregister a tool
   */
  async unregister(id: string): Promise<void> {
    this.tools.delete(id)

    if (this.eventSource || this.ws) {
      await fetch(`${this.baseUrl}/client-tools/unregister`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          clientID: this.clientID,
          toolIDs: [id],
        }),
      })
    }
  }

  /**
   * Start listening for tool execution requests
   */
  async connect(sessionID: string): Promise<void> {
    // Register all tools first
    await this.syncTools()

    if (this.config.useWebSocket) {
      await this.connectWebSocket()
    } else {
      await this.connectSSE()
    }
  }

  /**
   * Stop listening and cleanup
   */
  disconnect(): void {
    this.abortController.abort()
    this.eventSource?.close()
    this.ws?.close()
  }

  private async syncTools(): Promise<void> {
    const definitions = Array.from(this.tools.values()).map(t => t.definition)

    await fetch(`${this.baseUrl}/client-tools/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        clientID: this.clientID,
        tools: definitions,
      }),
    })
  }

  private async connectSSE(): Promise<void> {
    this.eventSource = new EventSource(
      `${this.baseUrl}/client-tools/pending/${this.clientID}`
    )

    this.eventSource.addEventListener("tool-request", async (event) => {
      const request = JSON.parse(event.data)
      await this.handleToolRequest(request)
    })

    this.eventSource.onerror = (error) => {
      console.error("Client tools SSE error:", error)
    }
  }

  private async connectWebSocket(): Promise<void> {
    const wsUrl = this.baseUrl.replace(/^http/, "ws")
    this.ws = new WebSocket(`${wsUrl}/client-tools/ws/${this.clientID}`)

    this.ws.onopen = async () => {
      // Register tools via WebSocket
      const definitions = Array.from(this.tools.values()).map(t => t.definition)
      this.ws!.send(JSON.stringify({
        type: "register",
        tools: definitions,
      }))
    }

    this.ws.onmessage = async (event) => {
      const message = JSON.parse(event.data)

      if (message.type === "request") {
        await this.handleToolRequest(message.request)
      }
    }

    this.ws.onerror = (error) => {
      console.error("Client tools WebSocket error:", error)
    }
  }

  private async handleToolRequest(request: {
    requestID: string
    sessionID: string
    messageID: string
    callID: string
    tool: string
    input: Record<string, unknown>
  }): Promise<void> {
    // Extract original tool ID (remove client_ prefix)
    const prefixedID = request.tool
    const originalID = prefixedID.replace(`client_${this.clientID}_`, "")

    const tool = this.tools.get(originalID)

    if (!tool) {
      await this.submitResult(request.requestID, {
        status: "error",
        error: `Unknown tool: ${originalID}`,
      })
      return
    }

    try {
      // Create abort controller for this execution
      const controller = new AbortController()
      const timeout = setTimeout(() => {
        controller.abort()
      }, this.config.timeout)

      const result = await tool.handler(request.input, {
        sessionID: request.sessionID,
        messageID: request.messageID,
        callID: request.callID,
        signal: controller.signal,
      })

      clearTimeout(timeout)

      await this.submitResult(request.requestID, {
        status: "success",
        title: result.title,
        output: result.output,
        metadata: result.metadata,
      })
    } catch (error) {
      await this.submitResult(request.requestID, {
        status: "error",
        error: error instanceof Error ? error.message : String(error),
      })
    }
  }

  private async submitResult(
    requestID: string,
    result: { status: "success" | "error"; [key: string]: unknown }
  ): Promise<void> {
    if (this.ws) {
      this.ws.send(JSON.stringify({
        type: "result",
        requestID,
        result,
      }))
    } else {
      await fetch(`${this.baseUrl}/client-tools/result`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ requestID, result }),
      })
    }
  }
}
```

### 3. Integration with OpencodeClient

Modify `/packages/sdk/js/src/client.ts`:

```typescript
import { ClientToolsManager } from "./client-tools"

export class OpencodeClient {
  private _client: Client
  private _clientTools?: ClientToolsManager
  private _clientID: string

  constructor(config: { client: Client }) {
    this._client = config.client
    this._clientID = crypto.randomUUID()
  }

  /**
   * Get client tools manager for registering and handling client-side tools
   */
  get clientTools(): ClientToolsManager {
    if (!this._clientTools) {
      const baseUrl = (this._client as any).baseUrl
      this._clientTools = new ClientToolsManager(this._clientID, baseUrl)
    }
    return this._clientTools
  }

  /**
   * Start a session with client tools support
   */
  async startSession(options?: {
    tools?: boolean
  }): Promise<SessionHandle> {
    const session = await this.session.create()

    if (options?.tools !== false) {
      await this.clientTools.connect(session.id)
    }

    return {
      session,
      prompt: (input: string) => this.session.prompt(session.id, input),
      close: () => {
        this.clientTools.disconnect()
      },
    }
  }
}

interface SessionHandle {
  session: Session
  prompt: (input: string) => Promise<Message>
  close: () => void
}
```

---

## Security Considerations

### 1. Client Authentication

```typescript
// Validate client owns the session
export function validateClientSession(
  clientID: string,
  sessionID: string
): boolean {
  const session = Session.get(sessionID)
  return session?.clientID === clientID
}

// Add clientID to session creation
export async function createSession(clientID: string) {
  return Session.create({
    clientID,
    // ... other fields
  })
}
```

### 2. Tool Sandboxing

- Client tools run in the client's environment (inherently sandboxed from server)
- Server tools continue to run on server
- Clear naming convention distinguishes client vs server tools

### 3. Input Validation

```typescript
// Validate tool input against JSON Schema before sending to client
import Ajv from "ajv"

const ajv = new Ajv()

export function validateToolInput(
  tool: ClientToolDefinition,
  input: Record<string, unknown>
): boolean {
  const validate = ajv.compile(tool.parameters)
  return validate(input)
}
```

### 4. Timeout and Rate Limiting

```typescript
// Server-side timeout for client tool execution
const CLIENT_TOOL_TIMEOUT = 30000 // 30 seconds

// Rate limiting per client
const rateLimiter = new Map<string, { count: number; reset: number }>()

export function checkRateLimit(clientID: string): boolean {
  const limit = rateLimiter.get(clientID)
  const now = Date.now()

  if (!limit || now > limit.reset) {
    rateLimiter.set(clientID, {
      count: 1,
      reset: now + 60000, // 1 minute window
    })
    return true
  }

  if (limit.count >= 100) { // 100 requests per minute
    return false
  }

  limit.count++
  return true
}
```

### 5. Permission Integration

```typescript
// Add client tool permission to Agent
export interface AgentPermission {
  // ... existing permissions
  client_tools: "allow" | "ask" | "deny"
}

// Check permission before executing client tool
if (agent.permission.client_tools === "deny") {
  throw new Error("Client tools are not allowed for this agent")
}

if (agent.permission.client_tools === "ask") {
  await Permission.ask({
    type: "client_tool",
    tool: toolID,
    sessionID,
    messageID,
    callID,
  })
}
```

---

## Error Handling

### 1. Connection Errors

```typescript
// Auto-reconnect with exponential backoff
class ClientToolsManager {
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5

  private async reconnect(): Promise<void> {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      throw new Error("Max reconnection attempts reached")
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    await new Promise(resolve => setTimeout(resolve, delay))

    this.reconnectAttempts++
    await this.connect(this.sessionID)
    this.reconnectAttempts = 0
  }
}
```

### 2. Tool Execution Errors

```typescript
// Graceful error handling in tool execution
try {
  const result = await tool.handler(input, context)
  return { status: "success", ...result }
} catch (error) {
  // Log error for debugging
  console.error(`Client tool ${toolID} failed:`, error)

  // Return error to server
  return {
    status: "error",
    error: error instanceof Error ? error.message : "Unknown error",
  }
}
```

### 3. Timeout Handling

```typescript
// Server-side timeout
const timeoutPromise = new Promise((_, reject) => {
  setTimeout(() => {
    reject(new Error(`Client tool timed out after ${timeout}ms`))
  }, timeout)
})

const result = await Promise.race([
  ClientToolRegistry.execute(clientID, request),
  timeoutPromise,
])
```

---

## Usage Examples

### Basic Client Tool

```typescript
import { createOpencode } from "@opencode/sdk"

const { client, server } = await createOpencode()

// Register a client tool
await client.clientTools.register(
  "get_local_time",
  {
    description: "Get the current local time on the client machine",
    parameters: {
      type: "object",
      properties: {
        timezone: {
          type: "string",
          description: "Timezone (e.g., 'America/New_York')",
        },
      },
    },
  },
  async (input, ctx) => {
    const tz = input.timezone as string || "UTC"
    const time = new Date().toLocaleString("en-US", { timeZone: tz })

    return {
      title: `Local time (${tz})`,
      output: time,
    }
  }
)

// Start session with client tools
const { session, prompt, close } = await client.startSession()

// Use the session - model can now call get_local_time
const response = await prompt("What time is it locally?")

// Cleanup
close()
server.close()
```

### File System Access Tool

```typescript
import { readFile } from "fs/promises"

await client.clientTools.register(
  "read_local_file",
  {
    description: "Read a file from the client's local filesystem",
    parameters: {
      type: "object",
      properties: {
        path: {
          type: "string",
          description: "Absolute path to the file",
        },
      },
      required: ["path"],
    },
  },
  async (input) => {
    const path = input.path as string
    const content = await readFile(path, "utf-8")

    return {
      title: `Read ${path}`,
      output: content,
    }
  }
)
```

### Database Query Tool

```typescript
import { createConnection } from "mysql2/promise"

const connection = await createConnection({
  host: "localhost",
  user: "root",
  database: "myapp",
})

await client.clientTools.register(
  "query_database",
  {
    description: "Execute a read-only SQL query on the local database",
    parameters: {
      type: "object",
      properties: {
        query: {
          type: "string",
          description: "SQL SELECT query to execute",
        },
      },
      required: ["query"],
    },
  },
  async (input) => {
    const query = input.query as string

    // Security: only allow SELECT queries
    if (!query.trim().toLowerCase().startsWith("select")) {
      throw new Error("Only SELECT queries are allowed")
    }

    const [rows] = await connection.execute(query)

    return {
      title: "Query results",
      output: JSON.stringify(rows, null, 2),
      metadata: { rowCount: (rows as any[]).length },
    }
  }
)
```

---

## Implementation Plan

### Phase 1: Core Infrastructure
1. Add message types to `message-v2.ts`
2. Create `ClientToolRegistry` module
3. Add API routes for registration and results
4. Integrate with `ToolRegistry`

### Phase 2: SDK Implementation
1. Create `ClientToolsManager` class
2. Add SSE connection support
3. Integrate with `OpencodeClient`
4. Add TypeScript types

### Phase 3: WebSocket Support
1. Add WebSocket handler on server
2. Add WebSocket connection option in SDK
3. Implement bidirectional messaging

### Phase 4: Security & Polish
1. Add client authentication
2. Implement rate limiting
3. Add permission integration
4. Add comprehensive error handling

### Phase 5: Testing & Documentation
1. Unit tests for registry and manager
2. Integration tests for full flow
3. Update SDK documentation
4. Add usage examples

---

## Appendix: Modified Files Summary

### New Files
- `/packages/opencode/src/tool/client-registry.ts`
- `/packages/opencode/src/server/routes/client-tools.ts`
- `/packages/opencode/src/server/routes/client-tools-ws.ts`
- `/packages/sdk/js/src/client-tools.ts`
- `/packages/sdk/js/src/types.ts` (new types)

### Modified Files
- `/packages/opencode/src/session/message-v2.ts` (new message types)
- `/packages/opencode/src/tool/registry.ts` (integrate client tools)
- `/packages/opencode/src/server/index.ts` (add routes)
- `/packages/sdk/js/src/client.ts` (add clientTools property)
- `/packages/sdk/js/src/index.ts` (export new types)

---

## Future Enhancements

1. **Tool Discovery**: Allow clients to query available server tools
2. **Tool Streaming**: Support streaming output from client tools
3. **Tool Composition**: Allow client tools to call server tools
4. **Persistent Tools**: Option to persist tool registrations across sessions
5. **Tool Marketplace**: Share and discover community tools
6. **Tool Versioning**: Support multiple versions of the same tool
