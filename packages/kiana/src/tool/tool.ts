import { z } from "zod"
import type { FilePart } from "../types/part.js"

/**
 * Tool execution context
 */
export interface ToolContext {
  sessionID: string
  messageID: string
  abort: AbortSignal
  callID?: string
  workingDirectory: string
  metadata(input: { title?: string; metadata?: Record<string, unknown> }): void
}

/**
 * Tool execution result
 */
export interface ToolResult {
  title: string
  output: string
  metadata: Record<string, unknown>
  attachments?: FilePart[]
}

/**
 * Tool definition
 */
export interface Tool<P extends z.ZodType = z.ZodType> {
  id: string
  description: string
  parameters: P
  execute(args: z.infer<P>, ctx: ToolContext): Promise<ToolResult>
}

/**
 * Define a tool with validation
 */
export function defineTool<P extends z.ZodType>(
  id: string,
  config: {
    description: string
    parameters: P
    execute(args: z.infer<P>, ctx: ToolContext): Promise<ToolResult>
  }
): Tool<P> {
  return {
    id,
    description: config.description,
    parameters: config.parameters,
    execute: async (args, ctx) => {
      // Validate parameters
      const result = config.parameters.safeParse(args)
      if (!result.success) {
        const errors = result.error.issues
          .map((e: z.ZodIssue) => `${e.path.join(".")}: ${e.message}`)
          .join(", ")
        throw new Error(
          `The ${id} tool was called with invalid arguments: ${errors}. Please fix the input.`
        )
      }
      return config.execute(result.data, ctx)
    },
  }
}
