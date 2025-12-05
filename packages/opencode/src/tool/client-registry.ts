import z from "zod"
import { Bus } from "../bus"
import { Log } from "../util/log"

/**
 * Client Tool Registry
 *
 * Manages client-side tools that are registered by SDK clients and executed
 * on the client rather than the server. When the AI model calls a client tool,
 * the server delegates execution to the originating client.
 */
export namespace ClientToolRegistry {
  const log = Log.create({ service: "client-tool-registry" })

  // ============================================================================
  // Types
  // ============================================================================

  export const ClientToolDefinition = z
    .object({
      id: z.string(),
      description: z.string(),
      parameters: z.record(z.string(), z.any()),
    })
    .meta({ ref: "ClientToolDefinition" })
  export type ClientToolDefinition = z.infer<typeof ClientToolDefinition>

  export const ClientToolExecutionRequest = z
    .object({
      type: z.literal("client-tool-request"),
      requestID: z.string(),
      sessionID: z.string(),
      messageID: z.string(),
      callID: z.string(),
      tool: z.string(),
      input: z.record(z.string(), z.any()),
    })
    .meta({ ref: "ClientToolExecutionRequest" })
  export type ClientToolExecutionRequest = z.infer<typeof ClientToolExecutionRequest>

  export const ClientToolResult = z
    .object({
      status: z.literal("success"),
      title: z.string(),
      output: z.string(),
      metadata: z.record(z.string(), z.any()).optional(),
    })
    .meta({ ref: "ClientToolResult" })
  export type ClientToolResult = z.infer<typeof ClientToolResult>

  export const ClientToolError = z
    .object({
      status: z.literal("error"),
      error: z.string(),
    })
    .meta({ ref: "ClientToolError" })
  export type ClientToolError = z.infer<typeof ClientToolError>

  export const ClientToolResponse = z.discriminatedUnion("status", [ClientToolResult, ClientToolError]).meta({
    ref: "ClientToolResponse",
  })
  export type ClientToolResponse = z.infer<typeof ClientToolResponse>

  // ============================================================================
  // Events
  // ============================================================================

  export const Event = {
    /** Emitted when tool execution is requested from a client */
    ToolRequest: Bus.event(
      "client-tool.request",
      z.object({
        clientID: z.string(),
        request: ClientToolExecutionRequest,
      }),
    ),
    /** Emitted when client tools are registered */
    Registered: Bus.event(
      "client-tool.registered",
      z.object({
        clientID: z.string(),
        toolIDs: z.array(z.string()),
      }),
    ),
    /** Emitted when client tools are unregistered */
    Unregistered: Bus.event(
      "client-tool.unregistered",
      z.object({
        clientID: z.string(),
        toolIDs: z.array(z.string()),
      }),
    ),
    /** Emitted when a client tool starts executing */
    Executing: Bus.event(
      "client-tool.executing",
      z.object({
        sessionID: z.string(),
        messageID: z.string(),
        callID: z.string(),
        tool: z.string(),
        clientID: z.string(),
      }),
    ),
    /** Emitted when a client tool completes successfully */
    Completed: Bus.event(
      "client-tool.completed",
      z.object({
        sessionID: z.string(),
        messageID: z.string(),
        callID: z.string(),
        tool: z.string(),
        clientID: z.string(),
        success: z.literal(true),
      }),
    ),
    /** Emitted when a client tool fails */
    Failed: Bus.event(
      "client-tool.failed",
      z.object({
        sessionID: z.string(),
        messageID: z.string(),
        callID: z.string(),
        tool: z.string(),
        clientID: z.string(),
        error: z.string(),
      }),
    ),
  }

  // ============================================================================
  // State
  // ============================================================================

  /** Store client tools by clientID -> toolID -> definition */
  const registry = new Map<string, Map<string, ClientToolDefinition>>()

  /** Pending execution requests by requestID */
  const pendingRequests = new Map<
    string,
    {
      request: ClientToolExecutionRequest
      clientID: string
      resolve: (result: ClientToolResult) => void
      reject: (error: Error) => void
      timeout: Timer
    }
  >()

  // ============================================================================
  // Public API
  // ============================================================================

  /**
   * Register tools for a client.
   * Tool IDs are prefixed with `client_{clientID}_` to avoid collisions.
   */
  export function register(clientID: string, tools: ClientToolDefinition[]): string[] {
    log.info("registering tools", { clientID, count: tools.length })

    if (!registry.has(clientID)) {
      registry.set(clientID, new Map())
    }

    const clientTools = registry.get(clientID)!
    const registered: string[] = []

    for (const tool of tools) {
      const toolID = prefixToolID(clientID, tool.id)
      clientTools.set(toolID, {
        ...tool,
        id: toolID,
      })
      registered.push(toolID)
      log.info("registered tool", { clientID, toolID })
    }

    // Emit registration event
    Bus.publish(Event.Registered, { clientID, toolIDs: registered })

    return registered
  }

  /**
   * Unregister tools for a client.
   * If toolIDs is not provided, unregisters all tools for the client.
   */
  export function unregister(clientID: string, toolIDs?: string[]): string[] {
    const clientTools = registry.get(clientID)
    if (!clientTools) return []

    const unregistered: string[] = []

    if (toolIDs) {
      for (const id of toolIDs) {
        const fullID = id.startsWith("client_") ? id : prefixToolID(clientID, id)
        if (clientTools.delete(fullID)) {
          unregistered.push(fullID)
        }
      }
    } else {
      unregistered.push(...clientTools.keys())
      registry.delete(clientID)
    }

    if (unregistered.length > 0) {
      log.info("unregistered tools", { clientID, toolIDs: unregistered })
      Bus.publish(Event.Unregistered, { clientID, toolIDs: unregistered })
    }

    return unregistered
  }

