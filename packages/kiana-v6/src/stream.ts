/**
 * AI SDK UI-compatible stream protocol types and utilities.
 *
 * This implements the Vercel AI SDK's UIMessageStream protocol for compatibility
 * with useChat and other AI SDK UI components.
 *
 * @see https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol
 */

import type { SessionInfo } from "./types/session.js"

// ============================================================================
// Stream Part Types (wire format)
// ============================================================================

/** Message start - indicates beginning of a new message */
export interface StartPart {
  type: "start"
  messageId: string
  metadata?: MessageMetadata
}

/** Text streaming start */
export interface TextStartPart {
  type: "text-start"
  id: string
}

/** Incremental text content */
export interface TextDeltaPart {
  type: "text-delta"
  id: string
  delta: string
}

/** Text streaming complete */
export interface TextEndPart {
  type: "text-end"
  id: string
}

/** Reasoning streaming start */
export interface ReasoningStartPart {
  type: "reasoning-start"
  id: string
}

/** Incremental reasoning content */
export interface ReasoningDeltaPart {
  type: "reasoning-delta"
  id: string
  delta: string
}

/** Reasoning streaming complete */
export interface ReasoningEndPart {
  type: "reasoning-end"
  id: string
}

/** Tool input streaming start */
export interface ToolInputStartPart {
  type: "tool-input-start"
  toolCallId: string
  toolName: string
}

/** Tool input delta */
export interface ToolInputDeltaPart {
  type: "tool-input-delta"
  toolCallId: string
  inputTextDelta: string
}

/** Tool input fully available - ready to execute */
export interface ToolInputAvailablePart {
  type: "tool-input-available"
  toolCallId: string
  toolName: string
  input: Record<string, unknown>
}

/** Tool output available - execution complete */
export interface ToolOutputAvailablePart {
  type: "tool-output-available"
  toolCallId: string
  output: unknown
  metadata?: ToolMetadata
}

/** Tool execution error */
export interface ToolOutputErrorPart {
  type: "tool-output-error"
  toolCallId: string
  error: string
}

/** Step start marker */
export interface StartStepPart {
  type: "start-step"
}

/** Step finish marker */
export interface FinishStepPart {
  type: "finish-step"
  usage?: TokenUsage
}

/** Message finish marker */
export interface FinishPart {
  type: "finish"
  finishReason?: "stop" | "tool-calls" | "length" | "content-filter" | "error"
  usage?: TokenUsage
}

/** Custom data part for session info */
export interface DataSessionPart {
  type: "data-session"
  data: SessionInfo
}

/** Custom data part for session idle */
export interface DataSessionIdlePart {
  type: "data-session-idle"
  data: { sessionID: string }
}

/** Custom data part for todos */
export interface DataTodoPart {
  type: "data-todo"
  data: {
    sessionID: string
    todos: Array<{
      content: string
      status: "pending" | "in_progress" | "completed"
      activeForm: string
    }>
  }
}

/** Custom data part for subagent context */
export interface DataSubagentContextPart {
  type: "data-subagent-context"
  data: {
    parentSessionID: string
    depth: number
    agentType?: string
  }
}

/** Error part */
export interface ErrorPart {
  type: "error"
  error: string
}

/** Stream termination marker (not JSON, literal string) */
export type DonePart = "[DONE]"

// ============================================================================
// Union of all stream parts
// ============================================================================

export type StreamPart =
  | StartPart
  | TextStartPart
  | TextDeltaPart
  | TextEndPart
  | ReasoningStartPart
  | ReasoningDeltaPart
  | ReasoningEndPart
  | ToolInputStartPart
  | ToolInputDeltaPart
  | ToolInputAvailablePart
  | ToolOutputAvailablePart
  | ToolOutputErrorPart
  | StartStepPart
  | FinishStepPart
  | FinishPart
  | DataSessionPart
  | DataSessionIdlePart
  | DataTodoPart
  | DataSubagentContextPart
  | ErrorPart

// ============================================================================
// Metadata types
// ============================================================================

export interface MessageMetadata {
  sessionID?: string
  parentMessageID?: string
  model?: string
  [key: string]: unknown
}

export interface ToolMetadata {
  title?: string
  executionTime?: number
  [key: string]: unknown
}

export interface TokenUsage {
  inputTokens?: number
  outputTokens?: number
  reasoningTokens?: number
  cacheReadTokens?: number
  cacheWriteTokens?: number
}

// ============================================================================
// Stream Writer
// ============================================================================

export interface StreamWriter {
  /** Write a stream part */
  write(part: StreamPart): void
  /** Write the [DONE] termination marker */
  done(): void
  /** Close the stream */
  close(): void
}

/**
 * Create a stream writer that formats parts as SSE.
 */
export function createStreamWriter(
  writable: WritableStreamDefaultWriter<Uint8Array>
): StreamWriter {
  const encoder = new TextEncoder()

  return {
    write(part: StreamPart) {
      const line = `data: ${JSON.stringify(part)}\n\n`
      writable.write(encoder.encode(line))
    },
    done() {
      writable.write(encoder.encode("data: [DONE]\n\n"))
    },
    close() {
      writable.close()
    },
  }
}

/**
 * Format a stream part as an SSE line (for CLI/non-HTTP usage).
 */
export function formatSSE(part: StreamPart | DonePart): string {
  if (part === "[DONE]") {
    return "data: [DONE]\n\n"
  }
  return `data: ${JSON.stringify(part)}\n\n`
}

/**
 * Parse an SSE line back to a stream part.
 */
export function parseSSE(line: string): StreamPart | DonePart | null {
  if (!line.startsWith("data: ")) {
    return null
  }
  const data = line.slice(6).trim()
  if (data === "[DONE]") {
    return "[DONE]"
  }
  try {
    return JSON.parse(data) as StreamPart
  } catch {
    return null
  }
}

// ============================================================================
// HTTP Response helpers (for web server usage)
// ============================================================================

/**
 * Create SSE headers for AI SDK UI stream protocol.
 */
export function createSSEHeaders(): Record<string, string> {
  return {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache",
    "Connection": "keep-alive",
    "x-vercel-ai-ui-message-stream": "v1",
  }
}
