package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
)

const globDescription = `Fast file pattern matching tool that works with any codebase size.

Usage:
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns`

// GlobTool implements file pattern matching.
type GlobTool struct {
	workDir string
}

// GlobInput represents the input for the glob tool.
type GlobInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// NewGlobTool creates a new glob tool.
func NewGlobTool(workDir string) *GlobTool {
	return &GlobTool{workDir: workDir}
}

func (t *GlobTool) ID() string          { return "glob" }
func (t *GlobTool) Description() string { return globDescription }

func (t *GlobTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "The glob pattern to match files against"
			},
			"path": {
				"type": "string",
				"description": "Directory to search in (default: current directory)"
			}
		},
		"required": ["pattern"]
	}`)
}

func (t *GlobTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params GlobInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	searchDir := t.workDir
	if toolCtx != nil && toolCtx.WorkDir != "" {
		searchDir = toolCtx.WorkDir
	}
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			searchDir = params.Path
		} else {
			searchDir = filepath.Join(searchDir, params.Path)
		}
	}

	// Use ripgrep for fast file enumeration
	cmd := exec.CommandContext(ctx, "rg", "--files", "--glob", params.Pattern)
	cmd.Dir = searchDir

	output, err := cmd.Output()
	if err != nil {
		// No matches is not an error
		if len(output) == 0 {
			return &Result{
				Title:  "Glob search",
				Output: "No files matched the pattern",
				Metadata: map[string]any{
					"pattern": params.Pattern,
					"count":   0,
				},
			}, nil
		}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter empty strings
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}

	// Limit results
	const maxFiles = 100
	truncated := false
	if len(result) > maxFiles {
		result = result[:maxFiles]
		truncated = true
	}

	outputStr := strings.Join(result, "\n")
	if truncated {
		outputStr += fmt.Sprintf("\n\n(Showing %d of more files)", maxFiles)
	}

	return &Result{
		Title:  fmt.Sprintf("Found %d files", len(result)),
		Output: outputStr,
		Metadata: map[string]any{
			"pattern":   params.Pattern,
			"count":     len(result),
			"truncated": truncated,
		},
	}, nil
}

func (t *GlobTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
