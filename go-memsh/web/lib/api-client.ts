// API Client for MemSh REST API

import {
  SessionInfo,
  CreateSessionResponse,
  ListSessionsResponse,
  RemoveSessionRequest,
  RemoveSessionResponse,
  ErrorResponse,
} from '@/types/api';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error: ErrorResponse = await response.json().catch(() => ({
      error: `HTTP ${response.status}: ${response.statusText}`,
    }));
    throw new Error(error.error);
  }
  return response.json();
}

export const apiClient = {
  /**
   * Create a new shell session
   */
  async createSession(): Promise<SessionInfo> {
    const response = await fetch(`${API_BASE_URL}/api/v1/session/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    const data = await handleResponse<CreateSessionResponse>(response);
    return data.session;
  },

  /**
   * List all active sessions
   */
  async listSessions(): Promise<SessionInfo[]> {
    const response = await fetch(`${API_BASE_URL}/api/v1/session/list`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    const data = await handleResponse<ListSessionsResponse>(response);
    return data.sessions;
  },

  /**
   * Remove a session by ID
   */
  async removeSession(sessionId: string): Promise<void> {
    const request: RemoveSessionRequest = { session_id: sessionId };
    const response = await fetch(`${API_BASE_URL}/api/v1/session/remove`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });
    await handleResponse<RemoveSessionResponse>(response);
  },

  /**
   * Get the WebSocket URL for REPL
   */
  getWebSocketURL(): string {
    const wsProtocol = API_BASE_URL.startsWith('https') ? 'wss' : 'ws';
    const url = new URL(API_BASE_URL);
    return `${wsProtocol}://${url.host}/api/v1/session/repl`;
  },
};
