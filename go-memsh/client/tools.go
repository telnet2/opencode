package client

import (
	"fmt"
	"strings"
)

const (
	// DefaultReadLimit is the default number of lines to read
	DefaultReadLimit = 2000
	// MaxLineLength is the maximum length for a single line
	MaxLineLength = 2000
	// DefaultMaxOutputLength is the maximum output length for bash
	DefaultMaxOutputLength = 30000
	// DefaultSearchLimit is the default limit for search results
	DefaultSearchLimit = 100
)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Title    string
	Output   string
	Metadata map[string]interface{}
}

// BashOptions configures the bash tool
type BashOptions struct {
	Command     string
	Description string
	Timeout     int // milliseconds
}

// BashTool executes a shell command
func BashTool(session *Session, opts BashOptions) (*ToolResult, error) {
	result, err := session.Execute(opts.Command)
	if err != nil {
		return nil, err
	}

	output := strings.Join(result.Output, "\n")

	// Truncate if too long
	if len(output) > DefaultMaxOutputLength {
		output = output[:DefaultMaxOutputLength]
		output += "\n\n(Output was truncated due to length limit)"
	}

	// Add error to output if present
	if result.Error != "" {
		output += fmt.Sprintf("\n\nError: %s", result.Error)
	}

	return &ToolResult{
		Title:  opts.Description,
		Output: output,
		Metadata: map[string]interface{}{
			"output":      output,
			"cwd":         result.Cwd,
			"error":       result.Error,
			"description": opts.Description,
		},
	}, nil
}

// ReadOptions configures the read tool
type ReadOptions struct {
	FilePath string
	Offset   int // line number to start from (0-based)
	Limit    int // number of lines to read
}

// ReadTool reads a file from the memsh filesystem
func ReadTool(session *Session, opts ReadOptions) (*ToolResult, error) {
	// Check if file exists
	isFile, err := session.IsFile(opts.FilePath)
	if err != nil {
		return nil, err
	}

	if !isFile {
		isDir, _ := session.IsDirectory(opts.FilePath)
		if isDir {
			return nil, fmt.Errorf("cannot read directory: %s. Use the ls tool to list directory contents", opts.FilePath)
		}
		return nil, fmt.Errorf("file not found: %s", opts.FilePath)
	}

	// Read file content
	content, err := session.ReadFile(opts.FilePath)
	if err != nil {
		return nil, err
	}

	allLines := strings.Split(content, "\n")

	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultReadLimit
	}

	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Slice lines based on offset and limit
	endIndex := offset + limit
	if endIndex > len(allLines) {
		endIndex = len(allLines)
	}

	raw := allLines[offset:endIndex]

	// Truncate long lines
	for i, line := range raw {
		if len(line) > MaxLineLength {
			raw[i] = line[:MaxLineLength] + "..."
		}
	}

	// Format with line numbers
	var numbered []string
	for i, line := range raw {
		lineNum := fmt.Sprintf("%05d", i+offset+1)
		numbered = append(numbered, fmt.Sprintf("%s| %s", lineNum, line))
	}

	preview := strings.Join(raw[:min(20, len(raw))], "\n")

	var output strings.Builder
	output.WriteString("<file>\n")
	output.WriteString(strings.Join(numbered, "\n"))

	totalLines := len(allLines)
	lastReadLine := offset + len(raw)
	hasMoreLines := totalLines > lastReadLine

	if hasMoreLines {
		output.WriteString(fmt.Sprintf("\n\n(File has more lines. Use 'offset' parameter to read beyond line %d)", lastReadLine))
	} else {
		output.WriteString(fmt.Sprintf("\n\n(End of file - total %d lines)", totalLines))
	}
	output.WriteString("\n</file>")

	return &ToolResult{
		Title:  opts.FilePath,
		Output: output.String(),
		Metadata: map[string]interface{}{
			"preview":   preview,
			"filepath":  opts.FilePath,
			"lines":     len(raw),
			"truncated": hasMoreLines,
		},
	}, nil
}

// WriteOptions configures the write tool
type WriteOptions struct {
	FilePath string
	Content  string
}

