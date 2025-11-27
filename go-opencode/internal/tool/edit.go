package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agnivade/levenshtein"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/opencode-ai/opencode/internal/event"
)

const editDescription = `Performs exact string replacements in files.

Usage:
- The file_path parameter must be an absolute path
- The old_string must exist in the file (exact match required)
- The new_string will replace old_string
- Use replace_all to replace all occurrences
- The edit will FAIL if old_string is not unique (unless using replace_all)`

// EditTool implements file editing.
type EditTool struct {
	workDir string
}

// EditInput represents the input for the edit tool.
type EditInput struct {
	FilePath   string `json:"filePath"`
	OldString  string `json:"oldString"`
	NewString  string `json:"newString"`
	ReplaceAll bool   `json:"replaceAll,omitempty"`
}

// NewEditTool creates a new edit tool.
func NewEditTool(workDir string) *EditTool {
	return &EditTool{workDir: workDir}
}

func (t *EditTool) ID() string          { return "edit" }
func (t *EditTool) Description() string { return editDescription }

func (t *EditTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"filePath": {
				"type": "string",
				"description": "The absolute path to the file to edit"
			},
			"oldString": {
				"type": "string",
				"description": "The exact text to replace"
			},
			"newString": {
				"type": "string",
				"description": "The text to replace it with"
			},
			"replaceAll": {
				"type": "boolean",
				"description": "Replace all occurrences (default: false)"
			}
		},
		"required": ["filePath", "oldString", "newString"]
	}`)
}

func (t *EditTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params EditInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if params.OldString == params.NewString {
		return nil, fmt.Errorf("old_string and new_string must be different")
	}

	// Read file
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)

	// Try exact match first
	var newText string
	var count int

	if params.ReplaceAll {
		count = strings.Count(text, params.OldString)
		if count == 0 {
			return t.fuzzyReplace(text, params, toolCtx)
		}
		newText = strings.ReplaceAll(text, params.OldString, params.NewString)
	} else {
		count = strings.Count(text, params.OldString)
		if count == 0 {
			return t.fuzzyReplace(text, params, toolCtx)
		}
		if count > 1 {
			return nil, fmt.Errorf("old_string appears %d times in file. Use replace_all or provide more context", count)
		}
		newText = strings.Replace(text, params.OldString, params.NewString, 1)
		count = 1
	}

	// Write file
	if err := os.WriteFile(params.FilePath, []byte(newText), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Publish event (SDK compatible: just file path)
	if toolCtx != nil && toolCtx.SessionID != "" {
		event.Publish(event.Event{
			Type: event.FileEdited,
			Data: event.FileEditedData{
				File: params.FilePath,
			},
		})
	}

	return &Result{
		Title:  fmt.Sprintf("Edited %s", filepath.Base(params.FilePath)),
		Output: fmt.Sprintf("Replaced %d occurrence(s)", count),
		Metadata: map[string]any{
			"file":         params.FilePath,
			"replacements": count,
		},
	}, nil
}

// fuzzyReplace attempts to find similar text when exact match fails.
func (t *EditTool) fuzzyReplace(text string, params EditInput, toolCtx *Context) (*Result, error) {
	// Try line-normalized matching
	normalizedOld := normalizeLineEndings(params.OldString)
	normalizedText := normalizeLineEndings(text)

	if strings.Contains(normalizedText, normalizedOld) {
		newText := strings.Replace(normalizedText, normalizedOld, params.NewString, 1)
		if err := os.WriteFile(params.FilePath, []byte(newText), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}

		// Publish event (SDK compatible: just file path)
		if toolCtx != nil && toolCtx.SessionID != "" {
			event.Publish(event.Event{
				Type: event.FileEdited,
				Data: event.FileEditedData{
					File: params.FilePath,
				},
			})
		}

		return &Result{
			Title:  fmt.Sprintf("Edited %s (normalized)", filepath.Base(params.FilePath)),
			Output: "Replaced 1 occurrence (with line ending normalization)",
		}, nil
	}

	// Try fuzzy matching with similarity
	match, similarity := findBestMatch(text, params.OldString)
	if match != "" && similarity >= 0.7 {
		newText := strings.Replace(text, match, params.NewString, 1)
		if err := os.WriteFile(params.FilePath, []byte(newText), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}

		// Publish event (SDK compatible: just file path)
		if toolCtx != nil && toolCtx.SessionID != "" {
			event.Publish(event.Event{
				Type: event.FileEdited,
				Data: event.FileEditedData{
					File: params.FilePath,
				},
			})
		}

		return &Result{
			Title:  fmt.Sprintf("Edited %s (fuzzy)", filepath.Base(params.FilePath)),
			Output: fmt.Sprintf("Replaced 1 occurrence (%.0f%% similarity)", similarity*100),
		}, nil
	}

	return nil, fmt.Errorf("old_string not found in file. The content may have changed or the string doesn't exist")
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// findBestMatch finds the substring most similar to target.
func findBestMatch(text, target string) (string, float64) {
	lines := strings.Split(text, "\n")
	targetLines := strings.Split(target, "\n")

	if len(targetLines) == 1 {
		// Single line - search for similar line
		bestMatch := ""
		bestSimilarity := 0.0

		for _, line := range lines {
			sim := similarity(line, target)
			if sim > bestSimilarity {
				bestSimilarity = sim
				bestMatch = line
			}
		}
		return bestMatch, bestSimilarity
	}

	// Multi-line - search for similar block
	targetLen := len(targetLines)
	bestMatch := ""
	bestSimilarity := 0.0

	for i := 0; i <= len(lines)-targetLen; i++ {
		block := strings.Join(lines[i:i+targetLen], "\n")
		sim := similarity(block, target)
		if sim > bestSimilarity {
			bestSimilarity = sim
			bestMatch = block
		}
	}

	return bestMatch, bestSimilarity
}

// similarity calculates normalized Levenshtein similarity using the agnivade/levenshtein package.
// This provides better performance and handles edge cases more robustly than a custom implementation.
func similarity(a, b string) float64 {
	// Handle empty strings
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// For very long strings, use a rough approximation to avoid performance issues
	// The levenshtein package handles large strings well, but we still cap for extreme cases
	if len(a) > 10000 || len(b) > 10000 {
		// Simple length-based approximation for extremely long strings
		maxLen := max(len(a), len(b))
		minLen := min(len(a), len(b))
		return float64(minLen) / float64(maxLen)
	}

	dist := levenshtein.ComputeDistance(a, b)
	maxLen := max(len(a), len(b))
	return 1.0 - float64(dist)/float64(maxLen)
}

func (t *EditTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}
