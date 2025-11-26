/**
 * Storage Module for Resumable Sessions
 *
 * This module provides a pluggable storage backend for pending tool calls.
 * The default is file-based storage, but MySQL can be configured for better
 * scalability in production environments.
 *
 * Usage:
 *
 * ```typescript
 * import { PendingToolStorage } from './storage'
 *
 * // Use default file-based storage
 * const storage = PendingToolStorage.create()
 * await storage.init()
 *
 * // Or use MySQL storage
 * const mysqlStorage = PendingToolStorage.create({
 *   type: 'mysql',
 *   mysql: {
 *     host: 'localhost',
 *     port: 3306,
 *     user: 'root',
 *     password: 'secret',
 *     database: 'opencode'
 *   }
 * })
 * await mysqlStorage.init()
 * ```
 */

export type { StorageBackend, StorageConfig, StorageBackendType } from "./backend"
export { FileStorage } from "./file"
export { MySQLStorage } from "./mysql"

import type { StorageBackend, StorageConfig } from "./backend"
import { FileStorage } from "./file"
import { MySQLStorage } from "./mysql"

/**
 * Pending Tool Storage Factory
 *
 * Creates the appropriate storage backend based on configuration.
 */
export namespace PendingToolStorage {
  let currentBackend: StorageBackend | null = null

  /**
   * Create a storage backend based on configuration
   */
  export function create(config?: StorageConfig): StorageBackend {
    if (!config || config.type === "file") {
      return FileStorage.createBackend()
    }

    if (config.type === "mysql") {
      if (!config.mysql) {
        throw new Error("MySQL configuration required when type is 'mysql'")
      }
      return MySQLStorage.createBackend(config.mysql)
    }

    throw new Error(`Unknown storage backend type: ${config.type}`)
  }

  /**
   * Initialize and set the global storage backend
   */
  export async function init(config?: StorageConfig): Promise<StorageBackend> {
    currentBackend = create(config)
    await currentBackend.init()
    return currentBackend
  }

  /**
   * Get the current storage backend
   * Throws if not initialized
   */
  export function get(): StorageBackend {
    if (!currentBackend) {
      throw new Error("Storage not initialized. Call PendingToolStorage.init() first.")
    }
    return currentBackend
  }

  /**
   * Get the current storage backend or undefined if not initialized
   */
  export function getOrUndefined(): StorageBackend | undefined {
    return currentBackend ?? undefined
  }

  /**
   * Close the current storage backend
   */
  export async function close(): Promise<void> {
    if (currentBackend) {
      await currentBackend.close()
      currentBackend = null
    }
  }

  /**
   * Check if storage is initialized
   */
  export function isInitialized(): boolean {
    return currentBackend !== null
  }
}