// WriteTool writes content to a file
func WriteTool(session *Session, opts WriteOptions) (*ToolResult, error) {
	// Check if file already exists
	exists, _ := session.Exists(opts.FilePath)

	// Ensure parent directory exists
	parts := strings.Split(opts.FilePath, "/")
	if len(parts) > 1 {
		parentDir := strings.Join(parts[:len(parts)-1], "/")
		if parentDir != "" {
			parentExists, _ := session.Exists(parentDir)
			if !parentExists {
				if err := session.Mkdir(parentDir, true); err != nil {
					return nil, fmt.Errorf("failed to create parent directory: %w", err)
				}
			}
		}
	}

	// Write the file
	if err := session.WriteFile(opts.FilePath, opts.Content); err != nil {
		return nil, err
	}

	output := fmt.Sprintf("File created: %s", opts.FilePath)
	if exists {
		output = fmt.Sprintf("File overwritten: %s", opts.FilePath)
	}

	return &ToolResult{
		Title:  opts.FilePath,
		Output: output,
		Metadata: map[string]interface{}{
			"filepath": opts.FilePath,
			"exists":   exists,
			"size":     len(opts.Content),
		},
	}, nil
}

// EditOptions configures the edit tool
type EditOptions struct {
	FilePath   string
	OldString  string
	NewString  string
	ReplaceAll bool
}

// EditTool edits a file by replacing text
func EditTool(session *Session, opts EditOptions) (*ToolResult, error) {
	if opts.OldString == opts.NewString {
		return nil, fmt.Errorf("oldString and newString must be different")
	}

	// Handle creating new file when oldString is empty
	if opts.OldString == "" {
		// Create parent directory if needed
		parts := strings.Split(opts.FilePath, "/")
		if len(parts) > 1 {
			parentDir := strings.Join(parts[:len(parts)-1], "/")
			if parentDir != "" {
				parentExists, _ := session.Exists(parentDir)
				if !parentExists {
					if err := session.Mkdir(parentDir, true); err != nil {
						return nil, err
					}
				}
			}
		}

		if err := session.WriteFile(opts.FilePath, opts.NewString); err != nil {
			return nil, err
		}

		return &ToolResult{
			Title:  opts.FilePath,
			Output: fmt.Sprintf("File created: %s", opts.FilePath),
			Metadata: map[string]interface{}{
				"filepath":  opts.FilePath,
				"additions": strings.Count(opts.NewString, "\n") + 1,
				"deletions": 0,
			},
		}, nil
	}

	// Check if file exists
	exists, _ := session.Exists(opts.FilePath)
	if !exists {
		return nil, fmt.Errorf("file not found: %s", opts.FilePath)
	}

	isDir, _ := session.IsDirectory(opts.FilePath)
	if isDir {
		return nil, fmt.Errorf("path is a directory, not a file: %s", opts.FilePath)
	}

	// Read current content
	oldContent, err := session.ReadFile(opts.FilePath)
	if err != nil {
		return nil, err
	}

	// Check if oldString exists
	if !strings.Contains(oldContent, opts.OldString) {
		return nil, fmt.Errorf("oldString not found in content")
	}

	// Check for multiple occurrences if not replaceAll
	if !opts.ReplaceAll {
		firstIndex := strings.Index(oldContent, opts.OldString)
		lastIndex := strings.LastIndex(oldContent, opts.OldString)

		if firstIndex != lastIndex {
			return nil, fmt.Errorf("found multiple matches for oldString. Provide more surrounding context or use ReplaceAll")
		}
	}

	// Perform replacement
	var newContent string
	if opts.ReplaceAll {
		newContent = strings.ReplaceAll(oldContent, opts.OldString, opts.NewString)
	} else {
		newContent = strings.Replace(oldContent, opts.OldString, opts.NewString, 1)
	}

	// Write updated content
	if err := session.WriteFile(opts.FilePath, newContent); err != nil {
		return nil, err
	}

	// Count changes
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	additions := 0
	deletions := 0
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		if oldLine != newLine {
			if oldLine != "" {
				deletions++
			}
			if newLine != "" {
				additions++
			}
		}
	}

	return &ToolResult{
		Title:  opts.FilePath,
		Output: fmt.Sprintf("File edited: %s (+%d -%d)", opts.FilePath, additions, deletions),
		Metadata: map[string]interface{}{
			"filepath":  opts.FilePath,
			"additions": additions,
			"deletions": deletions,
		},
	}, nil
}

// GlobOptions configures the glob tool
type GlobOptions struct {
	Pattern string
	Path    string // directory to search in
}

