import z from "zod"
import { Log } from "../../../util/log"
import { PendingToolCall } from "../pending-tool"
import type { StorageBackend } from "./backend"

/**
 * MySQL Storage Backend for Pending Tool Calls
 *
 * This module provides MySQL-based persistence for async tool call state,
 * offering better scalability and query capabilities compared to file storage.
 *
 * Required table schema:
 *
 * ```sql
 * CREATE TABLE pending_tool_calls (
 *   id VARCHAR(64) PRIMARY KEY,
 *   session_id VARCHAR(64) NOT NULL,
 *   message_id VARCHAR(64) NOT NULL,
 *   part_id VARCHAR(64) NOT NULL,
 *   call_id VARCHAR(64) NOT NULL,
 *   tool VARCHAR(128) NOT NULL,
 *   input JSON NOT NULL,
 *   status ENUM('waiting', 'processing', 'completed', 'failed', 'expired', 'cancelled') NOT NULL DEFAULT 'waiting',
 *   webhook_url VARCHAR(512),
 *   webhook_secret VARCHAR(256),
 *   external_ref VARCHAR(256),
 *   timeout_at BIGINT,
 *   created_at BIGINT NOT NULL,
 *   started_at BIGINT,
 *   completed_at BIGINT,
 *   result JSON,
 *   error TEXT,
 *   INDEX idx_session (session_id),
 *   INDEX idx_status (status),
 *   INDEX idx_timeout (timeout_at),
 *   INDEX idx_created (created_at)
 * ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
 * ```
 */
export namespace MySQLStorage {
  const log = Log.create({ service: "pending-tool-mysql" })

  // ============================================================================
  // Configuration
  // ============================================================================

  export interface Config {
    host: string
    port: number
    user: string
    password: string
    database: string
    connectionLimit?: number
  }

  export const ConfigSchema = z.object({
    host: z.string().default("localhost"),
    port: z.number().default(3306),
    user: z.string(),
    password: z.string(),
    database: z.string(),
    connectionLimit: z.number().default(10),
  })

  // ============================================================================
  // Connection Pool Types
  // ============================================================================

  // Note: This is a placeholder - actual implementation would use mysql2 or similar
  type Connection = {
    query<T>(sql: string, params?: any[]): Promise<T[]>
    execute(sql: string, params?: any[]): Promise<{ affectedRows: number }>
    release(): void
  }

  type Pool = {
    getConnection(): Promise<Connection>
    end(): Promise<void>
  }

  // ============================================================================
  // Helper Functions
  // ============================================================================

  /**
   * Convert database row to PendingToolCall.Info
   */
  function rowToInfo(row: any): PendingToolCall.Info {
    return {
      id: row.id,
      sessionID: row.session_id,
      messageID: row.message_id,
      partID: row.part_id,
      callID: row.call_id,
      tool: row.tool,
      input: typeof row.input === "string" ? JSON.parse(row.input) : row.input,
      status: row.status,
      webhookURL: row.webhook_url || undefined,
      webhookSecret: row.webhook_secret || undefined,
      externalRef: row.external_ref || undefined,
      timeout: row.timeout_at || undefined,
      time: {
        created: row.created_at,
        started: row.started_at || undefined,
        completed: row.completed_at || undefined,
      },
      result: row.result
        ? typeof row.result === "string"
          ? JSON.parse(row.result)
          : row.result
        : undefined,
      error: row.error || undefined,
    }
  }

  // ============================================================================
  // Backend Factory
  // ============================================================================

