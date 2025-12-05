package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/opencode-ai/opencode/internal/event"
)

const writeDescription = `Writes content to a file on the local filesystem.

Usage:
- The file_path parameter must be an absolute path
- This tool will overwrite existing files
- Parent directories will be created if they don't exist
- ALWAYS prefer editing existing files over creating new ones`

// WriteTool implements file writing.
type WriteTool struct {
	workDir string
}

// WriteInput represents the input for the write tool.
// SDK compatible: uses camelCase field names to match TypeScript.
type WriteInput struct {
	FilePath string `json:"filePath"`
	Content  string `json:"content"`
}

// NewWriteTool creates a new write tool.
func NewWriteTool(workDir string) *WriteTool {
	return &WriteTool{workDir: workDir}
}

func (t *WriteTool) ID() string          { return "Write" }
func (t *WriteTool) Description() string { return writeDescription }

func (t *WriteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"filePath": {
				"type": "string",
				"description": "The absolute path to the file to write"
			},
			"content": {
				"type": "string",
				"description": "The content to write to the file"
			}
		},
		"required": ["filePath", "content"]
	}`)
}

func (t *WriteTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params WriteInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(params.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(params.FilePath, []byte(params.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Publish file edited event (SDK compatible: just file path)
	if toolCtx != nil && toolCtx.SessionID != "" {
		event.Publish(event.Event{
			Type: event.FileEdited,
			Data: event.FileEditedData{
				File: params.FilePath,
			},
		})
	}

	return &Result{
		Title: fmt.Sprintf("Wrote %s", filepath.Base(params.FilePath)),
		Output: fmt.Sprintf("Successfully wrote %d bytes to %s",
			len(params.Content), params.FilePath),
		Metadata: map[string]any{
			"file":  params.FilePath,
			"bytes": len(params.Content),
		},
	}, nil
}

func (t *WriteTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
