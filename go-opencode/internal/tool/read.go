package tool

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
)

const readDescription = `Reads a file from the local filesystem.

Usage:
- The file_path parameter must be an absolute path
- By default, reads up to 2000 lines from the beginning
- You can optionally specify offset and limit for pagination
- Returns file contents with line numbers
- Can read image files and return them as base64 data`

// ReadTool implements file reading.
type ReadTool struct {
	workDir string
}

// ReadInput represents the input for the read tool.
type ReadInput struct {
	FilePath string `json:"filePath"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// NewReadTool creates a new read tool.
func NewReadTool(workDir string) *ReadTool {
	return &ReadTool{workDir: workDir}
}

func (t *ReadTool) ID() string          { return "read" }
func (t *ReadTool) Description() string { return readDescription }

func (t *ReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"filePath": {
				"type": "string",
				"description": "The absolute path to the file to read"
			},
			"offset": {
				"type": "integer",
				"description": "Line number to start reading from"
			},
			"limit": {
				"type": "integer",
				"description": "Number of lines to read (default: 2000)"
			}
		},
		"required": ["filePath"]
	}`)
}

func (t *ReadTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params ReadInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Default limit
	if params.Limit <= 0 {
		params.Limit = 2000
	}

	// Block .env files (except allowed patterns like .env.sample, .example)
	if shouldBlockEnvFile(params.FilePath) {
		return nil, fmt.Errorf("The user has blocked you from reading %s, DO NOT make further attempts to read it", params.FilePath)
	}

	// Check if file exists
	info, err := os.Stat(params.FilePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", params.FilePath)
	}

	// Handle directories
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", params.FilePath)
	}

	// Handle images
	if isImageFile(params.FilePath) {
		return t.readImage(params.FilePath)
	}

	// Check for binary content
	if isBinaryFile(params.FilePath) {
		return nil, fmt.Errorf("file appears to be binary")
	}

	// Read text file
	file, err := os.Open(params.FilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if params.Offset > 0 && lineNum < params.Offset {
			continue
		}
		if len(lines) >= params.Limit {
			break
		}

		line := scanner.Text()
		// Truncate long lines
		if len(line) > 2000 {
			line = line[:2000] + "..."
		}
		lines = append(lines, fmt.Sprintf("%05d| %s", lineNum, line))
	}

	// Format output with <file> tags like TypeScript
	var sb strings.Builder
	sb.WriteString("<file>\n")
	sb.WriteString(strings.Join(lines, "\n"))

	lastReadLine := params.Offset + len(lines)
	hasMoreLines := lineNum > lastReadLine

	if hasMoreLines {
		sb.WriteString(fmt.Sprintf("\n\n(File has more lines. Use 'offset' parameter to read beyond line %d)", lastReadLine))
	} else {
		sb.WriteString(fmt.Sprintf("\n\n(End of file - total %d lines)", lineNum))
	}
	sb.WriteString("\n</file>")
	output := sb.String()

	return &Result{
		Title:  fmt.Sprintf("Read %s", filepath.Base(params.FilePath)),
		Output: output,
		Metadata: map[string]any{
			"file":       params.FilePath,
			"lines":      len(lines),
			"totalLines": lineNum,
		},
	}, nil
}

func (t *ReadTool) readImage(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	mediaType := detectMediaType(path)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, base64.StdEncoding.EncodeToString(data))

	return &Result{
		Title:  fmt.Sprintf("Read %s", filepath.Base(path)),
		Output: "(Image file)",
		Attachments: []Attachment{
			{
				Filename:  filepath.Base(path),
				MediaType: mediaType,
				URL:       dataURL,
			},
		},
	}, nil
}

func (t *ReadTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" ||
		ext == ".gif" || ext == ".bmp" || ext == ".webp"
}

func isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 8000)
	n, _ := file.Read(buf)
	if n == 0 {
		return false
	}

	// Check for null bytes
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	// Check ratio of non-printable characters
	nonPrintable := 0
	for i := 0; i < n; i++ {
		if buf[i] < 32 && buf[i] != '\n' && buf[i] != '\r' && buf[i] != '\t' {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(n) > 0.3
}

func detectMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// shouldBlockEnvFile checks if a file should be blocked based on .env patterns.
// Whitelist: .env.sample, .example suffixes are allowed.
func shouldBlockEnvFile(filePath string) bool {
	// Whitelist patterns that are allowed
	whitelist := []string{".env.sample", ".example"}
	for _, w := range whitelist {
		if strings.HasSuffix(filePath, w) {
			return false
		}
	}

	// Block files containing .env in the path
	if strings.Contains(filePath, ".env") {
		return true
	}

	return false
}