  /**
   * Create a MySQL storage backend
   *
   * Usage:
   * ```typescript
   * const backend = MySQLStorage.createBackend({
   *   host: 'localhost',
   *   port: 3306,
   *   user: 'root',
   *   password: 'secret',
   *   database: 'opencode'
   * })
   * await backend.init()
   * ```
   */
  export function createBackend(config: Config): StorageBackend {
    let pool: Pool | null = null

    return {
      async init(): Promise<void> {
        // In real implementation, use mysql2:
        // import mysql from 'mysql2/promise'
        // pool = mysql.createPool(config)

        log.info("MySQL storage initialized", {
          host: config.host,
          database: config.database,
        })

        // Verify connection and create table if needed
        if (pool) {
          const conn = await pool.getConnection()
          try {
            await conn.execute(`
              CREATE TABLE IF NOT EXISTS pending_tool_calls (
                id VARCHAR(64) PRIMARY KEY,
                session_id VARCHAR(64) NOT NULL,
                message_id VARCHAR(64) NOT NULL,
                part_id VARCHAR(64) NOT NULL,
                call_id VARCHAR(64) NOT NULL,
                tool VARCHAR(128) NOT NULL,
                input JSON NOT NULL,
                status ENUM('waiting', 'processing', 'completed', 'failed', 'expired', 'cancelled') NOT NULL DEFAULT 'waiting',
                webhook_url VARCHAR(512),
                webhook_secret VARCHAR(256),
                external_ref VARCHAR(256),
                timeout_at BIGINT,
                created_at BIGINT NOT NULL,
                started_at BIGINT,
                completed_at BIGINT,
                result JSON,
                error TEXT,
                INDEX idx_session (session_id),
                INDEX idx_status (status),
                INDEX idx_timeout (timeout_at),
                INDEX idx_created (created_at)
              ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
            `)
            log.info("Table pending_tool_calls ensured")
          } finally {
            conn.release()
          }
        }
      },

      async close(): Promise<void> {
        if (pool) {
          await pool.end()
          pool = null
          log.info("MySQL connection pool closed")
        }
      },

      async create(pending: PendingToolCall.Info): Promise<void> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          await conn.execute(
            `INSERT INTO pending_tool_calls
             (id, session_id, message_id, part_id, call_id, tool, input, status,
              webhook_url, webhook_secret, external_ref, timeout_at, created_at,
              started_at, completed_at, result, error)
             VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
            [
              pending.id,
              pending.sessionID,
              pending.messageID,
              pending.partID,
              pending.callID,
              pending.tool,
              JSON.stringify(pending.input),
              pending.status,
              pending.webhookURL || null,
              pending.webhookSecret || null,
              pending.externalRef || null,
              pending.timeout || null,
              pending.time.created,
              pending.time.started || null,
              pending.time.completed || null,
              pending.result ? JSON.stringify(pending.result) : null,
              pending.error || null,
            ],
          )
          log.info("created pending tool call", { id: pending.id })
        } finally {
          conn.release()
        }
      },

      async get(id: string): Promise<PendingToolCall.Info | undefined> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          const rows = await conn.query<any>(
            `SELECT * FROM pending_tool_calls WHERE id = ?`,
            [id],
          )

          if (rows.length === 0) return undefined
          return rowToInfo(rows[0])
        } finally {
          conn.release()
        }
      },

      async update(
        id: string,
        updates: Partial<Omit<PendingToolCall.Info, "id">>,
      ): Promise<void> {
        if (!pool) throw new Error("MySQL not initialized")

        const setClauses: string[] = []
        const params: any[] = []

        if (updates.status !== undefined) {
          setClauses.push("status = ?")
          params.push(updates.status)
        }
        if (updates.externalRef !== undefined) {
          setClauses.push("external_ref = ?")
          params.push(updates.externalRef)
        }
        if (updates.timeout !== undefined) {
          setClauses.push("timeout_at = ?")
          params.push(updates.timeout)
        }
        if (updates.time?.started !== undefined) {
          setClauses.push("started_at = ?")
          params.push(updates.time.started)
        }
        if (updates.time?.completed !== undefined) {
          setClauses.push("completed_at = ?")
          params.push(updates.time.completed)
        }
        if (updates.result !== undefined) {
          setClauses.push("result = ?")
          params.push(JSON.stringify(updates.result))
        }
        if (updates.error !== undefined) {
          setClauses.push("error = ?")
          params.push(updates.error)
        }

        if (setClauses.length === 0) return

        params.push(id)

        const conn = await pool.getConnection()
        try {
          await conn.execute(
            `UPDATE pending_tool_calls SET ${setClauses.join(", ")} WHERE id = ?`,
            params,
          )
          log.info("updated pending tool call", { id })
        } finally {
          conn.release()
        }
      },

      async remove(id: string): Promise<void> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          await conn.execute(`DELETE FROM pending_tool_calls WHERE id = ?`, [id])
          log.info("removed pending tool call", { id })
        } finally {
          conn.release()
        }
      },

      async listBySession(sessionID: string): Promise<PendingToolCall.Info[]> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          const rows = await conn.query<any>(
            `SELECT * FROM pending_tool_calls WHERE session_id = ? ORDER BY created_at`,
            [sessionID],
          )
          return rows.map(rowToInfo)
        } finally {
          conn.release()
        }
      },

      async listByStatus(status: PendingToolCall.Status): Promise<PendingToolCall.Info[]> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          const rows = await conn.query<any>(
            `SELECT * FROM pending_tool_calls WHERE status = ? ORDER BY created_at`,
            [status],
          )
          return rows.map(rowToInfo)
        } finally {
          conn.release()
        }
      },

      async listAll(): Promise<PendingToolCall.Info[]> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          const rows = await conn.query<any>(
            `SELECT * FROM pending_tool_calls ORDER BY created_at`,
          )
          return rows.map(rowToInfo)
        } finally {
          conn.release()
        }
      },

      async findExpired(): Promise<PendingToolCall.Info[]> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          const now = Date.now()
          const rows = await conn.query<any>(
            `SELECT * FROM pending_tool_calls
             WHERE status = 'waiting'
             AND timeout_at IS NOT NULL
             AND timeout_at <= ?
             ORDER BY timeout_at`,
            [now],
          )
          return rows.map(rowToInfo)
        } finally {
          conn.release()
        }
      },

      async hasWaiting(sessionID: string): Promise<boolean> {
        if (!pool) throw new Error("MySQL not initialized")

        const conn = await pool.getConnection()
        try {
          const rows = await conn.query<any>(
            `SELECT 1 FROM pending_tool_calls
             WHERE session_id = ?
             AND status IN ('waiting', 'processing')
             LIMIT 1`,
            [sessionID],
          )
          return rows.length > 0
        } finally {
          conn.release()
        }
      },

      async cleanup(retentionMs: number): Promise<number> {
        if (!pool) throw new Error("MySQL not initialized")

        const cutoff = Date.now() - retentionMs
        const conn = await pool.getConnection()
        try {
          const result = await conn.execute(
            `DELETE FROM pending_tool_calls
             WHERE status IN ('completed', 'failed', 'expired', 'cancelled')
             AND completed_at IS NOT NULL
             AND completed_at < ?`,
            [cutoff],
          )
          log.info("cleaned up old records", { count: result.affectedRows })
          return result.affectedRows
        } finally {
          conn.release()
        }
      },
    }
  }

  // ============================================================================
  // Statistics (optional utility function)
  // ============================================================================

  /**
   * Get statistics from a MySQL backend
   */
  export async function getStats(backend: StorageBackend): Promise<{
    total: number
    byStatus: Record<PendingToolCall.Status, number>
    oldestWaiting: number | null
  }> {
    // This uses listAll and processes in-memory
    // A real implementation would use SQL aggregation
    const all = await backend.listAll()

    const byStatus: Record<string, number> = {
      waiting: 0,
      processing: 0,
      completed: 0,
      failed: 0,
      expired: 0,
      cancelled: 0,
    }

    let oldestWaiting: number | null = null

    for (const item of all) {
      byStatus[item.status] = (byStatus[item.status] || 0) + 1
      if (item.status === "waiting") {
        if (oldestWaiting === null || item.time.created < oldestWaiting) {
          oldestWaiting = item.time.created
        }
      }
    }

    return {
      total: all.length,
      byStatus: byStatus as Record<PendingToolCall.Status, number>,
      oldestWaiting,
    }
  }
}
