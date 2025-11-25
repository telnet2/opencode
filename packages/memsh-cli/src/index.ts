/**
 * @opencode-ai/memsh-cli
 *
 * TypeScript client for connecting to go-memsh service and providing
 * the same tool features as packages/opencode for working over
 * the memory file system.
 */

// Client exports
export { MemshClient, createClient } from "./client"
export type {
  MemshClientOptions,
  ConnectionState,
  JSONRPCRequest,
  JSONRPCResponse,
  JSONRPCError,
  SessionInfo,
  CreateSessionResponse,
  ListSessionsResponse,
  RemoveSessionRequest,
  RemoveSessionResponse,
  ExecuteCommandParams,
  ExecuteCommandResult,
} from "./client"

// Session exports
export { Session, createSession, type SessionOptions } from "./session"

// Tool exports
export { Tool, ToolRegistry, registry, registerDefaultTools, allTools } from "./tool"
export { BashTool } from "./tool/bash"
export { ReadTool } from "./tool/read"
export { WriteTool } from "./tool/write"
export { EditTool } from "./tool/edit"
export { GlobTool } from "./tool/glob"
export { GrepTool } from "./tool/grep"
export { LsTool } from "./tool/ls"

// Convenience function to create a fully configured client with session
import { createSession, type SessionOptions } from "./session"
import { registerDefaultTools, registry, Tool } from "./tool"

/**
 * Options for creating a MemshEnvironment
 */
export interface MemshEnvironmentOptions extends SessionOptions {}

/**
 * A fully configured environment for working with memsh
 */
export interface MemshEnvironment {
  /** The active session */
  session: Awaited<ReturnType<typeof createSession>>
  /** Execute a tool by name */
  executeTool<T extends Tool.Info>(
    toolId: string,
    args: Tool.InferParameters<T>,
    options?: { abort?: AbortSignal },
  ): Promise<Tool.Result<Tool.InferMetadata<T>>>
  /** Close the environment */
  close(removeSession?: boolean): Promise<void>
}

/**
 * Create a fully configured memsh environment
 *
 * @example
 * ```ts
 * const env = await createMemshEnvironment({ baseUrl: 'http://localhost:8080' })
 *
 * // Execute commands
 * const result = await env.session.run('ls -la')
 *
 * // Use tools
 * const files = await env.executeTool('glob', { pattern: '*.ts' })
 *
 * // Clean up
 * await env.close()
 * ```
 */
export async function createMemshEnvironment(options: MemshEnvironmentOptions): Promise<MemshEnvironment> {
  // Register default tools
  registerDefaultTools()

  // Create session
  const session = await createSession(options)

  return {
    session,

    async executeTool<T extends Tool.Info>(
      toolId: string,
      args: Tool.InferParameters<T>,
      execOptions?: { abort?: AbortSignal },
    ): Promise<Tool.Result<Tool.InferMetadata<T>>> {
      const ctx: Tool.Context = {
        session,
        abort: execOptions?.abort ?? new AbortController().signal,
        metadata: () => {
          /* no-op for simple usage */
        },
      }

      return registry.execute<T>(toolId, args, ctx as Tool.Context<Tool.InferMetadata<T>>)
    },

    async close(removeSession = false): Promise<void> {
      await session.close(removeSession)
    },
  }
}