// GlobTool finds files matching a pattern
func GlobTool(session *Session, opts GlobOptions) (*ToolResult, error) {
	searchPath := opts.Path
	if searchPath == "" {
		searchPath = "."
	}

	// Convert glob pattern to find pattern
	findPattern := strings.ReplaceAll(opts.Pattern, "**", "*")

	// Use -name for simple patterns, -path for patterns with directories
	findFlag := "-name"
	if strings.Contains(opts.Pattern, "/") || strings.Contains(opts.Pattern, "**") {
		findFlag = "-path"
	}

	// Find files
	cmd := fmt.Sprintf("find %s -type f %s '%s' 2>/dev/null | head -%d",
		escapePath(searchPath), findFlag, findPattern, DefaultSearchLimit+1)

	output, _, _, err := session.RunSafe(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	truncated := len(files) > DefaultSearchLimit
	if truncated {
		files = files[:DefaultSearchLimit]
	}

	var result strings.Builder
	if len(files) == 0 {
		result.WriteString("No files found")
	} else {
		result.WriteString(strings.Join(files, "\n"))
		if truncated {
			result.WriteString("\n\n(Results are truncated. Consider using a more specific path or pattern.)")
		}
	}

	return &ToolResult{
		Title:  searchPath,
		Output: result.String(),
		Metadata: map[string]interface{}{
			"count":     len(files),
			"truncated": truncated,
		},
	}, nil
}

// GrepOptions configures the grep tool
type GrepOptions struct {
	Pattern string
	Path    string // directory to search in
	Include string // file pattern to include
}

// GrepTool searches for patterns in files
func GrepTool(session *Session, opts GrepOptions) (*ToolResult, error) {
	searchPath := opts.Path
	if searchPath == "" {
		searchPath = "."
	}

	// Escape single quotes in pattern
	escapedPattern := strings.ReplaceAll(opts.Pattern, "'", "'\\''")

	// Build grep command
	var cmd string
	if opts.Include != "" {
		// Use find + grep for file filtering
		cmd = fmt.Sprintf("find %s -type f -name '%s' -exec grep -nH '%s' {} \\; 2>/dev/null | head -%d",
			escapePath(searchPath), opts.Include, escapedPattern, DefaultSearchLimit+1)
	} else {
		cmd = fmt.Sprintf("grep -rnH '%s' %s 2>/dev/null | head -%d",
			escapedPattern, escapePath(searchPath), DefaultSearchLimit+1)
	}

	output, _, _, err := session.RunSafe(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	var matches []string
	for _, line := range lines {
		if line != "" {
			matches = append(matches, line)
		}
	}

	if len(matches) == 0 {
		return &ToolResult{
			Title:  opts.Pattern,
			Output: "No matches found",
			Metadata: map[string]interface{}{
				"matches":   0,
				"truncated": false,
			},
		}, nil
	}

	truncated := len(matches) > DefaultSearchLimit
	if truncated {
		matches = matches[:DefaultSearchLimit]
	}

	// Format output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d matches\n", len(matches)))

	currentFile := ""
	for _, line := range matches {
		// Parse format: filename:linenum:content
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			continue
		}

		file := line[:colonIndex]
		rest := line[colonIndex+1:]

		secondColonIndex := strings.Index(rest, ":")
		if secondColonIndex == -1 {
			continue
		}

		lineNum := rest[:secondColonIndex]
		content := rest[secondColonIndex+1:]

		if currentFile != file {
			if currentFile != "" {
				result.WriteString("\n")
			}
			currentFile = file
			result.WriteString(fmt.Sprintf("%s:\n", file))
		}
		result.WriteString(fmt.Sprintf("  Line %s: %s\n", lineNum, content))
	}

	if truncated {
		result.WriteString("\n(Results are truncated. Consider using a more specific path or pattern.)")
	}

	return &ToolResult{
		Title:  opts.Pattern,
		Output: result.String(),
		Metadata: map[string]interface{}{
			"matches":   len(matches),
			"truncated": truncated,
		},
	}, nil
}

// LsOptions configures the ls tool
type LsOptions struct {
	Path string
	All  bool // show hidden files
	Long bool // show detailed information
}

// LsTool lists directory contents
func LsTool(session *Session, opts LsOptions) (*ToolResult, error) {
	searchPath := opts.Path
	if searchPath == "" {
		searchPath = "."
	}

	entries, err := session.Ls(searchPath, opts.All, opts.Long)
	if err != nil {
		return nil, err
	}

	// For long format, first line might be "total X"
	startIndex := 0
	if opts.Long && len(entries) > 0 && strings.HasPrefix(entries[0], "total ") {
		startIndex = 1
	}

	entries = entries[startIndex:]

	truncated := len(entries) > DefaultSearchLimit
	if truncated {
		entries = entries[:DefaultSearchLimit]
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s/\n", searchPath))
	result.WriteString(strings.Join(entries, "\n"))

	if truncated {
		result.WriteString("\n\n(Results are truncated. Consider using a more specific path.)")
	}

	return &ToolResult{
		Title:  searchPath,
		Output: result.String(),
		Metadata: map[string]interface{}{
			"count":     len(entries),
			"truncated": truncated,
		},
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
