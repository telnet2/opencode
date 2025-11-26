import { Storage } from "../../../storage/storage"
import { Log } from "../../../util/log"
import { PendingToolCall } from "../pending-tool"
import type { StorageBackend } from "./backend"

/**
 * File-based Storage Backend
 *
 * Uses the existing Storage module to persist pending tool calls as JSON files.
 * This is the default storage backend.
 */
export namespace FileStorage {
  const log = Log.create({ service: "pending-tool-file" })

  const STORAGE_PREFIX = "pending_tool"
  const INDEX_KEY = [STORAGE_PREFIX, "_index"]

  function key(id: string): string[] {
    return [STORAGE_PREFIX, id]
  }

  // ============================================================================
  // Index Management
  // ============================================================================

  async function loadIndex(): Promise<PendingToolCall.Index> {
    try {
      return await Storage.read<PendingToolCall.Index>(INDEX_KEY)
    } catch (e) {
      if (e instanceof Storage.NotFoundError) {
        return {
          bySession: {},
          byStatus: {
            waiting: [],
            processing: [],
            completed: [],
            failed: [],
            expired: [],
            cancelled: [],
          },
          byExpiration: [],
          lastUpdated: Date.now(),
        }
      }
      throw e
    }
  }

  async function saveIndex(index: PendingToolCall.Index): Promise<void> {
    index.lastUpdated = Date.now()
    await Storage.write(INDEX_KEY, index)
  }

  // ============================================================================
  // Backend Implementation
  // ============================================================================

  export function createBackend(): StorageBackend {
    return {
      async init(): Promise<void> {
        log.info("File storage initialized")
      },

      async close(): Promise<void> {
        log.info("File storage closed")
      },

      async create(pending: PendingToolCall.Info): Promise<void> {
        await Storage.write(key(pending.id), pending)

        // Update index
        const index = await loadIndex()

        // Add to session index
        if (!index.bySession[pending.sessionID]) {
          index.bySession[pending.sessionID] = []
        }
        if (!index.bySession[pending.sessionID].includes(pending.id)) {
          index.bySession[pending.sessionID].push(pending.id)
        }

        // Add to status index
        if (!index.byStatus[pending.status]) {
          index.byStatus[pending.status] = []
        }
        if (!index.byStatus[pending.status].includes(pending.id)) {
          index.byStatus[pending.status].push(pending.id)
        }

        // Add to expiration index if has timeout
        if (pending.timeout) {
          const existing = index.byExpiration.findIndex((e) => e.id === pending.id)
          if (existing >= 0) {
            index.byExpiration[existing].expiresAt = pending.timeout
          } else {
            index.byExpiration.push({ id: pending.id, expiresAt: pending.timeout })
          }
          index.byExpiration.sort((a, b) => a.expiresAt - b.expiresAt)
        }

        await saveIndex(index)
        log.info("created pending tool call", { id: pending.id })
      },

      async get(id: string): Promise<PendingToolCall.Info | undefined> {
        try {
          return await Storage.read<PendingToolCall.Info>(key(id))
        } catch (e) {
          if (e instanceof Storage.NotFoundError) {
            return undefined
          }
          throw e
        }
      },

      async update(id: string, updates: Partial<Omit<PendingToolCall.Info, "id">>): Promise<void> {
        const existing = await this.get(id)
        if (!existing) {
          throw new Error(`Pending tool call not found: ${id}`)
        }

        const previousStatus = existing.status
        const updated: PendingToolCall.Info = {
          ...existing,
          ...updates,
        }

        await Storage.write(key(id), updated)

        // Update index if status changed
        if (updates.status && updates.status !== previousStatus) {
          const index = await loadIndex()

          // Remove from old status
          if (index.byStatus[previousStatus]) {
            index.byStatus[previousStatus] = index.byStatus[previousStatus].filter((i) => i !== id)
          }

          // Add to new status
          if (!index.byStatus[updates.status]) {
            index.byStatus[updates.status] = []
          }
          if (!index.byStatus[updates.status].includes(id)) {
            index.byStatus[updates.status].push(id)
          }

          await saveIndex(index)
        }

        log.info("updated pending tool call", { id })
      },

      async remove(id: string): Promise<void> {
        const existing = await this.get(id)
        if (existing) {
          const index = await loadIndex()

          // Remove from session index
          if (index.bySession[existing.sessionID]) {
            index.bySession[existing.sessionID] = index.bySession[existing.sessionID].filter(
              (i) => i !== id,
            )
            if (index.bySession[existing.sessionID].length === 0) {
              delete index.bySession[existing.sessionID]
            }
          }

          // Remove from status index
          if (index.byStatus[existing.status]) {
            index.byStatus[existing.status] = index.byStatus[existing.status].filter((i) => i !== id)
          }

          // Remove from expiration index
          index.byExpiration = index.byExpiration.filter((e) => e.id !== id)

          await saveIndex(index)
        }

        await Storage.remove(key(id))
        log.info("removed pending tool call", { id })
      },

      async listBySession(sessionID: string): Promise<PendingToolCall.Info[]> {
        const index = await loadIndex()
        const ids = index.bySession[sessionID] || []
        const results: PendingToolCall.Info[] = []

        for (const id of ids) {
          const pending = await this.get(id)
          if (pending) results.push(pending)
        }

        return results
      },

      async listByStatus(status: PendingToolCall.Status): Promise<PendingToolCall.Info[]> {
        const index = await loadIndex()
        const ids = index.byStatus[status] || []
        const results: PendingToolCall.Info[] = []

        for (const id of ids) {
          const pending = await this.get(id)
          if (pending) results.push(pending)
        }

        return results
      },

      async listAll(): Promise<PendingToolCall.Info[]> {
        const keys = await Storage.list([STORAGE_PREFIX])
        const results: PendingToolCall.Info[] = []

        for (const k of keys) {
          if (k[1] === "_index") continue
          const pending = await this.get(k[1])
          if (pending) results.push(pending)
        }

        return results
      },

      async findExpired(): Promise<PendingToolCall.Info[]> {
        const index = await loadIndex()
        const now = Date.now()
        const expired: PendingToolCall.Info[] = []

        for (const entry of index.byExpiration) {
          if (entry.expiresAt <= now) {
            const pending = await this.get(entry.id)
            if (pending && pending.status === "waiting") {
              expired.push(pending)
            }
          } else {
            // Sorted by expiration, so we can stop here
            break
          }
        }

        return expired
      },

      async hasWaiting(sessionID: string): Promise<boolean> {
        const pending = await this.listBySession(sessionID)
        return pending.some((p) => p.status === "waiting" || p.status === "processing")
      },

      async cleanup(retentionMs: number): Promise<number> {
        const cutoff = Date.now() - retentionMs
        let count = 0

        for (const status of ["completed", "failed", "expired", "cancelled"] as PendingToolCall.Status[]) {
          const pending = await this.listByStatus(status)
          for (const p of pending) {
            if (p.time.completed && p.time.completed < cutoff) {
              await this.remove(p.id)
              count++
            }
          }
        }

        log.info("cleaned up pending tool calls", { count })
        return count
      },
    }
  }
}
