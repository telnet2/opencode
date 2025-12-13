package client

import (
	"encoding/json"
	"time"
)

// SessionInfo represents session information
type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	Cwd       string    `json:"cwd"`
}

// CreateSessionResponse represents the response for session creation
type CreateSessionResponse struct {
	Session SessionInfo `json:"session"`
}

// ListSessionsResponse represents the response for listing sessions
type ListSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

// RemoveSessionRequest represents the request for removing a session
type RemoveSessionRequest struct {
	SessionID string `json:"session_id"`
}

// RemoveSessionResponse represents the response for session removal
type RemoveSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ExecuteCommandParams represents parameters for execute command
type ExecuteCommandParams struct {
	SessionID string   `json:"session_id"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
}

// ExecuteCommandResult represents the result of command execution
type ExecuteCommandResult struct {
	Output []string `json:"output"`
	Cwd    string   `json:"cwd"`
	Error  string   `json:"error,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      int64           `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      int64         `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)
