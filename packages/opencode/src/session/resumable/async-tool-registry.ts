import z from "zod"
import { Bus } from "../../bus"
import { Log } from "../../util/log"
import { Identifier } from "../../id/id"
import { Tool } from "../../tool/tool"
import { PendingToolCall } from "./pending-tool"

/**
 * AsyncToolRegistry - Manages async tool definitions and execution
 *
 * Async tools are tools that start a long-running operation and return
 * immediately with a pending reference. The actual result is submitted
 * later via the SessionResumer.
 *
 * This registry wraps regular tools to make them async-capable, handling:
 * 1. Pending state creation
 * 2. Immediate response generation
 * 3. Webhook configuration
 */
export namespace AsyncToolRegistry {
  const log = Log.create({ service: "async-tool-registry" })

  // ============================================================================
  // Types
  // ============================================================================

  /**
   * Result from an async tool execution
   */
  export const AsyncResult = z.object({
    async: z.literal(true),
    pendingID: z.string(),
    externalRef: z.string().optional(),
    estimatedDuration: z.number().optional(),
    output: z.string(),
  })
  export type AsyncResult = z.infer<typeof AsyncResult>

  /**
   * Configuration for async tool execution
   */
  export interface AsyncConfig {
    // Timeout in milliseconds (default: 24 hours)
    timeout?: number
    // Webhook URL for the external system to call back
    webhookURL?: string
    // Secret for webhook signature verification
    webhookSecret?: string
    // External reference (job ID, request ID, etc.)
    externalRef?: string
    // Estimated duration hint
    estimatedDuration?: number
  }

  /**
   * Context passed to async tool execute function
   */
  export interface AsyncContext extends Tool.Context {
    // The pending ID assigned to this async call
    pendingID: string
    // Configuration for this async execution
    asyncConfig: AsyncConfig
    // Mark the tool as started (called automatically)
    markStarted(): Promise<void>
    // Generate the async result to return to the LLM
    createAsyncResult(output: string, externalRef?: string): AsyncResult
  }

  /**
   * Definition of an async tool
   */
  export interface AsyncToolDefinition<P extends z.ZodType = z.ZodType> {
    id: string
    description: string
    parameters: P
    // Default timeout for this tool
    defaultTimeout?: number
    // Execute the tool and return immediately
    execute(input: z.infer<P>, ctx: AsyncContext): Promise<AsyncResult>
    // Optional: validate the result before accepting it
    validateResult?(result: PendingToolCall.Result): boolean
  }

  // ============================================================================
  // Events
  // ============================================================================

  export const Event = {
    Registered: Bus.event(
      "async-tool.registered",
      z.object({
        toolID: z.string(),
      }),
    ),
    Unregistered: Bus.event(
      "async-tool.unregistered",
      z.object({
        toolID: z.string(),
      }),
    ),
    Started: Bus.event(
      "async-tool.started",
      z.object({
        toolID: z.string(),
        pendingID: z.string(),
        sessionID: z.string(),
      }),
    ),
  }

  // ============================================================================
  // Registry State
  // ============================================================================

  const registry = new Map<string, AsyncToolDefinition>()
  const validators = new Map<string, (result: PendingToolCall.Result) => boolean>()

  // ============================================================================
  // Registration
  // ============================================================================

  /**
   * Register an async tool
   */
  export function register<P extends z.ZodType>(definition: AsyncToolDefinition<P>): void {
    log.info("registering async tool", { id: definition.id })
    registry.set(definition.id, definition as AsyncToolDefinition)

    if (definition.validateResult) {
      validators.set(definition.id, definition.validateResult)
    }

    // Only publish if within an instance context (safe for testing)
    try {
      Bus.publish(Event.Registered, { toolID: definition.id })
    } catch {
      // Ignore context errors during testing
    }
  }

  /**
   * Unregister an async tool
   */
  export function unregister(toolID: string): boolean {
    const existed = registry.delete(toolID)
    validators.delete(toolID)

    if (existed) {
      // Only publish if within an instance context (safe for testing)
      try {
        Bus.publish(Event.Unregistered, { toolID })
      } catch {
        // Ignore context errors during testing
      }
      log.info("unregistered async tool", { id: toolID })
    }

    return existed
  }

  /**
   * Check if a tool is registered as async
   */
  export function isAsync(toolID: string): boolean {
    return registry.has(toolID)
  }

  /**
   * Get an async tool definition
   */
  export function get(toolID: string): AsyncToolDefinition | undefined {
    return registry.get(toolID)
  }

