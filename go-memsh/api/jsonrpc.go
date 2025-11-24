package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
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

// Error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// HandleJSONRPC processes a JSON-RPC request
func HandleJSONRPC(ctx context.Context, sm *SessionManager, request *JSONRPCRequest) *JSONRPCResponse {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	// Validate JSON-RPC version
	if request.JSONRPC != "2.0" {
		response.Error = &JSONRPCError{
			Code:    InvalidRequest,
			Message: "Invalid JSON-RPC version",
		}
		return response
	}

	// Handle methods
	switch request.Method {
	case "shell.execute":
		result, err := handleExecute(ctx, sm, request.Params)
		if err != nil {
			response.Error = err
		} else {
			response.Result = result
		}

	default:
		response.Error = &JSONRPCError{
			Code:    MethodNotFound,
			Message: fmt.Sprintf("Method not found: %s", request.Method),
		}
	}

	return response
}

// handleExecute handles the shell.execute method
func handleExecute(ctx context.Context, sm *SessionManager, params json.RawMessage) (*ExecuteCommandResult, *JSONRPCError) {
	var execParams ExecuteCommandParams
	if err := json.Unmarshal(params, &execParams); err != nil {
		return nil, &JSONRPCError{
			Code:    InvalidParams,
			Message: "Invalid parameters",
			Data:    err.Error(),
		}
	}

	// Validate session ID
	if execParams.SessionID == "" {
		return nil, &JSONRPCError{
			Code:    InvalidParams,
			Message: "session_id is required",
		}
	}

	// Validate command
	if execParams.Command == "" {
		return nil, &JSONRPCError{
			Code:    InvalidParams,
			Message: "command is required",
		}
	}

	// Get session
	session, err := sm.GetSession(execParams.SessionID)
	if err != nil {
		return nil, &JSONRPCError{
			Code:    InvalidParams,
			Message: "Invalid session",
			Data:    err.Error(),
		}
	}

	// Execute command
	output, cwd, execErr := session.ExecuteCommand(ctx, execParams.Command, execParams.Args)

	result := &ExecuteCommandResult{
		Output: output,
		Cwd:    cwd,
	}

	if execErr != nil {
		result.Error = execErr.Error()
	}

	return result, nil
}
