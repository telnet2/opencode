package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
)

const listDescription = `Lists files and directories in a specified path.

Usage:
- Returns file names, types (file/directory), and sizes
- Useful for exploring directory structure`

// ListTool implements directory listing.
type ListTool struct {
	workDir string
}

// ListInput represents the input for the list tool.
type ListInput struct {
	Path string `json:"path"`
}

// NewListTool creates a new list tool.
func NewListTool(workDir string) *ListTool {
	return &ListTool{workDir: workDir}
}

func (t *ListTool) ID() string          { return "List" }
func (t *ListTool) Description() string { return listDescription }

func (t *ListTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to list (defaults to current directory)"
			}
		}
	}`)
}

// FileEntry represents a file or directory entry.
type FileEntry struct {
	Name        string `json:"name"`
	IsDirectory bool   `json:"isDirectory"`
	Size        int64  `json:"size"`
}

func (t *ListTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params ListInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	listPath := t.workDir
	if toolCtx != nil && toolCtx.WorkDir != "" {
		listPath = toolCtx.WorkDir
	}
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			listPath = params.Path
		} else {
			listPath = filepath.Join(listPath, params.Path)
		}
	}

	entries, err := os.ReadDir(listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileEntry
	for _, entry := range entries {
		info, _ := entry.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, FileEntry{
			Name:        entry.Name(),
			IsDirectory: entry.IsDir(),
			Size:        size,
		})
	}

	// Format output
	var sb strings.Builder
	for _, f := range files {
		typeStr := "file"
		if f.IsDirectory {
			typeStr = "dir "
		}
		sb.WriteString(fmt.Sprintf("[%s] %s", typeStr, f.Name))
		if !f.IsDirectory {
			sb.WriteString(fmt.Sprintf(" (%d bytes)", f.Size))
		}
		sb.WriteString("\n")
	}

	return &Result{
		Title:  fmt.Sprintf("Listed %d items", len(files)),
		Output: sb.String(),
		Metadata: map[string]any{
			"path":  listPath,
			"count": len(files),
		},
	}, nil
}

func (t *ListTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
