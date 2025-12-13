// API Types for MemSh

export interface SessionInfo {
  id: string;
  created_at: string;
  last_used: string;
  cwd: string;
}

export interface CreateSessionResponse {
  session: SessionInfo;
}

export interface ListSessionsResponse {
  sessions: SessionInfo[];
}

export interface RemoveSessionRequest {
  session_id: string;
}

export interface RemoveSessionResponse {
  success: boolean;
  message?: string;
}

export interface ErrorResponse {
  error: string;
}

// JSON-RPC Types

export interface JSONRPCRequest {
  jsonrpc: '2.0';
  method: string;
  params?: {
    session_id: string;
    command: string;
    args?: string[];
  };
  id: number;
}

export interface ExecuteCommandResult {
  output: string[];
  cwd: string;
  error?: string;
}

export interface JSONRPCResponse {
  jsonrpc: '2.0';
  result?: ExecuteCommandResult;
  error?: {
    code: number;
    message: string;
    data?: string;
  };
  id: number;
}

// File System Types

export interface FileNode {
  name: string;
  path: string;
  isDir: boolean;
  size?: number;
  children?: FileNode[];
  expanded?: boolean;
}
