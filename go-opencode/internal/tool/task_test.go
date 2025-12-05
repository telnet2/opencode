package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/opencode-ai/opencode/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskTool(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	assert.NotNil(t, tool)
	assert.Equal(t, "Task", tool.ID())
	assert.NotEmpty(t, tool.Description())
}

func TestNewTaskTool_WithRegistry(t *testing.T) {
	registry := agent.NewRegistry()
	tool := NewTaskTool("/tmp", registry)
	assert.NotNil(t, tool)
}

func TestTaskTool_Parameters(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	params := tool.Parameters()
	assert.NotNil(t, params)

	// Verify JSON is valid
	var schema map[string]any
	err := json.Unmarshal(params, &schema)
	require.NoError(t, err)

	assert.Equal(t, "object", schema["type"])
	properties := schema["properties"].(map[string]any)
	assert.Contains(t, properties, "description")
	assert.Contains(t, properties, "prompt")
	assert.Contains(t, properties, "subagentType")
	assert.Contains(t, properties, "model")
	assert.Contains(t, properties, "resume")
}

func TestTaskTool_Execute_MissingDescription(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	input := json.RawMessage(`{"prompt": "test", "subagentType": "general"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestTaskTool_Execute_MissingPrompt(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	input := json.RawMessage(`{"description": "test", "subagentType": "general"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestTaskTool_Execute_MissingSubagentType(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	input := json.RawMessage(`{"description": "test", "prompt": "test prompt"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subagentType is required")
}

func TestTaskTool_Execute_UnknownSubagent(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	input := json.RawMessage(`{"description": "test", "prompt": "test prompt", "subagentType": "nonexistent"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown subagent type")
}

func TestTaskTool_Execute_NonSubagentMode(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	// "build" is a primary agent, not a subagent
	input := json.RawMessage(`{"description": "test", "prompt": "test prompt", "subagentType": "build"}`)
	_, err := tool.Execute(ctx, input, toolCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used as subagent")
}

func TestTaskTool_Execute_WithoutExecutor(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	input := json.RawMessage(`{"description": "test task", "prompt": "test prompt", "subagentType": "general"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Title, "Task: test task")
	assert.Contains(t, result.Output, "Subtask execution not configured")
	assert.Equal(t, "skipped", result.Metadata["status"])
}

// MockTaskExecutor is a mock implementation of TaskExecutor.
type MockTaskExecutor struct {
	ExecuteFunc func(ctx context.Context, sessionID, agentName, prompt string, opts TaskOptions) (*TaskResult, error)
}

func (m *MockTaskExecutor) ExecuteSubtask(ctx context.Context, sessionID, agentName, prompt string, opts TaskOptions) (*TaskResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, sessionID, agentName, prompt, opts)
	}
	return &TaskResult{Output: "mock output"}, nil
}

func TestTaskTool_Execute_WithExecutor(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	executor := &MockTaskExecutor{
		ExecuteFunc: func(ctx context.Context, sessionID, agentName, prompt string, opts TaskOptions) (*TaskResult, error) {
			return &TaskResult{
				Output:    "subtask completed successfully",
				SessionID: "session-123",
				AgentID:   "agent-456",
				Metadata: map[string]any{
					"tokens": 100,
				},
			}, nil
		},
	}
	tool.SetExecutor(executor)

	ctx := context.Background()
	toolCtx := &Context{
		WorkDir:   "/tmp",
		SessionID: "parent-session",
	}

	input := json.RawMessage(`{"description": "test task", "prompt": "test prompt", "subagentType": "general"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Title, "Completed: test task")
	assert.Equal(t, "subtask completed successfully", result.Output)
	assert.Equal(t, "completed", result.Metadata["status"])
	assert.Equal(t, "session-123", result.Metadata["sessionID"])
	assert.Equal(t, "agent-456", result.Metadata["agentID"])
	assert.Equal(t, 100, result.Metadata["tokens"])
}

func TestTaskTool_Execute_ExecutorError(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	executor := &MockTaskExecutor{
		ExecuteFunc: func(ctx context.Context, sessionID, agentName, prompt string, opts TaskOptions) (*TaskResult, error) {
			return nil, assert.AnError
		},
	}
	tool.SetExecutor(executor)

	ctx := context.Background()
	toolCtx := &Context{WorkDir: "/tmp"}

	input := json.RawMessage(`{"description": "test task", "prompt": "test prompt", "subagentType": "general"}`)
	result, err := tool.Execute(ctx, input, toolCtx)
	require.NoError(t, err) // Execute itself doesn't error
	assert.Contains(t, result.Title, "Subtask failed")
	assert.Equal(t, "failed", result.Metadata["status"])
}

func TestTaskTool_GetAvailableAgents(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	agents := tool.GetAvailableAgents()
	assert.NotEmpty(t, agents)
	assert.Contains(t, agents, "general")
	assert.Contains(t, agents, "explore")
}

func TestTaskTool_GetAgentDescription(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)

	desc, err := tool.GetAgentDescription("general")
	require.NoError(t, err)
	assert.NotEmpty(t, desc)

	_, err = tool.GetAgentDescription("nonexistent")
	assert.Error(t, err)
}

func TestTaskTool_EinoTool(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	einoTool := tool.EinoTool()
	assert.NotNil(t, einoTool)
}

func TestTaskTool_MetadataCallback(t *testing.T) {
	tool := NewTaskTool("/tmp", nil)
	ctx := context.Background()

	metadataCalled := false
	toolCtx := &Context{
		WorkDir: "/tmp",
		OnMetadata: func(title string, meta map[string]any) {
			metadataCalled = true
			assert.Equal(t, "test task", title)
			assert.Equal(t, "general", meta["subagent"])
			assert.Equal(t, "starting", meta["status"])
		},
	}

	input := json.RawMessage(`{"description": "test task", "prompt": "test prompt", "subagentType": "general"}`)
	_, _ = tool.Execute(ctx, input, toolCtx)
	assert.True(t, metadataCalled)
}

func TestTaskInput(t *testing.T) {
	input := TaskInput{
		Description:  "test",
		Prompt:       "test prompt",
		SubagentType: "general",
		Model:        "sonnet",
		Resume:       "agent-123",
	}

	data, err := json.Marshal(input)
	require.NoError(t, err)

	var decoded TaskInput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, input.Description, decoded.Description)
	assert.Equal(t, input.Prompt, decoded.Prompt)
	assert.Equal(t, input.SubagentType, decoded.SubagentType)
	assert.Equal(t, input.Model, decoded.Model)
	assert.Equal(t, input.Resume, decoded.Resume)
}

func TestTaskOptions(t *testing.T) {
	opts := TaskOptions{
		Model:       "opus",
		ResumeFrom:  "session-123",
		Description: "test task",
	}

	assert.Equal(t, "opus", opts.Model)
	assert.Equal(t, "session-123", opts.ResumeFrom)
	assert.Equal(t, "test task", opts.Description)
}

func TestTaskResult(t *testing.T) {
	result := TaskResult{
		Output:    "completed",
		SessionID: "session-123",
		AgentID:   "agent-456",
		Metadata: map[string]any{
			"tokens": 100,
		},
	}

	assert.Equal(t, "completed", result.Output)
	assert.Equal(t, "session-123", result.SessionID)
	assert.Equal(t, 100, result.Metadata["tokens"])
}
