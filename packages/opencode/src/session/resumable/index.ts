/**
 * Resumable Sessions Module
 *
 * This module provides the infrastructure for resumable agent sessions that can
 * survive agent shutdowns during long-running asynchronous tool executions.
 *
 * Key Components:
 * - PendingToolCall: Persistent storage for async tool call state
 * - AsyncToolRegistry: Registry for tools that execute asynchronously
 * - SessionResumer: Handles session resumption when results arrive
 * - Recovery: Startup recovery routines
 * - PendingToolStorage: Pluggable storage backend (file or MySQL)
 *
 * Usage:
 *
 * 1. Register an async tool:
 * ```typescript
 * AsyncToolRegistry.register({
 *   id: 'deploy-preview',
 *   description: 'Deploy a preview environment',
 *   parameters: z.object({ branch: z.string() }),
 *   async execute(input, ctx) {
 *     const job = await startDeployment(input.branch, ctx.pendingID)
 *     return ctx.createAsyncResult(`Deployment ${job.id} started`, job.id)
 *   }
 * })
 * ```
 *
 * 2. Submit results from external system:
 * ```typescript
 * await SessionResumer.submitResult({
 *   pendingID: 'pending_xxx',
 *   result: {
 *     title: 'Deployment Complete',
 *     output: 'Preview: https://preview.example.com',
 *     metadata: { url: '...' }
 *   }
 * })
 * ```
 *
 * 3. Run recovery on startup:
 * ```typescript
 * await Recovery.run()
 * ```
 *
 * 4. Configure MySQL storage (optional):
 * ```typescript
 * await PendingToolStorage.init({
 *   type: 'mysql',
 *   mysql: {
 *     host: 'localhost',
 *     port: 3306,
 *     user: 'root',
 *     password: 'secret',
 *     database: 'opencode'
 *   }
 * })
 * ```
 */

export { PendingToolCall } from "./pending-tool"
export { SessionResumer } from "./session-resumer"
export { AsyncToolRegistry } from "./async-tool-registry"
export { Recovery } from "./recovery"
export { PendingToolStorage } from "./storage"

// Re-export storage types
export type { StorageBackend, StorageConfig, StorageBackendType } from "./storage"

// Re-export key types
export type { PendingToolCall as PendingToolCallTypes } from "./pending-tool"
export type { SessionResumer as SessionResumerTypes } from "./session-resumer"
export type { AsyncToolRegistry as AsyncToolRegistryTypes } from "./async-tool-registry"
