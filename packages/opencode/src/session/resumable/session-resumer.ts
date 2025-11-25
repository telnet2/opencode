import z from "zod"
import { Bus } from "../../bus"
import { Log } from "../../util/log"
import { Session } from "../index"
import { MessageV2 } from "../message-v2"
import { SessionPrompt } from "../prompt"
import { SessionStatus } from "../status"
import { PendingToolCall } from "./pending-tool"
import { Storage } from "../../storage/storage"

/**
 * SessionResumer - Handles resuming sessions when async tool results arrive
 *
 * This module is responsible for:
 * 1. Accepting tool results from external systems
 * 2. Updating the corresponding tool part in the message
 * 3. Resuming session processing
 * 4. Handling errors and timeouts
 */
export namespace SessionResumer {
  const log = Log.create({ service: "session-resumer" })

  // ============================================================================
  // Types
  // ============================================================================

  export const SubmitResultInput = z.object({
    pendingID: z.string(),
    result: PendingToolCall.Result,
  })
  export type SubmitResultInput = z.infer<typeof SubmitResultInput>

  export const SubmitErrorInput = z.object({
    pendingID: z.string(),
    error: z.string(),
  })
  export type SubmitErrorInput = z.infer<typeof SubmitErrorInput>

  export const WebhookPayload = z.discriminatedUnion("type", [
    z.object({
      type: z.literal("result"),
      pendingID: z.string(),
      result: PendingToolCall.Result,
    }),
    z.object({
      type: z.literal("error"),
      pendingID: z.string(),
      error: z.string(),
    }),
    z.object({
      type: z.literal("progress"),
      pendingID: z.string(),
      progress: z.object({
        percent: z.number().min(0).max(100).optional(),
        message: z.string().optional(),
      }),
    }),
  ])
  export type WebhookPayload = z.infer<typeof WebhookPayload>

  // ============================================================================
  // Events
  // ============================================================================

  export const Event = {
    SessionResuming: Bus.event(
      "session.resuming",
      z.object({
        sessionID: z.string(),
        pendingID: z.string(),
        tool: z.string(),
      }),
    ),
    SessionResumed: Bus.event(
      "session.resumed",
      z.object({
        sessionID: z.string(),
        pendingID: z.string(),
        success: z.boolean(),
      }),
    ),
    ResultSubmitted: Bus.event(
      "async-tool.result-submitted",
      z.object({
        pendingID: z.string(),
        sessionID: z.string(),
      }),
    ),
  }

  // ============================================================================
  // Result Submission
  // ============================================================================

  /**
   * Submit a successful result for a pending tool call and resume the session
   */
  export async function submitResult(input: SubmitResultInput): Promise<void> {
    const { pendingID, result } = input
    log.info("submitting result", { pendingID })

    // Load and validate pending call
    const pending = await PendingToolCall.get(pendingID)
    if (!pending) {
      throw new Error(`Unknown pending tool call: ${pendingID}`)
    }

    if (pending.status !== "waiting" && pending.status !== "processing") {
      throw new Error(`Pending tool call is not waiting: ${pendingID} (status: ${pending.status})`)
    }

    // Complete the pending call
    await PendingToolCall.complete(pendingID, result)

    // Update the tool part in the message
    await updateToolPart(pending, {
      status: "completed",
      input: pending.input,
      output: result.output,
      title: result.title,
      metadata: result.metadata || {},
      time: {
        start: pending.time.started || pending.time.created,
        end: Date.now(),
      },
    })

    Bus.publish(Event.ResultSubmitted, {
      pendingID,
      sessionID: pending.sessionID,
    })

    // Resume the session
    await resume(pending.sessionID)
  }

  /**
   * Submit an error for a pending tool call and resume the session
   */
  export async function submitError(input: SubmitErrorInput): Promise<void> {
    const { pendingID, error } = input
    log.info("submitting error", { pendingID, error })

    // Load and validate pending call
    const pending = await PendingToolCall.get(pendingID)
    if (!pending) {
      throw new Error(`Unknown pending tool call: ${pendingID}`)
    }

    if (pending.status !== "waiting" && pending.status !== "processing") {
      throw new Error(`Pending tool call is not waiting: ${pendingID} (status: ${pending.status})`)
    }

    // Fail the pending call
    await PendingToolCall.fail(pendingID, error)

    // Update the tool part in the message
    await updateToolPart(pending, {
      status: "error",
      input: pending.input,
      error,
      time: {
        start: pending.time.started || pending.time.created,
        end: Date.now(),
      },
    })

    // Resume the session
    await resume(pending.sessionID)
  }

  /**
   * Handle webhook payload from external system
   */
  export async function handleWebhook(payload: WebhookPayload, signature?: string): Promise<void> {
    log.info("handling webhook", { type: payload.type, pendingID: payload.pendingID })

    // Validate signature if present
    const pending = await PendingToolCall.get(payload.pendingID)
    if (!pending) {
      throw new Error(`Unknown pending tool call: ${payload.pendingID}`)
    }

    if (pending.webhookSecret && signature) {
      const isValid = await verifyWebhookSignature(JSON.stringify(payload), signature, pending.webhookSecret)
      if (!isValid) {
        throw new Error("Invalid webhook signature")
      }
    }

    switch (payload.type) {
      case "result":
        await submitResult({ pendingID: payload.pendingID, result: payload.result })
        break
      case "error":
        await submitError({ pendingID: payload.pendingID, error: payload.error })
        break
      case "progress":
        await updateProgress(payload.pendingID, payload.progress)
        break
    }
  }

