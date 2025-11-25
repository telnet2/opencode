import { Log } from "../../util/log"
import { Session } from "../index"
import { SessionStatus } from "../status"
import { PendingToolCall } from "./pending-tool"
import { SessionResumer } from "./session-resumer"

/**
 * Recovery - Startup recovery routines for resumable sessions
 *
 * This module handles:
 * 1. Finding sessions that were waiting for async tool results
 * 2. Checking for expired tool calls and resuming with errors
 * 3. Restoring session status to waiting_async state
 * 4. Cleaning up stale pending calls
 */
export namespace Recovery {
  const log = Log.create({ service: "session-recovery" })

  // Default timeout for async tool calls (24 hours)
  const DEFAULT_TIMEOUT_MS = 24 * 60 * 60 * 1000

  // Cleanup retention period (7 days)
  const CLEANUP_RETENTION_MS = 7 * 24 * 60 * 60 * 1000

  /**
   * Run full recovery on startup
   * This should be called when the server starts
   */
  export async function run(): Promise<RecoveryResult> {
    log.info("starting session recovery")
    const result: RecoveryResult = {
      recovered: [],
      expired: [],
      errors: [],
      cleanedUp: 0,
    }

    try {
      // First, rebuild the index to ensure consistency
      await PendingToolCall.rebuildIndex()

      // Check for expired tool calls
      const expired = await PendingToolCall.findExpired()
      for (const pending of expired) {
        try {
          log.info("expiring timed out tool call", {
            id: pending.id,
            sessionID: pending.sessionID,
            tool: pending.tool,
          })

          await PendingToolCall.expire(pending.id)
          await SessionResumer.resumeWithError(pending, "Tool execution timed out")
          result.expired.push({
            pendingID: pending.id,
            sessionID: pending.sessionID,
            tool: pending.tool,
          })
        } catch (e) {
          log.error("failed to handle expired tool call", {
            id: pending.id,
            error: e,
          })
          result.errors.push({
            pendingID: pending.id,
            error: e instanceof Error ? e.message : String(e),
          })
        }
      }

      // Find all waiting sessions and restore their status
      const waiting = await PendingToolCall.listByStatus("waiting")
      const processing = await PendingToolCall.listByStatus("processing")
      const allPending = [...waiting, ...processing]

      // Group by session
      const bySession = new Map<string, PendingToolCall.Info[]>()
      for (const pending of allPending) {
        const list = bySession.get(pending.sessionID) || []
        list.push(pending)
        bySession.set(pending.sessionID, list)
      }

      // Restore session status for each session
      for (const [sessionID, pendingCalls] of bySession) {
        try {
          // Check if session still exists
          const session = await Session.get(sessionID).catch(() => null)
          if (!session) {
            log.warn("session not found, cleaning up pending calls", {
              sessionID,
              count: pendingCalls.length,
            })

            // Cancel orphaned pending calls
            for (const pending of pendingCalls) {
              await PendingToolCall.cancel(pending.id)
            }
            continue
          }

          // Mark any "processing" calls back to "waiting" since we're restarting
          for (const pending of pendingCalls) {
            if (pending.status === "processing") {
              await PendingToolCall.update(pending.id, { status: "waiting" })
            }
          }

          // Set session status to waiting_async
          const firstPending = pendingCalls[0]
          SessionStatus.set(sessionID, {
            type: "waiting_async",
            pendingID: firstPending.id,
            tool: firstPending.tool,
            since: firstPending.time.created,
            timeout: firstPending.timeout,
          })

          result.recovered.push({
            sessionID,
            pendingCount: pendingCalls.length,
            tools: pendingCalls.map((p) => p.tool),
          })

          log.info("recovered session with pending calls", {
            sessionID,
            pendingCount: pendingCalls.length,
          })
        } catch (e) {
          log.error("failed to recover session", {
            sessionID,
            error: e,
          })
          result.errors.push({
            pendingID: pendingCalls[0]?.id || "unknown",
            error: e instanceof Error ? e.message : String(e),
          })
        }
      }

      // Clean up old completed/failed calls
      result.cleanedUp = await PendingToolCall.cleanup(CLEANUP_RETENTION_MS)

      log.info("session recovery complete", {
        recovered: result.recovered.length,
        expired: result.expired.length,
        errors: result.errors.length,
        cleanedUp: result.cleanedUp,
      })
    } catch (e) {
      log.error("recovery failed", { error: e })
      result.errors.push({
        pendingID: "global",
        error: e instanceof Error ? e.message : String(e),
      })
    }

    return result
  }

  /**
   * Check and expire timed out tool calls
   * This can be called periodically
   */
  export async function checkExpired(): Promise<number> {
    const expired = await PendingToolCall.findExpired()
    let count = 0

    for (const pending of expired) {
      try {
        await PendingToolCall.expire(pending.id)
        await SessionResumer.resumeWithError(pending, "Tool execution timed out")
        count++
      } catch (e) {
        log.error("failed to expire tool call", {
          id: pending.id,
          error: e,
        })
      }
    }

    if (count > 0) {
      log.info("expired timed out tool calls", { count })
    }

    return count
  }

  /**
   * Get recovery status summary
   */
  export async function getStatus(): Promise<RecoveryStatus> {
    const waiting = await PendingToolCall.listByStatus("waiting")
    const processing = await PendingToolCall.listByStatus("processing")

    const sessionIDs = new Set<string>()
    for (const p of [...waiting, ...processing]) {
      sessionIDs.add(p.sessionID)
    }

    return {
      waitingSessions: sessionIDs.size,
      waitingCalls: waiting.length,
      processingCalls: processing.length,
      sessions: Array.from(sessionIDs).map((sessionID) => {
        const calls = [...waiting, ...processing].filter((p) => p.sessionID === sessionID)
        return {
          sessionID,
          pendingCalls: calls.length,
          oldestCall: Math.min(...calls.map((p) => p.time.created)),
        }
      }),
    }
  }

  // ============================================================================
  // Types
  // ============================================================================

  export interface RecoveryResult {
    recovered: Array<{
      sessionID: string
      pendingCount: number
      tools: string[]
    }>
    expired: Array<{
      pendingID: string
      sessionID: string
      tool: string
    }>
    errors: Array<{
      pendingID: string
      error: string
    }>
    cleanedUp: number
  }

  export interface RecoveryStatus {
    waitingSessions: number
    waitingCalls: number
    processingCalls: number
    sessions: Array<{
      sessionID: string
      pendingCalls: number
      oldestCall: number
    }>
  }

  // ============================================================================
  // Background Tasks
  // ============================================================================

  let expirationInterval: ReturnType<typeof setInterval> | null = null

  /**
   * Start background expiration checking
   */
  export function startExpirationChecker(intervalMs: number = 60 * 1000): void {
    if (expirationInterval) {
      clearInterval(expirationInterval)
    }

    expirationInterval = setInterval(async () => {
      try {
        await checkExpired()
      } catch (e) {
        log.error("expiration check failed", { error: e })
      }
    }, intervalMs)

    log.info("started expiration checker", { intervalMs })
  }

  /**
   * Stop background expiration checking
   */
  export function stopExpirationChecker(): void {
    if (expirationInterval) {
      clearInterval(expirationInterval)
      expirationInterval = null
      log.info("stopped expiration checker")
    }
  }
}
