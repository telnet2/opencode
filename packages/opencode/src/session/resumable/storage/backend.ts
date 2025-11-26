import { PendingToolCall } from "../pending-tool"

/**
 * Storage Backend Interface
 *
 * Abstraction layer for persistent storage of pending tool calls.
 * Implementations can use file-based storage (default), MySQL, or other backends.
 */
export interface StorageBackend {
  /**
   * Initialize the storage backend
   */
  init(): Promise<void>

  /**
   * Close/cleanup the storage backend
   */
  close(): Promise<void>

  /**
   * Create a new pending tool call
   */
  create(pending: PendingToolCall.Info): Promise<void>

  /**
   * Get a pending tool call by ID
   */
  get(id: string): Promise<PendingToolCall.Info | undefined>

  /**
   * Update a pending tool call
   */
  update(id: string, updates: Partial<Omit<PendingToolCall.Info, "id">>): Promise<void>

  /**
   * Delete a pending tool call
   */
  remove(id: string): Promise<void>

  /**
   * List pending tool calls by session ID
   */
  listBySession(sessionID: string): Promise<PendingToolCall.Info[]>

  /**
   * List pending tool calls by status
   */
  listByStatus(status: PendingToolCall.Status): Promise<PendingToolCall.Info[]>

  /**
   * List all pending tool calls
   */
  listAll(): Promise<PendingToolCall.Info[]>

  /**
   * Find expired pending tool calls
   */
  findExpired(): Promise<PendingToolCall.Info[]>

  /**
   * Check if session has waiting calls
   */
  hasWaiting(sessionID: string): Promise<boolean>

  /**
   * Clean up old records
   */
  cleanup(retentionMs: number): Promise<number>
}

/**
 * Storage backend type
 */
export type StorageBackendType = "file" | "mysql"

/**
 * Storage configuration
 */
export interface StorageConfig {
  type: StorageBackendType
  mysql?: {
    host: string
    port: number
    user: string
    password: string
    database: string
    connectionLimit?: number
  }
}
