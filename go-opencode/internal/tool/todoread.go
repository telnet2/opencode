package tool

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/pkg/types"
)

const todoreadDescription = `Use this tool to read your todo list`

// TodoReadTool reads the current todo list for a session.
type TodoReadTool struct {
	workDir string
	storage *storage.Storage
}

// NewTodoReadTool creates a new todoread tool.
func NewTodoReadTool(workDir string, store *storage.Storage) *TodoReadTool {
	return &TodoReadTool{
		workDir: workDir,
		storage: store,
	}
}

func (t *TodoReadTool) ID() string          { return "todoread" }
func (t *TodoReadTool) Description() string { return todoreadDescription }

func (t *TodoReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": []
	}`)
}

func (t *TodoReadTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	// Get todos directly from storage (avoiding session import)
	var todos []types.TodoInfo
	err := t.storage.Get(ctx, []string{"todo", toolCtx.SessionID}, &todos)
	if err == storage.ErrNotFound {
		todos = []types.TodoInfo{}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get todos: %w", err)
	}

	// Count non-completed todos
	nonCompleted := 0
	for _, todo := range todos {
		if todo.Status != "completed" {
			nonCompleted++
		}
	}

	output, _ := json.MarshalIndent(todos, "", "  ")
	return &Result{
		Title:  fmt.Sprintf("%d todos", nonCompleted),
		Output: string(output),
		Metadata: map[string]any{
			"todos": todos,
		},
	}, nil
}

func (t *TodoReadTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
