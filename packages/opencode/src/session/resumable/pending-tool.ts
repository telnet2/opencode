import z from "zod"
import { Storage } from "../../storage/storage"
import { Bus } from "../../bus"
import { Log } from "../../util/log"
import { Identifier } from "../../id/id"
import { Lock } from "../../util/lock"

/**
 * PendingToolCall - Persistent storage for async tool calls that survive agent shutdown
 *
 * This module manages the lifecycle of long-running tool calls that may outlive
 * the agent process. When a tool is marked as async, its pending state is persisted
 * to disk so that results can be submitted later, even after agent restart.
 */
export namespace PendingToolCall {
  const log = Log.create({ service: "pending-tool" })

  // ============================================================================
  // Types
  // ============================================================================

  export const Status = z.enum(["waiting", "processing", "completed", "failed", "expired", "cancelled"])
  export type Status = z.infer<typeof Status>

  export const Result = z
    .object({
      title: z.string(),
      output: z.string(),
      metadata: z.record(z.string(), z.any()).optional(),
    })
    .meta({ ref: "PendingToolResult" })
  export type Result = z.infer<typeof Result>

  export const Info = z
    .object({
      id: z.string(),
      sessionID: z.string(),
      messageID: z.string(),
      partID: z.string(),
      callID: z.string(),
      tool: z.string(),
      input: z.record(z.string(), z.any()),
      status: Status,
      webhookURL: z.string().optional(),
      webhookSecret: z.string().optional(),
      externalRef: z.string().optional(),
      timeout: z.number().optional(),
      time: z.object({
        created: z.number(),
        started: z.number().optional(),
        completed: z.number().optional(),
      }),
      result: Result.optional(),
      error: z.string().optional(),
    })
    .meta({ ref: "PendingToolCall" })
  export type Info = z.infer<typeof Info>

  // Index for fast lookups without scanning all files
  export const Index = z.object({
    bySession: z.record(z.string(), z.array(z.string())),
    byStatus: z.record(Status, z.array(z.string())),
    byExpiration: z.array(
      z.object({
        id: z.string(),
        expiresAt: z.number(),
      }),
    ),
    lastUpdated: z.number(),
  })
  export type Index = z.infer<typeof Index>

  // ============================================================================
  // Events
  // ============================================================================

  export const Event = {
    Created: Bus.event(
      "pending-tool.created",
      z.object({
        pending: Info,
      }),
    ),
    Updated: Bus.event(
      "pending-tool.updated",
      z.object({
        pending: Info,
        previousStatus: Status,
      }),
    ),
    Completed: Bus.event(
      "pending-tool.completed",
      z.object({
        pending: Info,
      }),
    ),
    Failed: Bus.event(
      "pending-tool.failed",
      z.object({
        pending: Info,
        error: z.string(),
      }),
    ),
    Expired: Bus.event(
      "pending-tool.expired",
      z.object({
        pending: Info,
      }),
    ),
  }

  // ============================================================================
  // Storage Keys
  // ============================================================================

  const STORAGE_PREFIX = "pending_tool"
  const INDEX_KEY = [STORAGE_PREFIX, "_index"]

  function key(id: string): string[] {
    return [STORAGE_PREFIX, id]
  }

  // ============================================================================
  // Index Management
  // ============================================================================