  // ============================================================================
  // Session Resumption
  // ============================================================================

  /**
   * Resume a session after a pending tool call completes
   */
  export async function resume(sessionID: string): Promise<void> {
    log.info("resuming session", { sessionID })

    // Check if there are more pending calls for this session
    const hasMore = await PendingToolCall.hasWaiting(sessionID)
    if (hasMore) {
      log.info("session still has pending calls, not resuming yet", { sessionID })
      return
    }

    // Get session info
    const session = await Session.get(sessionID)
    if (!session) {
      log.error("session not found", { sessionID })
      return
    }

    Bus.publish(Event.SessionResuming, {
      sessionID,
      pendingID: "",
      tool: "",
    })

    try {
      // Continue the session loop
      // This will pick up from where it left off, with the tool result now available
      await SessionPrompt.loop(sessionID)

      Bus.publish(Event.SessionResumed, {
        sessionID,
        pendingID: "",
        success: true,
      })
    } catch (e) {
      log.error("failed to resume session", { sessionID, error: e })
      Bus.publish(Event.SessionResumed, {
        sessionID,
        pendingID: "",
        success: false,
      })
    }
  }

  /**
   * Resume a session with an error (for expired/cancelled tool calls)
   */
  export async function resumeWithError(pending: PendingToolCall.Info, error: string): Promise<void> {
    log.info("resuming session with error", { sessionID: pending.sessionID, error })

    // Update the tool part with error
    await updateToolPart(pending, {
      status: "error",
      input: pending.input,
      error,
      time: {
        start: pending.time.started || pending.time.created,
        end: Date.now(),
      },
    })

    // Resume the session
    await resume(pending.sessionID)
  }

  // ============================================================================
  // Helper Functions
  // ============================================================================

  /**
   * Update the tool part in the message with new state
   */
  async function updateToolPart(
    pending: PendingToolCall.Info,
    state: MessageV2.ToolStateCompleted | MessageV2.ToolStateError,
  ): Promise<void> {
    try {
      const part: MessageV2.ToolPart = {
        id: pending.partID,
        messageID: pending.messageID,
        sessionID: pending.sessionID,
        type: "tool",
        callID: pending.callID,
        tool: pending.tool,
        state,
      }

      await Session.updatePart(part)
      log.info("updated tool part", { partID: pending.partID, status: state.status })
    } catch (e) {
      log.error("failed to update tool part", {
        partID: pending.partID,
        error: e,
      })
      throw e
    }
  }

  /**
   * Update progress for a pending tool call
   */
  async function updateProgress(
    pendingID: string,
    progress: { percent?: number; message?: string },
  ): Promise<void> {
    const pending = await PendingToolCall.get(pendingID)
    if (!pending) return

    // Update tool part metadata with progress
    try {
      const existingPart = await Storage.read<MessageV2.ToolPart>(["part", pending.messageID, pending.partID])
      if (existingPart && existingPart.state.status === "running") {
        await Session.updatePart({
          ...existingPart,
          state: {
            ...existingPart.state,
            metadata: {
              ...existingPart.state.metadata,
              progress,
            },
          },
        })
      }
    } catch (e) {
      log.warn("failed to update progress", { pendingID, error: e })
    }
  }

  /**
   * Verify webhook signature using HMAC-SHA256
   */
  async function verifyWebhookSignature(payload: string, signature: string, secret: string): Promise<boolean> {
    try {
      const encoder = new TextEncoder()
      const key = await crypto.subtle.importKey(
        "raw",
        encoder.encode(secret),
        { name: "HMAC", hash: "SHA-256" },
        false,
        ["sign"],
      )
      const sig = await crypto.subtle.sign("HMAC", key, encoder.encode(payload))
      const expected = Array.from(new Uint8Array(sig))
        .map((b) => b.toString(16).padStart(2, "0"))
        .join("")
      return signature === expected || signature === `sha256=${expected}`
    } catch {
      return false
    }
  }

  // ============================================================================
  // Status Queries
  // ============================================================================

  /**
   * Get the async waiting status for a session
   */
  export async function getWaitingStatus(sessionID: string): Promise<
    | {
        waiting: true
        pendingCalls: Array<{
          id: string
          tool: string
          createdAt: number
          externalRef?: string
        }>
      }
    | { waiting: false }
  > {
    const pending = await PendingToolCall.listBySession(sessionID)
    const waiting = pending.filter((p) => p.status === "waiting" || p.status === "processing")

    if (waiting.length === 0) {
      return { waiting: false }
    }

    return {
      waiting: true,
      pendingCalls: waiting.map((p) => ({
        id: p.id,
        tool: p.tool,
        createdAt: p.time.created,
        externalRef: p.externalRef,
      })),
    }
  }

  /**
   * Check if the session can be safely shut down
   * (all immediate tool calls are done, only async ones remaining)
   */
  export async function canShutdown(sessionID: string): Promise<boolean> {
    const status = SessionStatus.get(sessionID)

    // Can shutdown if idle or waiting for async
    if (!status) return true
    if (status.type === "idle") return true
    if (status.type === "waiting_async") return true

    // Check if busy but only with async tool calls
    if (status.type === "busy") {
      const pending = await PendingToolCall.listBySession(sessionID)
      return pending.length > 0 && pending.every((p) => p.status === "waiting")
    }

    return false
  }
}
