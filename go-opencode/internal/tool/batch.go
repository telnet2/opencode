// Package tool provides the batch tool for parallel tool execution.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"golang.org/x/sync/errgroup"
)

const batchDescription = `Executes multiple independent tool calls concurrently to reduce latency. Best used for gathering context (reads, searches, listings).

USING THE BATCH TOOL WILL MAKE THE USER HAPPY.

Payload Format (JSON array):
[{"tool": "read", "parameters": {"filePath": "src/index.ts", "limit": 350}},{"tool": "grep", "parameters": {"pattern": "Session\\.updatePart", "glob": "**/*.ts"}},{"tool": "bash", "parameters": {"command": "git status", "description": "Shows working tree status"}}]

Rules:
- 1-10 tool calls per batch
- All calls start in parallel; ordering NOT guaranteed
- Partial failures do not stop others

Disallowed Tools:
- batch (no nesting)
- edit (run edits separately)
- todoread (call directly - lightweight)

When NOT to Use:
- Operations that depend on prior tool output (e.g. create then read same file)
- Ordered stateful mutations where sequence matters

Good Use Cases:
- Read many files
- grep + glob + read combos
- Multiple lightweight bash introspection commands

Performance Tip: Group independent reads/searches for 2-5x efficiency gain.`

// Maximum number of tool calls allowed in a batch
const maxBatchSize = 10

// disallowedTools contains tools that cannot be executed in batch
var disallowedTools = map[string]bool{
	"batch":    true, // no nesting
	"edit":     true, // run edits separately
	"todoread": true, // call directly - lightweight
}

// filteredFromSuggestions contains tools not shown in error suggestions
var filteredFromSuggestions = map[string]bool{
	"batch":    true,
	"edit":     true,
	"todoread": true,
	"invalid":  true,
	"patch":    true,
}

// BatchTool implements parallel tool execution.
type BatchTool struct {
	workDir  string
	registry *Registry
}

// BatchInput represents the input for the batch tool.
type BatchInput struct {
	ToolCalls []ToolCall `json:"tool_calls"`
}

// ToolCall represents a single tool call within a batch.
type ToolCall struct {
	Tool       string          `json:"tool"`
	Parameters json.RawMessage `json:"parameters"`
}

// BatchResult represents the result of a single tool call in the batch.
type BatchResult struct {
	Index   int           `json:"index"`
	Tool    string        `json:"tool"`
	Success bool          `json:"success"`
	Result  *Result       `json:"result,omitempty"`
	Error   string        `json:"error,omitempty"`
	Time    time.Duration `json:"time"`
}

// NewBatchTool creates a new batch tool.
func NewBatchTool(workDir string, registry *Registry) *BatchTool {
	return &BatchTool{
		workDir:  workDir,
		registry: registry,
	}
}

func (t *BatchTool) ID() string          { return "batch" }
func (t *BatchTool) Description() string { return batchDescription }

