package tool

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/opencode-ai/opencode/internal/agent"
)

const taskDescription = `Launch a new agent to handle complex, multi-step tasks autonomously.

The Task tool launches specialized agents (subprocesses) that autonomously handle complex tasks.
Each agent type has specific capabilities and tools available to it.

Available agent types:
- general: General-purpose agent for researching and exploration
- explore: Fast agent specialized for codebase exploration
- plan: Planning agent for analysis without making changes

Usage notes:
- Launch multiple agents concurrently when possible
- Each agent invocation is stateless
- The agent's outputs should be trusted
- Specify desired thoroughness level when calling explore agent`

// TaskTool allows spawning sub-agents for complex tasks.
type TaskTool struct {
	workDir       string
	agentRegistry *agent.Registry
	executor      TaskExecutor
}

// TaskExecutor is the interface for executing subtasks.
type TaskExecutor interface {
	// ExecuteSubtask runs a subtask with the given agent and prompt.
	ExecuteSubtask(ctx context.Context, sessionID string, agentName string, prompt string, opts TaskOptions) (*TaskResult, error)
}

// TaskOptions contains options for task execution.
type TaskOptions struct {
	Model       string // Optional model override (sonnet, opus, haiku)
	ResumeFrom  string // Optional agent ID to resume from
	Description string // Short description of the task
}

// TaskResult represents the result of a subtask.
type TaskResult struct {
	Output    string         `json:"output"`
	SessionID string         `json:"sessionID"`
	AgentID   string         `json:"agentID,omitempty"`
	Error     string         `json:"error,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TaskInput represents the input for the task tool.
// SDK compatible: uses camelCase field names to match TypeScript.
type TaskInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagentType"`
	Model        string `json:"model,omitempty"`
	Resume       string `json:"resume,omitempty"`
}

// NewTaskTool creates a new task tool.
func NewTaskTool(workDir string, registry *agent.Registry) *TaskTool {
	if registry == nil {
		registry = agent.NewRegistry()
	}
	return &TaskTool{
		workDir:       workDir,
		agentRegistry: registry,
	}
}

// SetExecutor sets the task executor.
func (t *TaskTool) SetExecutor(executor TaskExecutor) {
	t.executor = executor
}

func (t *TaskTool) ID() string          { return "Task" }
func (t *TaskTool) Description() string { return taskDescription }

func (t *TaskTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"description": {
				"type": "string",
				"description": "A short (3-5 word) description of the task"
			},
			"prompt": {
				"type": "string",
				"description": "The detailed task for the agent to perform"
			},
			"subagentType": {
				"type": "string",
				"description": "The type of specialized agent to use (general, explore, plan)"
			},
			"model": {
				"type": "string",
				"description": "Optional model to use (sonnet, opus, haiku)",
				"enum": ["sonnet", "opus", "haiku"]
			},
			"resume": {
				"type": "string",
				"description": "Optional agent ID to resume from"
			}
		},
		"required": ["description", "prompt", "subagentType"]
	}`)
}

func (t *TaskTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params TaskInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate required fields
	if params.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if params.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if params.SubagentType == "" {
		return nil, fmt.Errorf("subagentType is required")
	}

	// Get subagent configuration
	subagent, err := t.agentRegistry.Get(params.SubagentType)
	if err != nil {
		// Try with lowercase
		subagent, err = t.agentRegistry.Get(params.SubagentType)
		if err != nil {
			return nil, fmt.Errorf("unknown subagent type: %s. Available types: general, explore, plan", params.SubagentType)
		}
	}

	// Verify subagent mode
	if !subagent.IsSubagent() {
		return nil, fmt.Errorf("agent %s cannot be used as subagent (mode: %s)", params.SubagentType, subagent.Mode)
	}

	// Update metadata
	toolCtx.SetMetadata(params.Description, map[string]any{
		"subagent": params.SubagentType,
		"status":   "starting",
	})

	// If no executor is set, return a placeholder result
	if t.executor == nil {
		return &Result{
			Title:  fmt.Sprintf("Task: %s", params.Description),
			Output: fmt.Sprintf("[Subtask execution not configured]\n\nAgent: %s\nPrompt: %s", params.SubagentType, params.Prompt),
			Metadata: map[string]any{
				"subagent":    params.SubagentType,
				"status":      "skipped",
				"description": params.Description,
			},
		}, nil
	}

	// Execute subtask
	opts := TaskOptions{
		Model:       params.Model,
		ResumeFrom:  params.Resume,
		Description: params.Description,
	}

	result, err := t.executor.ExecuteSubtask(ctx, toolCtx.SessionID, params.SubagentType, params.Prompt, opts)
	if err != nil {
		return &Result{
			Title:  fmt.Sprintf("Subtask failed: %s", params.Description),
			Output: fmt.Sprintf("Error: %s", err.Error()),
			Metadata: map[string]any{
				"subagent": params.SubagentType,
				"status":   "failed",
				"error":    err.Error(),
			},
		}, nil
	}

	metadata := map[string]any{
		"subagent": params.SubagentType,
		"status":   "completed",
	}
	if result.SessionID != "" {
		metadata["sessionID"] = result.SessionID
	}
	if result.AgentID != "" {
		metadata["agentID"] = result.AgentID
	}
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			metadata[k] = v
		}
	}

	return &Result{
		Title:    fmt.Sprintf("Completed: %s", params.Description),
		Output:   result.Output,
		Metadata: metadata,
	}, nil
}

func (t *TaskTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}

// GetAvailableAgents returns a list of available subagent types.
func (t *TaskTool) GetAvailableAgents() []string {
	agents := t.agentRegistry.ListSubagents()
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	return names
}

// GetAgentDescription returns the description of a specific agent.
func (t *TaskTool) GetAgentDescription(name string) (string, error) {
	ag, err := t.agentRegistry.Get(name)
	if err != nil {
		return "", err
	}
	return ag.Description, nil
}