  async function loadIndex(): Promise<Index> {
    try {
      return await Storage.read<Index>(INDEX_KEY)
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

  async function saveIndex(index: Index): Promise<void> {
    index.lastUpdated = Date.now()
    await Storage.write(INDEX_KEY, index)
  }

  async function addToIndex(pending: Info): Promise<void> {
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
      // Sort by expiration time
      index.byExpiration.sort((a, b) => a.expiresAt - b.expiresAt)
    }

    await saveIndex(index)
  }

  async function updateIndexStatus(id: string, oldStatus: Status, newStatus: Status): Promise<void> {
    const index = await loadIndex()

    // Remove from old status
    if (index.byStatus[oldStatus]) {
      index.byStatus[oldStatus] = index.byStatus[oldStatus].filter((i) => i !== id)
    }

    // Add to new status
    if (!index.byStatus[newStatus]) {
      index.byStatus[newStatus] = []
    }
    if (!index.byStatus[newStatus].includes(id)) {
      index.byStatus[newStatus].push(id)
    }

    await saveIndex(index)
  }

  async function removeFromIndex(pending: Info): Promise<void> {
    const index = await loadIndex()

    // Remove from session index
    if (index.bySession[pending.sessionID]) {
      index.bySession[pending.sessionID] = index.bySession[pending.sessionID].filter((i) => i !== pending.id)
      if (index.bySession[pending.sessionID].length === 0) {
        delete index.bySession[pending.sessionID]
      }
    }

    // Remove from status index
    if (index.byStatus[pending.status]) {
      index.byStatus[pending.status] = index.byStatus[pending.status].filter((i) => i !== pending.id)
    }

    // Remove from expiration index
    index.byExpiration = index.byExpiration.filter((e) => e.id !== pending.id)

    await saveIndex(index)
  }

  // ============================================================================
  // CRUD Operations
  // ============================================================================

  /**
   * Create a new pending tool call
   */
  export async function create(
    input: Omit<Info, "id" | "status" | "time"> & {
      timeout?: number
    },
  ): Promise<Info> {
    const id = Identifier.ascending("pending")
    const now = Date.now()

    const pending: Info = {
      ...input,
      id,
      status: "waiting",
      timeout: input.timeout ? now + input.timeout : undefined,
      time: {
        created: now,
      },
    }

    log.info("creating pending tool call", {
      id,
      sessionID: pending.sessionID,
      tool: pending.tool,
    })

    await Storage.write(key(id), pending)
    await addToIndex(pending)

    Bus.publish(Event.Created, { pending })

    return pending
  }

  /**
   * Get a pending tool call by ID
   */
  export async function get(id: string): Promise<Info | undefined> {
    try {
      return await Storage.read<Info>(key(id))
    } catch (e) {
      if (e instanceof Storage.NotFoundError) {
        return undefined
      }
      throw e
    }
  }

  /**
   * Update a pending tool call
   */
  export async function update(id: string, updates: Partial<Omit<Info, "id">>): Promise<Info> {
    const existing = await get(id)
    if (!existing) {
      throw new Error(`Pending tool call not found: ${id}`)
    }

    const previousStatus = existing.status
    const updated: Info = {
      ...existing,
      ...updates,
    }

    await Storage.write(key(id), updated)

    // Update index if status changed
    if (updates.status && updates.status !== previousStatus) {
      await updateIndexStatus(id, previousStatus, updates.status)
    }

    Bus.publish(Event.Updated, { pending: updated, previousStatus })

    return updated
  }

  /**
   * Mark as processing (external system picked it up)
   */
  export async function markProcessing(id: string): Promise<Info> {
    log.info("marking as processing", { id })
    return update(id, {
      status: "processing",
      time: {
        ...(await get(id))!.time,
        started: Date.now(),
      },
    })
  }

  /**
   * Complete a pending tool call with result
   */
  export async function complete(id: string, result: Result): Promise<Info> {
    log.info("completing pending tool call", { id })

    const existing = await get(id)
    if (!existing) {
      throw new Error(`Pending tool call not found: ${id}`)
    }

    const updated = await update(id, {
      status: "completed",
      result,
      time: {
        ...existing.time,
        completed: Date.now(),
      },
    })

    Bus.publish(Event.Completed, { pending: updated })

    return updated
  }

  /**
   * Mark a pending tool call as failed
   */
  export async function fail(id: string, error: string): Promise<Info> {
    log.info("failing pending tool call", { id, error })

    const existing = await get(id)
    if (!existing) {
      throw new Error(`Pending tool call not found: ${id}`)
    }

    const updated = await update(id, {
      status: "failed",
      error,
      time: {
        ...existing.time,
        completed: Date.now(),
      },
    })

    Bus.publish(Event.Failed, { pending: updated, error })

    return updated
  }

  /**
   * Mark a pending tool call as expired
   */
  export async function expire(id: string): Promise<Info> {
    log.info("expiring pending tool call", { id })

    const existing = await get(id)
    if (!existing) {
      throw new Error(`Pending tool call not found: ${id}`)
    }

    const updated = await update(id, {
      status: "expired",
      error: "Tool execution timed out",
      time: {
        ...existing.time,
        completed: Date.now(),
      },
    })

    Bus.publish(Event.Expired, { pending: updated })

    return updated
  }

  /**
   * Cancel a pending tool call
   */
  export async function cancel(id: string): Promise<Info> {
    log.info("cancelling pending tool call", { id })

    const existing = await get(id)
    if (!existing) {
      throw new Error(`Pending tool call not found: ${id}`)
    }

    return update(id, {
      status: "cancelled",
      error: "Cancelled by user",
      time: {
        ...existing.time,
        completed: Date.now(),
      },
    })
  }

  /**
   * Delete a pending tool call (cleanup)
   */
  export async function remove(id: string): Promise<void> {
    const existing = await get(id)
    if (existing) {
      await removeFromIndex(existing)
    }
    await Storage.remove(key(id))
    log.info("removed pending tool call", { id })
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * List all pending tool calls for a session
   */
  export async function listBySession(sessionID: string): Promise<Info[]> {
    const index = await loadIndex()
    const ids = index.bySession[sessionID] || []
    const results: Info[] = []

    for (const id of ids) {
      const pending = await get(id)
      if (pending) results.push(pending)
    }

    return results
  }

  /**
   * List all pending tool calls with a specific status
   */
  export async function listByStatus(status: Status): Promise<Info[]> {
    const index = await loadIndex()
    const ids = index.byStatus[status] || []
    const results: Info[] = []

    for (const id of ids) {
      const pending = await get(id)
      if (pending) results.push(pending)
    }

    return results
  }

  /**
   * List all pending tool calls (any status)
   */
  export async function listAll(): Promise<Info[]> {
    const keys = await Storage.list([STORAGE_PREFIX])
    const results: Info[] = []

    for (const k of keys) {
      if (k[1] === "_index") continue
      const pending = await get(k[1])
      if (pending) results.push(pending)
    }

    return results
  }

  /**
   * Find expired pending tool calls
   */
  export async function findExpired(): Promise<Info[]> {
    const index = await loadIndex()
    const now = Date.now()
    const expired: Info[] = []

    for (const entry of index.byExpiration) {
      if (entry.expiresAt <= now) {
        const pending = await get(entry.id)
        if (pending && pending.status === "waiting") {
          expired.push(pending)
        }
      } else {
        // Sorted by expiration, so we can stop here
        break
      }
    }

    return expired
  }

  /**
   * Check if a session has any waiting async tool calls
   */
  export async function hasWaiting(sessionID: string): Promise<boolean> {
    const pending = await listBySession(sessionID)
    return pending.some((p) => p.status === "waiting" || p.status === "processing")
  }

  /**
   * Get the next pending tool call for a session (oldest first)
   */
  export async function getNextWaiting(sessionID: string): Promise<Info | undefined> {
    const pending = await listBySession(sessionID)
    const waiting = pending.filter((p) => p.status === "waiting" || p.status === "processing")
    waiting.sort((a, b) => a.time.created - b.time.created)
    return waiting[0]
  }

  // ============================================================================
  // Cleanup Operations
  // ============================================================================

  /**
   * Clean up completed/failed/expired calls older than retention period
   */
  export async function cleanup(retentionMs: number = 7 * 24 * 60 * 60 * 1000): Promise<number> {
    const cutoff = Date.now() - retentionMs
    let count = 0

    for (const status of ["completed", "failed", "expired", "cancelled"] as Status[]) {
      const pending = await listByStatus(status)
      for (const p of pending) {
        if (p.time.completed && p.time.completed < cutoff) {
          await remove(p.id)
          count++
        }
      }
    }

    log.info("cleaned up pending tool calls", { count })
    return count
  }

  /**
   * Rebuild index from stored files (recovery operation)
   */
  export async function rebuildIndex(): Promise<void> {
    log.info("rebuilding pending tool index")

    const index: Index = {
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

    const keys = await Storage.list([STORAGE_PREFIX])
    for (const k of keys) {
      if (k[1] === "_index") continue
      try {
        const pending = await Storage.read<Info>(k)

        // Add to session index
        if (!index.bySession[pending.sessionID]) {
          index.bySession[pending.sessionID] = []
        }
        index.bySession[pending.sessionID].push(pending.id)

        // Add to status index
        if (!index.byStatus[pending.status]) {
          index.byStatus[pending.status] = []
        }
        index.byStatus[pending.status].push(pending.id)

        // Add to expiration index
        if (pending.timeout) {
          index.byExpiration.push({ id: pending.id, expiresAt: pending.timeout })
        }
      } catch (e) {
        log.warn("failed to read pending tool call", { key: k, error: e })
      }
    }

    // Sort expiration index
    index.byExpiration.sort((a, b) => a.expiresAt - b.expiresAt)

    await saveIndex(index)
    log.info("index rebuilt", {
      sessions: Object.keys(index.bySession).length,
      waiting: index.byStatus.waiting?.length || 0,
    })
  }
}
