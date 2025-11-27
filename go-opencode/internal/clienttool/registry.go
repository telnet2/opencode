// Package clienttool provides a registry for client-side tools.
// Client tools are external tools that clients can register and execute
// via the HTTP API.
package clienttool

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/opencode-ai/opencode/internal/event"
)

// ToolDefinition represents a client-registered tool.
type ToolDefinition struct {
	ID          string         `json:"id"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ExecutionRequest represents a pending tool execution request.
type ExecutionRequest struct {
	Type      string         `json:"type"`
	RequestID string         `json:"requestID"`
	SessionID string         `json:"sessionID"`
	MessageID string         `json:"messageID"`
	CallID    string         `json:"callID"`
	Tool      string         `json:"tool"`
	Input     map[string]any `json:"input"`
}

// ToolResult represents a successful execution result.
type ToolResult struct {
	Status   string         `json:"status"` // "success"
	Title    string         `json:"title"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToolResponse is the response from a client tool execution.
type ToolResponse struct {
	Status   string         `json:"status"`
	Title    string         `json:"title,omitempty"`
	Output   string         `json:"output,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// pendingRequest represents a pending tool execution waiting for result.
type pendingRequest struct {
	request  ExecutionRequest
	clientID string
	result   chan ToolResponse
	timeout  *time.Timer
}

// Registry manages client-side tools.
type Registry struct {
	mu sync.RWMutex

	// clientID -> toolID -> definition
	tools map[string]map[string]ToolDefinition

	// requestID -> pending request
	pending map[string]*pendingRequest
}

// globalRegistry is the default registry instance.
var globalRegistry = NewRegistry()

// NewRegistry creates a new client tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:   make(map[string]map[string]ToolDefinition),
		pending: make(map[string]*pendingRequest),
	}
}

// Register registers tools for a client.
// Returns the list of registered tool IDs (with client prefix).
func Register(clientID string, tools []ToolDefinition) []string {
	return globalRegistry.Register(clientID, tools)
}

// Register registers tools for a client.
func (r *Registry) Register(clientID string, tools []ToolDefinition) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tools[clientID] == nil {
		r.tools[clientID] = make(map[string]ToolDefinition)
	}

	registered := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolID := prefixToolID(clientID, tool.ID)
		r.tools[clientID][toolID] = ToolDefinition{
			ID:          toolID,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		}
		registered = append(registered, toolID)
	}

	// Publish event
	event.Publish(event.Event{
		Type: event.ClientToolRegistered,
		Data: event.ClientToolRegisteredData{
			ClientID: clientID,
			ToolIDs:  registered,
		},
	})

	return registered
}

// Unregister removes tools for a client.
// If toolIDs is empty, all tools for the client are removed.
// Returns the list of unregistered tool IDs.
func Unregister(clientID string, toolIDs []string) []string {
	return globalRegistry.Unregister(clientID, toolIDs)
}

// Unregister removes tools for a client.
func (r *Registry) Unregister(clientID string, toolIDs []string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	clientTools := r.tools[clientID]
	if clientTools == nil {
		return nil
	}

	var unregistered []string
	if len(toolIDs) == 0 {
		// Unregister all
		for id := range clientTools {
			unregistered = append(unregistered, id)
		}
		delete(r.tools, clientID)
	} else {
		for _, id := range toolIDs {
			fullID := id
			if !IsClientTool(id) {
				fullID = prefixToolID(clientID, id)
			}
			if _, ok := clientTools[fullID]; ok {
				delete(clientTools, fullID)
				unregistered = append(unregistered, fullID)
			}
		}
	}

	if len(unregistered) > 0 {
		event.Publish(event.Event{
			Type: event.ClientToolUnregistered,
			Data: event.ClientToolUnregisteredData{
				ClientID: clientID,
				ToolIDs:  unregistered,
			},
		})
	}

	return unregistered
}

// GetTools returns tools for a specific client.
func GetTools(clientID string) []ToolDefinition {
	return globalRegistry.GetTools(clientID)
}

// GetTools returns tools for a specific client.
func (r *Registry) GetTools(clientID string) []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clientTools := r.tools[clientID]
	if clientTools == nil {
		return nil
	}

	tools := make([]ToolDefinition, 0, len(clientTools))
	for _, t := range clientTools {
		tools = append(tools, t)
	}
	return tools
}

// GetAllTools returns all registered client tools.
func GetAllTools() map[string]ToolDefinition {
	return globalRegistry.GetAllTools()
}

// GetAllTools returns all registered client tools.
func (r *Registry) GetAllTools() map[string]ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make(map[string]ToolDefinition)
	for _, clientTools := range r.tools {
		for id, tool := range clientTools {
			all[id] = tool
		}
	}
	return all
}

// Execute sends a tool request to the client and waits for response.
func Execute(ctx context.Context, clientID string, req ExecutionRequest, timeout time.Duration) (*ToolResult, error) {
	return globalRegistry.Execute(ctx, clientID, req, timeout)
}

// Execute sends a tool request to the client and waits for response.
func (r *Registry) Execute(ctx context.Context, clientID string, req ExecutionRequest, timeout time.Duration) (*ToolResult, error) {
	req.Type = "client-tool-request"

	resultCh := make(chan ToolResponse, 1)
	timer := time.NewTimer(timeout)

	pending := &pendingRequest{
		request:  req,
		clientID: clientID,
		result:   resultCh,
		timeout:  timer,
	}

	r.mu.Lock()
	r.pending[req.RequestID] = pending
	r.mu.Unlock()

	// Publish event for SSE clients
	event.Publish(event.Event{
		Type: event.ClientToolRequest,
		Data: event.ClientToolRequestData{
			ClientID: clientID,
			Request:  req,
		},
	})

	event.Publish(event.Event{
		Type: event.ClientToolExecuting,
		Data: event.ClientToolStatusData{
			SessionID: req.SessionID,
			MessageID: req.MessageID,
			CallID:    req.CallID,
			Tool:      req.Tool,
			ClientID:  clientID,
		},
	})

	// Wait for result or timeout
	select {
	case resp := <-resultCh:
		timer.Stop()
		r.mu.Lock()
		delete(r.pending, req.RequestID)
		r.mu.Unlock()

		if resp.Status == "error" {
			event.Publish(event.Event{
				Type: event.ClientToolFailed,
				Data: event.ClientToolStatusData{
					SessionID: req.SessionID,
					MessageID: req.MessageID,
					CallID:    req.CallID,
					Tool:      req.Tool,
					ClientID:  clientID,
					Error:     resp.Error,
				},
			})
			return nil, errors.New(resp.Error)
		}

		event.Publish(event.Event{
			Type: event.ClientToolCompleted,
			Data: event.ClientToolStatusData{
				SessionID: req.SessionID,
				MessageID: req.MessageID,
				CallID:    req.CallID,
				Tool:      req.Tool,
				ClientID:  clientID,
				Success:   true,
			},
		})

		return &ToolResult{
			Status:   resp.Status,
			Title:    resp.Title,
			Output:   resp.Output,
			Metadata: resp.Metadata,
		}, nil

	case <-timer.C:
		r.mu.Lock()
		delete(r.pending, req.RequestID)
		r.mu.Unlock()

		event.Publish(event.Event{
			Type: event.ClientToolFailed,
			Data: event.ClientToolStatusData{
				SessionID: req.SessionID,
				MessageID: req.MessageID,
				CallID:    req.CallID,
				Tool:      req.Tool,
				ClientID:  clientID,
				Error:     "timeout",
			},
		})
		return nil, errors.New("client tool execution timed out")

	case <-ctx.Done():
		timer.Stop()
		r.mu.Lock()
		delete(r.pending, req.RequestID)
		r.mu.Unlock()
		return nil, ctx.Err()
	}
}

// SubmitResult handles result submission from client.
// Returns true if the result was accepted, false if the request was not found.
func SubmitResult(requestID string, resp ToolResponse) bool {
	return globalRegistry.SubmitResult(requestID, resp)
}

// SubmitResult handles result submission from client.
func (r *Registry) SubmitResult(requestID string, resp ToolResponse) bool {
	r.mu.RLock()
	pending := r.pending[requestID]
	r.mu.RUnlock()

	if pending == nil {
		return false
	}

	select {
	case pending.result <- resp:
		return true
	default:
		return false
	}
}

// Cleanup removes all tools and cancels pending requests for a client.
func Cleanup(clientID string) {
	globalRegistry.Cleanup(clientID)
}

// Cleanup removes all tools and cancels pending requests for a client.
func (r *Registry) Cleanup(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Cancel pending requests
	for reqID, pending := range r.pending {
		if pending.clientID == clientID {
			pending.timeout.Stop()
			close(pending.result)
			delete(r.pending, reqID)
		}
	}

	// Remove tools
	if tools := r.tools[clientID]; tools != nil {
		toolIDs := make([]string, 0, len(tools))
		for id := range tools {
			toolIDs = append(toolIDs, id)
		}
		delete(r.tools, clientID)

		if len(toolIDs) > 0 {
			event.Publish(event.Event{
				Type: event.ClientToolUnregistered,
				Data: event.ClientToolUnregisteredData{
					ClientID: clientID,
					ToolIDs:  toolIDs,
				},
			})
		}
	}
}

// FindClientForTool finds which client owns a tool.
// Returns empty string if not found.
func FindClientForTool(toolID string) string {
	return globalRegistry.FindClientForTool(toolID)
}

// FindClientForTool finds which client owns a tool.
func (r *Registry) FindClientForTool(toolID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for clientID, tools := range r.tools {
		if _, ok := tools[toolID]; ok {
			return clientID
		}
	}
	return ""
}

// GetTool returns a tool definition by its ID.
func GetTool(toolID string) (ToolDefinition, bool) {
	return globalRegistry.GetTool(toolID)
}

// GetTool returns a tool definition by its ID.
func (r *Registry) GetTool(toolID string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, tools := range r.tools {
		if tool, ok := tools[toolID]; ok {
			return tool, true
		}
	}
	return ToolDefinition{}, false
}

// IsClientTool checks if a tool ID is a client tool.
func IsClientTool(toolID string) bool {
	return strings.HasPrefix(toolID, "client_")
}

// prefixToolID adds the client prefix to a tool ID.
func prefixToolID(clientID, toolID string) string {
	return "client_" + clientID + "_" + toolID
}

// Reset clears all registered tools and pending requests (for testing).
func Reset() {
	globalRegistry.Reset()
}

// Reset clears all registered tools and pending requests.
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Cancel all pending requests
	for _, pending := range r.pending {
		pending.timeout.Stop()
		close(pending.result)
	}

	r.tools = make(map[string]map[string]ToolDefinition)
	r.pending = make(map[string]*pendingRequest)
}
