import { z } from "zod"
import type { Session } from "../session"

/**
 * Tool namespace - defines the structure for memsh-cli tools
 * Mirrors the structure from packages/opencode/src/tool/tool.ts
 */
export namespace Tool {
  interface Metadata {
    [key: string]: unknown
  }

  /**
   * Context provided to tool execution
   */
  export type Context<M extends Metadata = Metadata> = {
    /** Session for executing commands */
    session: Session
    /** Abort signal for cancellation */
    abort: AbortSignal
    /** Optional call ID for tracking */
    callID?: string
    /** Extra context data */
    extra?: Record<string, unknown>
    /** Update metadata during execution */
    metadata(input: { title?: string; metadata?: M }): void
  }

  /**
   * Tool execution result
   */
  export interface Result<M extends Metadata = Metadata> {
    /** Title for display */
    title: string
    /** Metadata for the result */
    metadata: M
    /** Output string */
    output: string
  }

  /**
   * Tool definition interface
   */
  export interface Info<Parameters extends z.ZodType = z.ZodType, M extends Metadata = Metadata> {
    /** Unique tool identifier */
    id: string
    /** Initialize the tool */
    init: () => Promise<{
      /** Tool description */
      description: string
      /** Parameter schema */
      parameters: Parameters
      /** Execute the tool */
      execute(args: z.infer<Parameters>, ctx: Context<M>): Promise<Result<M>>
      /** Format validation errors */
      formatValidationError?(error: z.ZodError): string
    }>
  }

  /**
   * Infer parameter types from tool info
   */
  export type InferParameters<T extends Info> = T extends Info<infer P> ? z.infer<P> : never

  /**
   * Infer metadata types from tool info
   */
  export type InferMetadata<T extends Info> = T extends Info<z.ZodType, infer M> ? M : never

  /**
   * Define a new tool
   */
  export function define<Parameters extends z.ZodType, Result extends Metadata>(
    id: string,
    init: Info<Parameters, Result>["init"] | Awaited<ReturnType<Info<Parameters, Result>["init"]>>,
  ): Info<Parameters, Result> {
    return {
      id,
      init: async () => {
        const toolInfo = init instanceof Function ? await init() : init
        const execute = toolInfo.execute

        // Wrap execute to validate parameters
        toolInfo.execute = (args, ctx) => {
          try {
            toolInfo.parameters.parse(args)
          } catch (error) {
            if (error instanceof z.ZodError && toolInfo.formatValidationError) {
              throw new Error(toolInfo.formatValidationError(error), { cause: error })
            }
            throw new Error(
              `The ${id} tool was called with invalid arguments: ${error}.\nPlease rewrite the input so it satisfies the expected schema.`,
              { cause: error },
            )
          }
          return execute(args, ctx)
        }

        return toolInfo
      },
    }
  }
}

/**
 * Tool registry for managing available tools
 */
export class ToolRegistry {
  private tools: Map<string, Tool.Info> = new Map()
  private initialized: Map<string, Awaited<ReturnType<Tool.Info["init"]>>> = new Map()

  /**
   * Register a tool
   */
  register(tool: Tool.Info): void {
    this.tools.set(tool.id, tool)
  }

  /**
   * Register multiple tools
   */
  registerAll(...tools: Tool.Info[]): void {
    for (const tool of tools) {
      this.register(tool)
    }
  }

  /**
   * Get a tool by ID
   */
  get(id: string): Tool.Info | undefined {
    return this.tools.get(id)
  }

  /**
   * Get an initialized tool
   */
  async getInitialized(id: string): Promise<Awaited<ReturnType<Tool.Info["init"]>> | undefined> {
    if (this.initialized.has(id)) {
      return this.initialized.get(id)
    }

    const tool = this.tools.get(id)
    if (!tool) {
      return undefined
    }

    const init = await tool.init()
    this.initialized.set(id, init)
    return init
  }

  /**
   * List all registered tool IDs
   */
  list(): string[] {
    return Array.from(this.tools.keys())
  }

  /**
   * Check if a tool is registered
   */
  has(id: string): boolean {
    return this.tools.has(id)
  }

  /**
   * Execute a tool
   */
  async execute<T extends Tool.Info>(
    id: string,
    args: Tool.InferParameters<T>,
    ctx: Tool.Context<Tool.InferMetadata<T>>,
  ): Promise<Tool.Result<Tool.InferMetadata<T>>> {
    const tool = await this.getInitialized(id)
    if (!tool) {
      throw new Error(`Tool not found: ${id}`)
    }

    return tool.execute(args, ctx) as Promise<Tool.Result<Tool.InferMetadata<T>>>
  }
}

/**
 * Default tool registry
 */
export const registry = new ToolRegistry()
