/**
 * JSON-RPC 2.0 Types for MemSh API
 */

export interface JSONRPCRequest {
  jsonrpc: "2.0"
  method: string
  params?: Record<string, unknown>
  id: number | string
}

export interface JSONRPCResponse<T = unknown> {
  jsonrpc: "2.0"
  result?: T
  error?: JSONRPCError
  id: number | string | null
}

export interface JSONRPCError {
  code: number
  message: string
  data?: unknown
}

/**
 * JSON-RPC Error Codes
 */
export const ErrorCodes = {
  ParseError: -32700,
  InvalidRequest: -32600,
  MethodNotFound: -32601,
  InvalidParams: -32602,
  InternalError: -32603,
} as const

/**
 * Session information
 */
export interface SessionInfo {
  id: string
  created_at: string
  last_used: string
  cwd: string
}

/**
 * Create session response
 */
export interface CreateSessionResponse {
  session: SessionInfo
}

/**
 * List sessions response
 */
export interface ListSessionsResponse {
  sessions: SessionInfo[]
}

/**
 * Remove session request
 */
export interface RemoveSessionRequest {
  session_id: string
}

/**
 * Remove session response
 */
export interface RemoveSessionResponse {
  success: boolean
  message: string
}

/**
 * Execute command parameters
 */
export interface ExecuteCommandParams {
  session_id: string
  command: string
  args?: string[]
}

/**
 * Execute command result
 */
export interface ExecuteCommandResult {
  output: string[]
  cwd: string
  error?: string
}

/**
 * Client configuration options
 */
export interface MemshClientOptions {
  /** Base URL of the memsh server (e.g., "http://localhost:8080") */
  baseUrl: string
  /** Connection timeout in milliseconds */
  timeout?: number
  /** Auto-reconnect on disconnect */
  autoReconnect?: boolean
  /** Maximum reconnection attempts */
  maxReconnectAttempts?: number
  /** Reconnection delay in milliseconds */
  reconnectDelay?: number
}

/**
 * Connection state
 */
export type ConnectionState = "disconnected" | "connecting" | "connected" | "reconnecting"
