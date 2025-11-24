// WebSocket Client for JSON-RPC communication

import {
  JSONRPCRequest,
  JSONRPCResponse,
  ExecuteCommandResult,
} from '@/types/api';

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private requestId = 0;
  private pendingRequests = new Map<number, {
    resolve: (result: ExecuteCommandResult) => void;
    reject: (error: Error) => void;
  }>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private onConnectCallbacks: (() => void)[] = [];
  private onDisconnectCallbacks: (() => void)[] = [];

  constructor(private url: string) {}

  /**
   * Connect to the WebSocket server
   */
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          this.reconnectAttempts = 0;
          this.onConnectCallbacks.forEach(cb => cb());
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
        };

        this.ws.onclose = () => {
          this.onDisconnectCallbacks.forEach(cb => cb());
          this.attemptReconnect();
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Disconnect from the WebSocket server
   */
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.pendingRequests.clear();
  }

  /**
   * Execute a shell command
   */
  executeCommand(
    sessionId: string,
    command: string,
    args: string[] = []
  ): Promise<ExecuteCommandResult> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket is not connected'));
        return;
      }

      const id = ++this.requestId;
      const request: JSONRPCRequest = {
        jsonrpc: '2.0',
        method: 'shell.execute',
        params: {
          session_id: sessionId,
          command,
          args,
        },
        id,
      };

      this.pendingRequests.set(id, { resolve, reject });
      this.ws.send(JSON.stringify(request));
    });
  }

  /**
   * Register a callback for connection events
   */
  onConnect(callback: () => void): void {
    this.onConnectCallbacks.push(callback);
  }

  /**
   * Register a callback for disconnection events
   */
  onDisconnect(callback: () => void): void {
    this.onDisconnectCallbacks.push(callback);
  }

  /**
   * Check if the WebSocket is connected
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  private handleMessage(data: string): void {
    try {
      const response: JSONRPCResponse = JSON.parse(data);
      const pending = this.pendingRequests.get(response.id);

      if (!pending) {
        console.warn('Received response for unknown request ID:', response.id);
        return;
      }

      this.pendingRequests.delete(response.id);

      if (response.error) {
        pending.reject(new Error(response.error.message));
      } else if (response.result) {
        pending.resolve(response.result);
      } else {
        pending.reject(new Error('Invalid response: no result or error'));
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      this.connect().catch((error) => {
        console.error('Reconnection failed:', error);
      });
    }, delay);
  }
}