  /**
   * List all registered async tools
   */
  export function list(): string[] {
    return Array.from(registry.keys())
  }

  // ============================================================================
  // Execution
  // ============================================================================

  /**
   * Execute an async tool
   *
   * This creates a pending state, calls the tool's execute function,
   * and returns the async result.
   */
  export async function execute(
    toolID: string,
    input: Record<string, any>,
    ctx: Tool.Context,
    config: AsyncConfig = {},
  ): Promise<AsyncResult> {
    const definition = registry.get(toolID)
    if (!definition) {
      throw new Error(`Async tool not found: ${toolID}`)
    }

    // Generate pending ID
    const pendingID = Identifier.ascending("pending")

    // Calculate timeout
    const timeout = config.timeout ?? definition.defaultTimeout ?? 24 * 60 * 60 * 1000

    // Create pending state
    const pending = await PendingToolCall.create({
      sessionID: ctx.sessionID,
      messageID: ctx.messageID,
      partID: ctx.callID || Identifier.ascending("part"),
      callID: ctx.callID || pendingID,
      tool: toolID,
      input,
      webhookURL: config.webhookURL,
      webhookSecret: config.webhookSecret,
      externalRef: config.externalRef,
      timeout,
    })

    log.info("executing async tool", {
      toolID,
      pendingID,
      sessionID: ctx.sessionID,
    })

    // Create async context
    const asyncCtx: AsyncContext = {
      ...ctx,
      pendingID,
      asyncConfig: config,
      async markStarted() {
        await PendingToolCall.markProcessing(pendingID)
      },
      createAsyncResult(output: string, externalRef?: string): AsyncResult {
        return {
          async: true,
          pendingID,
          externalRef: externalRef || config.externalRef,
          estimatedDuration: config.estimatedDuration,
          output,
        }
      },
    }

    // Only publish if within an instance context (safe for testing)
    try {
      Bus.publish(Event.Started, {
        toolID,
        pendingID,
        sessionID: ctx.sessionID,
      })
    } catch {
      // Ignore context errors during testing
    }

    // Execute the tool
    try {
      const result = await definition.execute(input, asyncCtx)

      // Update external ref if returned
      if (result.externalRef && result.externalRef !== pending.externalRef) {
        await PendingToolCall.update(pendingID, { externalRef: result.externalRef })
      }

      return result
    } catch (e) {
      // If execution fails immediately, mark as failed
      await PendingToolCall.fail(pendingID, e instanceof Error ? e.message : String(e))
      throw e
    }
  }

  /**
   * Validate a result for a specific tool
   */
  export function validateResult(toolID: string, result: PendingToolCall.Result): boolean {
    const validator = validators.get(toolID)
    if (!validator) return true
    return validator(result)
  }

  // ============================================================================
  // Tool Wrapper
  // ============================================================================

  /**
   * Create a Tool.Info from an AsyncToolDefinition
   *
   * This wraps an async tool so it can be registered with the regular ToolRegistry
   */
  export function createToolInfo(definition: AsyncToolDefinition): Tool.Info {
    return {
      id: definition.id,
      init: async () => ({
        description: definition.description + " (async - results may take time)",
        parameters: definition.parameters,
        async execute(input, ctx) {
          const result = await execute(definition.id, input, ctx)
          return {
            title: `Async operation started`,
            metadata: {
              async: true,
              pendingID: result.pendingID,
              externalRef: result.externalRef,
              estimatedDuration: result.estimatedDuration,
            },
            output: result.output,
          }
        },
      }),
    }
  }

  // ============================================================================
  // Helper Functions
  // ============================================================================

  /**
   * Create a simple async tool that delegates to an external service
   */
  export function createExternalServiceTool<P extends z.ZodType>(config: {
    id: string
    description: string
    parameters: P
    defaultTimeout?: number
    startJob: (
      input: z.infer<P>,
      webhookURL: string,
      pendingID: string,
    ) => Promise<{ jobID: string; message: string }>
  }): AsyncToolDefinition<P> {
    return {
      id: config.id,
      description: config.description,
      parameters: config.parameters,
      defaultTimeout: config.defaultTimeout,
      async execute(input, ctx) {
        const webhookURL = ctx.asyncConfig.webhookURL || `http://localhost:3000/async-tool/webhook`

        await ctx.markStarted()

        const { jobID, message } = await config.startJob(input, webhookURL, ctx.pendingID)

        return ctx.createAsyncResult(message, jobID)
      },
    }
  }

  // ============================================================================
  // Reset (for testing)
  // ============================================================================

  export function _reset(): void {
    registry.clear()
    validators.clear()
  }
}
