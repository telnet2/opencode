package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
)

const grepDescription = `A powerful content search tool built on ripgrep.

Usage:
- Supports full regex syntax (e.g., "log.*Error", "function\\s+\\w+")
- Filter files with glob parameter (e.g., "*.js", "**/*.tsx")
- Returns matching lines with file paths and line numbers`

// GrepTool implements content search.
type GrepTool struct {
	workDir string
}

// GrepInput represents the input for the grep tool.
type GrepInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Include string `json:"include,omitempty"` // file pattern to include (e.g., "*.js")
}

// NewGrepTool creates a new grep tool.
func NewGrepTool(workDir string) *GrepTool {
	return &GrepTool{workDir: workDir}
}

func (t *GrepTool) ID() string          { return "grep" }
func (t *GrepTool) Description() string { return grepDescription }

func (t *GrepTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "The regex pattern to search for in file contents"
			},
			"path": {
				"type": "string",
				"description": "The directory to search in. Defaults to the current working directory."
			},
			"include": {
				"type": "string",
				"description": "File pattern to include in the search (e.g. \"*.js\", \"*.{ts,tsx}\")"
			}
		},
		"required": ["pattern"]
	}`)
}

// GrepMatch represents a search match.
type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

func (t *GrepTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params GrepInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	args := []string{
		"--line-number",
		"--with-filename",
		"--color=never",
	}

	if params.Include != "" {
		args = append(args, "--glob", params.Include)
	}

	args = append(args, params.Pattern)

	searchPath := t.workDir
	if toolCtx != nil && toolCtx.WorkDir != "" {
		searchPath = toolCtx.WorkDir
	}
	if params.Path != "" {
		searchPath = params.Path
	}
	args = append(args, searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	output, _ := cmd.Output()

	if len(output) == 0 {
		return &Result{
			Title:  "Search results",
			Output: "No matches found",
			Metadata: map[string]any{
				"pattern": params.Pattern,
				"count":   0,
			},
		}, nil
	}

	var matches []GrepMatch
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		// Parse: file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		lineNum, _ := strconv.Atoi(parts[1])
		matches = append(matches, GrepMatch{
			File:    parts[0],
			Line:    lineNum,
			Content: parts[2],
		})
	}

	// Limit results
	const maxMatches = 100
	truncated := false
	if len(matches) > maxMatches {
		matches = matches[:maxMatches]
		truncated = true
	}

	// Format output
	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("%s:%d: %s\n", m.File, m.Line, m.Content))
	}

	if truncated {
		sb.WriteString(fmt.Sprintf("\n(Showing %d of more matches)", maxMatches))
	}

	return &Result{
		Title:  fmt.Sprintf("Found %d matches", len(matches)),
		Output: sb.String(),
		Metadata: map[string]any{
			"pattern":   params.Pattern,
			"count":     len(matches),
			"truncated": truncated,
		},
	}, nil
}

func (t *GrepTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