  /**
   * Get all tools registered by a specific client.
   */
  export function getTools(clientID: string): ClientToolDefinition[] {
    const clientTools = registry.get(clientID)
    if (!clientTools) return []
    return Array.from(clientTools.values())
  }

  /**
   * Get all client tools across all clients.
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
   * Check if a tool ID belongs to a client tool.
   */
  export function isClientTool(toolID: string): boolean {
    return toolID.startsWith("client_")
  }

  /**
   * Find which client owns a tool.
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
   * Get a specific tool definition.
   */
  export function getTool(toolID: string): ClientToolDefinition | undefined {
    for (const [_, clientTools] of registry) {
      const tool = clientTools.get(toolID)
      if (tool) return tool
    }
    return undefined
  }

  /**
   * Execute a client tool.
   * Sends request to client and waits for response.
   */
  export async function execute(
    clientID: string,
    request: Omit<ClientToolExecutionRequest, "type">,
    timeoutMs: number = 30000,
  ): Promise<ClientToolResult> {
    const fullRequest: ClientToolExecutionRequest = {
      type: "client-tool-request",
      ...request,
    }

    log.info("executing client tool", {
      clientID,
      tool: request.tool,
      requestID: request.requestID,
    })

    // Emit executing event
    Bus.publish(Event.Executing, {
      sessionID: request.sessionID,
      messageID: request.messageID,
      callID: request.callID,
      tool: request.tool,
      clientID,
    })

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        pendingRequests.delete(request.requestID)
        const error = new Error(`Client tool execution timed out after ${timeoutMs}ms`)

        // Emit failed event
        Bus.publish(Event.Failed, {
          sessionID: request.sessionID,
          messageID: request.messageID,
          callID: request.callID,
          tool: request.tool,
          clientID,
          error: error.message,
        })

        reject(error)
      }, timeoutMs)

      pendingRequests.set(request.requestID, {
        request: fullRequest,
        clientID,
        resolve: (result) => {
          // Emit completed event
          Bus.publish(Event.Completed, {
            sessionID: request.sessionID,
            messageID: request.messageID,
            callID: request.callID,
            tool: request.tool,
            clientID,
            success: true,
          })
          resolve(result)
        },
        reject: (error) => {
          // Emit failed event
          Bus.publish(Event.Failed, {
            sessionID: request.sessionID,
            messageID: request.messageID,
            callID: request.callID,
            tool: request.tool,
            clientID,
            error: error.message,
          })
          reject(error)
        },
        timeout,
      })

      // Emit event for client to receive via SSE
      Bus.publish(Event.ToolRequest, {
        clientID,
        request: fullRequest,
      })
    })
  }

  /**
   * Submit result from client.
   * Returns true if the request was found and resolved, false otherwise.
   */
  export function submitResult(requestID: string, result: ClientToolResponse): boolean {
    const pending = pendingRequests.get(requestID)
    if (!pending) {
      log.warn("unknown request ID", { requestID })
      return false
    }

    clearTimeout(pending.timeout)
    pendingRequests.delete(requestID)

    log.info("received result", {
      requestID,
      status: result.status,
    })

    if (result.status === "error") {
      pending.reject(new Error(result.error))
    } else {
      pending.resolve(result)
    }

    return true
  }

  /**
   * Get pending request for a specific request ID.
   */
  export function getPendingRequest(requestID: string): ClientToolExecutionRequest | undefined {
    return pendingRequests.get(requestID)?.request
  }

  /**
   * Clean up all tools and pending requests for a client (on disconnect).
   */
  export function cleanup(clientID: string): void {
    log.info("cleaning up client", { clientID })

    // Cancel all pending requests for this client
    for (const [requestID, pending] of pendingRequests) {
      if (pending.clientID === clientID) {
        clearTimeout(pending.timeout)
        pending.reject(new Error("Client disconnected"))
        pendingRequests.delete(requestID)
      }
    }

    // Remove all tools
    const tools = registry.get(clientID)
    if (tools) {
      const toolIDs = Array.from(tools.keys())
      registry.delete(clientID)
      if (toolIDs.length > 0) {
        Bus.publish(Event.Unregistered, { clientID, toolIDs })
      }
    }
  }

  /**
   * Get list of all registered client IDs.
   */
  export function getClientIDs(): string[] {
    return Array.from(registry.keys())
  }

  /**
   * Check if a client has any registered tools.
   */
  export function hasTools(clientID: string): boolean {
    const tools = registry.get(clientID)
    return tools !== undefined && tools.size > 0
  }

  // ============================================================================
  // Helper Functions
  // ============================================================================

  function prefixToolID(clientID: string, toolID: string): string {
    return `client_${clientID}_${toolID}`
  }

  /**
   * Extract the original tool ID from a prefixed tool ID.
   */
  export function extractOriginalToolID(prefixedToolID: string, clientID: string): string {
    const prefix = `client_${clientID}_`
    if (prefixedToolID.startsWith(prefix)) {
      return prefixedToolID.slice(prefix.length)
    }
    return prefixedToolID
  }

  // ============================================================================
  // Testing Utilities
  // ============================================================================

  /**
   * Reset the registry state. Only for testing.
   */
  export function _reset(): void {
    registry.clear()
    for (const [requestID, pending] of pendingRequests) {
      clearTimeout(pending.timeout)
    }
    pendingRequests.clear()
  }
}