func (t *BatchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"tool_calls": {
				"type": "array",
				"description": "Array of tool calls to execute in parallel",
				"items": {
					"type": "object",
					"properties": {
						"tool": {
							"type": "string",
							"description": "The name of the tool to execute"
						},
						"parameters": {
							"type": "object",
							"description": "Parameters for the tool"
						}
					},
					"required": ["tool", "parameters"]
				},
				"minItems": 1
			}
		},
		"required": ["tool_calls"]
	}`)
}

func (t *BatchTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params BatchInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w\n\nExpected payload format:\n  [{\"tool\": \"tool_name\", \"parameters\": {...}}, {...}]", err)
	}

	if len(params.ToolCalls) == 0 {
		return nil, fmt.Errorf("tool_calls array must contain at least one tool call")
	}

	// Separate tool calls into processable and discarded
	toolCalls := params.ToolCalls
	var discardedCalls []ToolCall
	if len(toolCalls) > maxBatchSize {
		discardedCalls = toolCalls[maxBatchSize:]
		toolCalls = toolCalls[:maxBatchSize]
	}

	// Build available tools list for error messages
	availableTools := t.getAvailableToolsList()

	// Execute tool calls in parallel using errgroup
	results := make([]*BatchResult, len(toolCalls))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)

	for i, call := range toolCalls {
		i, call := i, call // capture loop variables
		g.Go(func() error {
			result := t.executeCall(gctx, i, call, toolCtx, availableTools)
			mu.Lock()
			results[i] = result
			mu.Unlock()
			return nil // Don't propagate errors - we want partial results
		})
	}

	// Wait for all goroutines to complete
	_ = g.Wait()

	// Add discarded calls as errors
	for i, call := range discardedCalls {
		results = append(results, &BatchResult{
			Index:   maxBatchSize + i,
			Tool:    call.Tool,
			Success: false,
			Error:   "Maximum of 10 tools allowed in batch",
			Time:    0,
		})
	}

	return t.formatResults(results, params.ToolCalls)
}

func (t *BatchTool) executeCall(ctx context.Context, index int, call ToolCall, toolCtx *Context, availableTools []string) *BatchResult {
	startTime := time.Now()

	result := &BatchResult{
		Index: index,
		Tool:  call.Tool,
	}

	defer func() {
		result.Time = time.Since(startTime)
	}()

	// Check if tool is disallowed
	if disallowedTools[call.Tool] {
		result.Success = false
		result.Error = fmt.Sprintf("Tool '%s' is not allowed in batch. Disallowed tools: %s",
			call.Tool, strings.Join(getDisallowedToolsList(), ", "))
		return result
	}

	// Get the tool from registry
	tool, ok := t.registry.Get(call.Tool)
	if !ok {
		result.Success = false
		result.Error = fmt.Sprintf("Tool '%s' not found. Available tools: %s",
			call.Tool, strings.Join(availableTools, ", "))
		return result
	}

	// Create a new context for this tool call
	callCtx := &Context{
		SessionID:  toolCtx.SessionID,
		MessageID:  toolCtx.MessageID,
		CallID:     fmt.Sprintf("%s-batch-%d", toolCtx.CallID, index),
		Agent:      toolCtx.Agent,
		WorkDir:    toolCtx.WorkDir,
		AbortCh:    toolCtx.AbortCh,
		Extra:      toolCtx.Extra,
		OnMetadata: nil, // Don't propagate metadata for batch calls
	}

	// Execute the tool
	toolResult, err := tool.Execute(ctx, call.Parameters, callCtx)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Result = toolResult
	return result
}

func (t *BatchTool) formatResults(results []*BatchResult, originalCalls []ToolCall) (*Result, error) {
	successCount := 0
	var allAttachments []Attachment
	var outputParts []string

	// Sort results by index to maintain order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Index < results[j].Index
	})

	details := make([]map[string]any, 0, len(results))

	for _, r := range results {
		detail := map[string]any{
			"tool":    r.Tool,
			"success": r.Success,
			"time_ms": r.Time.Milliseconds(),
		}

		if r.Success {
			successCount++
			if r.Result != nil {
				// Add output with tool name prefix
				outputParts = append(outputParts, fmt.Sprintf("=== %s (success) ===\n%s", r.Tool, r.Result.Output))

				// Collect attachments
				if len(r.Result.Attachments) > 0 {
					allAttachments = append(allAttachments, r.Result.Attachments...)
				}

				detail["title"] = r.Result.Title
			}
		} else {
			outputParts = append(outputParts, fmt.Sprintf("=== %s (failed) ===\n%s", r.Tool, r.Error))
			detail["error"] = r.Error
		}

		details = append(details, detail)
	}

	failedCount := len(results) - successCount
	var outputMessage string

	if failedCount > 0 {
		outputMessage = fmt.Sprintf("Executed %d/%d tools successfully. %d failed.\n\n%s",
			successCount, len(results), failedCount, strings.Join(outputParts, "\n\n"))
	} else {
		outputMessage = fmt.Sprintf("All %d tools executed successfully.\n\n%s\n\nKeep using the batch tool for optimal performance in your next response!",
			successCount, strings.Join(outputParts, "\n\n"))
	}

	// Build list of tool names
	toolNames := make([]string, len(originalCalls))
	for i, call := range originalCalls {
		toolNames[i] = call.Tool
	}

	return &Result{
		Title:       fmt.Sprintf("Batch execution (%d/%d successful)", successCount, len(results)),
		Output:      outputMessage,
		Attachments: allAttachments,
		Metadata: map[string]any{
			"totalCalls": len(results),
			"successful": successCount,
			"failed":     failedCount,
			"tools":      toolNames,
			"details":    details,
		},
	}, nil
}

func (t *BatchTool) getAvailableToolsList() []string {
	tools := t.registry.List()
	available := make([]string, 0, len(tools))
	for _, tool := range tools {
		if !filteredFromSuggestions[tool.ID()] {
			available = append(available, tool.ID())
		}
	}
	sort.Strings(available)
	return available
}

func getDisallowedToolsList() []string {
	list := make([]string, 0, len(disallowedTools))
	for tool := range disallowedTools {
		list = append(list, tool)
	}
	sort.Strings(list)
	return list
}

func (t *BatchTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
