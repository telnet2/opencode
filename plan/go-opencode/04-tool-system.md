# Phase 4: Tool System (Weeks 7-8)

## Overview

Implement the tool framework and all core tools. The tool system is central to OpenCode's functionality, enabling the LLM to interact with files, execute commands, and perform searches.

---

## 4.1 Tool Framework

### Tool Interface

```go
// internal/tool/tool.go
package tool

import (
    "context"
    "encoding/json"
)

// Tool defines the interface for all tools
type Tool interface {
    ID() string
    Description() string
    Parameters() json.RawMessage // JSON Schema
    Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error)
}

// Context provides execution context to tools
type Context struct {
    SessionID string
    MessageID string
    CallID    string
    Agent     string
    Abort     context.Context
    Extra     map[string]any

    // Metadata callback for real-time updates
    metadata func(title string, meta map[string]any)
}

// SetMetadata updates tool execution metadata
func (c *Context) SetMetadata(title string, meta map[string]any) {
    if c.metadata != nil {
        c.metadata(title, meta)
    }
}

// Result represents the output of a tool execution
type Result struct {
    Title       string            `json:"title"`
    Output      string            `json:"output"`
    Metadata    map[string]any    `json:"metadata,omitempty"`
    Attachments []Attachment      `json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
    Filename  string `json:"filename"`
    MediaType string `json:"mediaType"`
    URL       string `json:"url"` // data: URL or file path
}
```

### Tool Registry

```go
// internal/tool/registry.go
package tool

import (
    "fmt"
    "sync"
)

// Registry manages tool registration and lookup
type Registry struct {
    mu    sync.RWMutex
    tools map[string]Tool
}

func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]Tool),
    }
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[tool.ID()] = tool
}

// Get retrieves a tool by ID
func (r *Registry) Get(id string) (Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    tool, ok := r.tools[id]
    return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tools := make([]Tool, 0, len(r.tools))
    for _, tool := range r.tools {
        tools = append(tools, tool)
    }
    return tools
}

// Enabled returns tools enabled for a given agent
func (r *Registry) Enabled(agent *Agent) map[string]Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    enabled := make(map[string]Tool)
    for id, tool := range r.tools {
        if agent.ToolEnabled(id) {
            enabled[id] = tool
        }
    }
    return enabled
}

// DefaultRegistry creates a registry with all built-in tools
func DefaultRegistry(workDir string, checker *permission.Checker) *Registry {
    r := NewRegistry()

    r.Register(NewReadTool(workDir))
    r.Register(NewWriteTool(workDir))
    r.Register(NewEditTool(workDir))
    r.Register(NewBashTool(workDir, checker))
    r.Register(NewGlobTool(workDir))
    r.Register(NewGrepTool(workDir))
    r.Register(NewListTool(workDir))
    r.Register(NewWebFetchTool())
    r.Register(NewTodoWriteTool())
    r.Register(NewTodoReadTool())

    return r
}
```

---

## 4.2 Read Tool

```go
// internal/tool/read.go
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
)

type ReadTool struct {
    workDir string
}

type ReadInput struct {
    FilePath string `json:"file_path"`
    Offset   int    `json:"offset,omitempty"`
    Limit    int    `json:"limit,omitempty"`
}

func NewReadTool(workDir string) *ReadTool {
    return &ReadTool{workDir: workDir}
}

func (t *ReadTool) ID() string          { return "read" }
func (t *ReadTool) Description() string { return readDescription }

func (t *ReadTool) Parameters() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "file_path": {
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
        "required": ["file_path"]
    }`)
}

func (t *ReadTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
    var params ReadInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    // Default limit
    if params.Limit <= 0 {
        params.Limit = 2000
    }

    // Block .env files
    if strings.HasSuffix(params.FilePath, ".env") {
        return nil, fmt.Errorf(".env files cannot be read for security reasons")
    }

    // Check if file exists
    info, err := os.Stat(params.FilePath)
    if err != nil {
        return nil, fmt.Errorf("file not found: %s", params.FilePath)
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
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        if lineNum < params.Offset {
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
        lines = append(lines, fmt.Sprintf("%5d\t%s", lineNum, line))
    }

    output := strings.Join(lines, "\n")
    if lineNum > params.Offset+params.Limit {
        output += fmt.Sprintf("\n\n(File has more lines. Use offset to read more.)")
    }

    return &Result{
        Title:  fmt.Sprintf("Read %s", filepath.Base(params.FilePath)),
        Output: output,
        Metadata: map[string]any{
            "file":      params.FilePath,
            "lines":     len(lines),
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
```

---

## 4.3 Write Tool

```go
// internal/tool/write.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/opencode-ai/opencode-server/internal/event"
)

type WriteTool struct {
    workDir string
}

type WriteInput struct {
    FilePath string `json:"file_path"`
    Content  string `json:"content"`
}

func NewWriteTool(workDir string) *WriteTool {
    return &WriteTool{workDir: workDir}
}

func (t *WriteTool) ID() string          { return "write" }
func (t *WriteTool) Description() string { return writeDescription }

func (t *WriteTool) Parameters() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "file_path": {
                "type": "string",
                "description": "The absolute path to the file to write"
            },
            "content": {
                "type": "string",
                "description": "The content to write to the file"
            }
        },
        "required": ["file_path", "content"]
    }`)
}

func (t *WriteTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
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

    // Publish file edited event
    event.Publish(event.Event{
        Type: event.FileEdited,
        Data: map[string]any{
            "file":      params.FilePath,
            "sessionID": toolCtx.SessionID,
        },
    })

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
```

---

## 4.4 Edit Tool

The edit tool is the most complex, requiring fuzzy string matching.

```go
// internal/tool/edit.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strings"

    "github.com/opencode-ai/opencode-server/internal/event"
)

type EditTool struct {
    workDir string
}

type EditInput struct {
    FilePath   string `json:"file_path"`
    OldString  string `json:"old_string"`
    NewString  string `json:"new_string"`
    ReplaceAll bool   `json:"replace_all,omitempty"`
}

func NewEditTool(workDir string) *EditTool {
    return &EditTool{workDir: workDir}
}

func (t *EditTool) ID() string          { return "edit" }
func (t *EditTool) Description() string { return editDescription }

func (t *EditTool) Parameters() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "file_path": {
                "type": "string",
                "description": "The absolute path to the file to edit"
            },
            "old_string": {
                "type": "string",
                "description": "The exact text to replace"
            },
            "new_string": {
                "type": "string",
                "description": "The text to replace it with"
            },
            "replace_all": {
                "type": "boolean",
                "description": "Replace all occurrences (default: false)"
            }
        },
        "required": ["file_path", "old_string", "new_string"]
    }`)
}

func (t *EditTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
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
        if strings.Contains(text, params.OldString) {
            count = 1
            newText = strings.Replace(text, params.OldString, params.NewString, 1)
        } else {
            return t.fuzzyReplace(text, params, toolCtx)
        }
    }

    // Write file
    if err := os.WriteFile(params.FilePath, []byte(newText), 0644); err != nil {
        return nil, fmt.Errorf("failed to write file: %w", err)
    }

    // Publish event
    event.Publish(event.Event{
        Type: event.FileEdited,
        Data: map[string]any{
            "file":      params.FilePath,
            "sessionID": toolCtx.SessionID,
        },
    })

    return &Result{
        Title:  fmt.Sprintf("Edited %s", filepath.Base(params.FilePath)),
        Output: fmt.Sprintf("Replaced %d occurrence(s)", count),
        Metadata: map[string]any{
            "file":         params.FilePath,
            "replacements": count,
        },
    }, nil
}

// fuzzyReplace attempts to find similar text when exact match fails
func (t *EditTool) fuzzyReplace(text string, params EditInput, toolCtx Context) (*Result, error) {
    // Try line-normalized matching
    normalizedOld := normalizeLineEndings(params.OldString)
    normalizedText := normalizeLineEndings(text)

    if strings.Contains(normalizedText, normalizedOld) {
        newText := strings.Replace(normalizedText, normalizedOld, params.NewString, 1)
        if err := os.WriteFile(params.FilePath, []byte(newText), 0644); err != nil {
            return nil, fmt.Errorf("failed to write file: %w", err)
        }
        return &Result{
            Title:  fmt.Sprintf("Edited %s (normalized)", filepath.Base(params.FilePath)),
            Output: "Replaced 1 occurrence (with line ending normalization)",
        }, nil
    }

    // Try fuzzy matching with Levenshtein distance
    match, similarity := findBestMatch(text, params.OldString)
    if match != "" && similarity >= 0.7 {
        newText := strings.Replace(text, match, params.NewString, 1)
        if err := os.WriteFile(params.FilePath, []byte(newText), 0644); err != nil {
            return nil, fmt.Errorf("failed to write file: %w", err)
        }
        return &Result{
            Title:  fmt.Sprintf("Edited %s (fuzzy)", filepath.Base(params.FilePath)),
            Output: fmt.Sprintf("Replaced 1 occurrence (%.0f%% similarity)", similarity*100),
        }, nil
    }

    return nil, fmt.Errorf("old_string not found in file. The content may have changed or the string doesn't exist.")
}

func normalizeLineEndings(s string) string {
    return strings.ReplaceAll(s, "\r\n", "\n")
}

// findBestMatch finds the substring most similar to target
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

// similarity calculates normalized Levenshtein similarity
func similarity(a, b string) float64 {
    dist := levenshtein(a, b)
    maxLen := max(len(a), len(b))
    if maxLen == 0 {
        return 1.0
    }
    return 1.0 - float64(dist)/float64(maxLen)
}

// levenshtein calculates edit distance between two strings
func levenshtein(a, b string) int {
    if len(a) == 0 {
        return len(b)
    }
    if len(b) == 0 {
        return len(a)
    }

    // Create distance matrix
    d := make([][]int, len(a)+1)
    for i := range d {
        d[i] = make([]int, len(b)+1)
        d[i][0] = i
    }
    for j := range d[0] {
        d[0][j] = j
    }

    for i := 1; i <= len(a); i++ {
        for j := 1; j <= len(b); j++ {
            cost := 1
            if a[i-1] == b[j-1] {
                cost = 0
            }
            d[i][j] = min(
                d[i-1][j]+1,      // deletion
                d[i][j-1]+1,      // insertion
                d[i-1][j-1]+cost, // substitution
            )
        }
    }

    return d[len(a)][len(b)]
}
```

---

## 4.5 Bash Tool

```go
// internal/tool/bash.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "strings"
    "syscall"
    "time"

    "github.com/opencode-ai/opencode-server/internal/permission"
)

const (
    DefaultTimeout    = 60 * time.Second
    MaxTimeout        = 10 * time.Minute
    MaxOutputLength   = 30000
    SigkillTimeout    = 200 * time.Millisecond
)

type BashTool struct {
    workDir     string
    permChecker *permission.Checker
    shell       string
}

type BashInput struct {
    Command     string `json:"command"`
    Timeout     int    `json:"timeout,omitempty"` // milliseconds
    Description string `json:"description"`
}

func NewBashTool(workDir string, checker *permission.Checker) *BashTool {
    shell := detectShell()
    return &BashTool{
        workDir:     workDir,
        permChecker: checker,
        shell:       shell,
    }
}

func detectShell() string {
    if s := os.Getenv("SHELL"); s != "" {
        // Exclude unsupported shells
        if s != "/bin/fish" && s != "/usr/bin/fish" &&
           s != "/bin/nu" && s != "/usr/bin/nu" {
            return s
        }
    }

    if runtime.GOOS == "darwin" {
        return "/bin/zsh"
    }
    if runtime.GOOS == "windows" {
        if comspec := os.Getenv("COMSPEC"); comspec != "" {
            return comspec
        }
        return "cmd.exe"
    }

    if bash, err := exec.LookPath("bash"); err == nil {
        return bash
    }

    return "/bin/sh"
}

func (t *BashTool) ID() string          { return "bash" }
func (t *BashTool) Description() string { return bashDescription }

func (t *BashTool) Parameters() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "command": {
                "type": "string",
                "description": "The command to execute"
            },
            "timeout": {
                "type": "integer",
                "description": "Optional timeout in milliseconds"
            },
            "description": {
                "type": "string",
                "description": "Brief description of what this command does"
            }
        },
        "required": ["command", "description"]
    }`)
}

func (t *BashTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
    var params BashInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    // Check permissions
    if err := t.checkPermissions(ctx, params.Command, toolCtx); err != nil {
        return nil, err
    }

    // Calculate timeout
    timeout := DefaultTimeout
    if params.Timeout > 0 {
        timeout = time.Duration(params.Timeout) * time.Millisecond
        if timeout > MaxTimeout {
            timeout = MaxTimeout
        }
    }

    // Create command
    cmd := exec.CommandContext(ctx, t.shell, "-c", params.Command)
    cmd.Dir = t.workDir
    cmd.Env = os.Environ()

    // Set process group for Unix (allows killing child processes)
    if runtime.GOOS != "windows" {
        cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    }

    // Capture output
    var output strings.Builder
    cmd.Stdout = &output
    cmd.Stderr = &output

    // Initialize metadata
    toolCtx.SetMetadata(params.Description, map[string]any{
        "output":      "",
        "description": params.Description,
    })

    // Start command
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("failed to start command: %w", err)
    }

    // Setup timeout
    timer := time.AfterFunc(timeout, func() {
        t.killProcess(cmd)
    })
    defer timer.Stop()

    // Wait for completion
    err := cmd.Wait()
    timedOut := !timer.Stop() && err != nil

    // Truncate output if needed
    result := output.String()
    if len(result) > MaxOutputLength {
        result = result[:MaxOutputLength] + "\n\n(Output truncated)"
    }

    if timedOut {
        result += fmt.Sprintf("\n\n(Command timed out after %v)", timeout)
    }

    exitCode := 0
    if cmd.ProcessState != nil {
        exitCode = cmd.ProcessState.ExitCode()
    }

    return &Result{
        Title:  params.Description,
        Output: result,
        Metadata: map[string]any{
            "output":      result,
            "exit":        exitCode,
            "description": params.Description,
        },
    }, nil
}

func (t *BashTool) killProcess(cmd *exec.Cmd) {
    if cmd.Process == nil {
        return
    }

    pid := cmd.Process.Pid

    if runtime.GOOS == "windows" {
        exec.Command("taskkill", "/pid", fmt.Sprint(pid), "/f", "/t").Run()
        return
    }

    // Kill process group
    syscall.Kill(-pid, syscall.SIGTERM)
    time.Sleep(SigkillTimeout)

    // Force kill if still running
    if cmd.ProcessState == nil {
        syscall.Kill(-pid, syscall.SIGKILL)
    }
}
```

---

## 4.6 Glob Tool

```go
// internal/tool/glob.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
)

type GlobTool struct {
    workDir string
}

type GlobInput struct {
    Pattern string `json:"pattern"`
    Path    string `json:"path,omitempty"`
}

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

func (t *GlobTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
    var params GlobInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    searchDir := t.workDir
    if params.Path != "" {
        if filepath.IsAbs(params.Path) {
            searchDir = params.Path
        } else {
            searchDir = filepath.Join(t.workDir, params.Path)
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
            }, nil
        }
    }

    files := strings.Split(strings.TrimSpace(string(output)), "\n")

    // Limit results
    const maxFiles = 100
    truncated := false
    if len(files) > maxFiles {
        files = files[:maxFiles]
        truncated = true
    }

    // Sort by modification time (most recent first)
    sortByModTime(searchDir, files)

    result := strings.Join(files, "\n")
    if truncated {
        result += fmt.Sprintf("\n\n(Showing %d of more files)", maxFiles)
    }

    return &Result{
        Title:  fmt.Sprintf("Found %d files", len(files)),
        Output: result,
        Metadata: map[string]any{
            "pattern": params.Pattern,
            "count":   len(files),
        },
    }, nil
}
```

---

## 4.7 Grep Tool

```go
// internal/tool/grep.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
)

type GrepTool struct {
    workDir string
}

type GrepInput struct {
    Pattern string `json:"pattern"`
    Path    string `json:"path,omitempty"`
    Include string `json:"include,omitempty"` // glob filter
}

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
                "description": "The regex pattern to search for"
            },
            "path": {
                "type": "string",
                "description": "File or directory to search"
            },
            "include": {
                "type": "string",
                "description": "Glob pattern to filter files"
            }
        },
        "required": ["pattern"]
    }`)
}

func (t *GrepTool) Execute(ctx context.Context, input json.RawMessage, toolCtx Context) (*Result, error) {
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
    if params.Path != "" {
        searchPath = params.Path
    }
    args = append(args, searchPath)

    cmd := exec.CommandContext(ctx, "rg", args...)
    output, err := cmd.Output()

    if err != nil {
        if len(output) == 0 {
            return &Result{
                Title:  "Search results",
                Output: "No matches found",
            }, nil
        }
    }

    lines := strings.Split(strings.TrimSpace(string(output)), "\n")

    // Limit results
    const maxMatches = 100
    truncated := false
    if len(lines) > maxMatches {
        lines = lines[:maxMatches]
        truncated = true
    }

    result := strings.Join(lines, "\n")
    if truncated {
        result += fmt.Sprintf("\n\n(Showing %d of more matches)", maxMatches)
    }

    return &Result{
        Title:  fmt.Sprintf("Found %d matches", len(lines)),
        Output: result,
        Metadata: map[string]any{
            "pattern": params.Pattern,
            "count":   len(lines),
        },
    }, nil
}
```

---

## 4.8 Deliverables

### Files to Create

| File | Lines (Est.) | Complexity |
|------|--------------|------------|
| `internal/tool/tool.go` | 80 | Low |
| `internal/tool/registry.go` | 100 | Low |
| `internal/tool/read.go` | 200 | Medium |
| `internal/tool/write.go` | 80 | Low |
| `internal/tool/edit.go` | 300 | High |
| `internal/tool/bash.go` | 250 | High |
| `internal/tool/glob.go` | 100 | Low |
| `internal/tool/grep.go` | 100 | Low |
| `internal/tool/list.go` | 100 | Low |
| `internal/tool/webfetch.go` | 150 | Medium |
| `internal/tool/todo.go` | 80 | Low |

### Tests

```go
// test/integration/tool_test.go

func TestReadTool_TextFile(t *testing.T) { /* ... */ }
func TestReadTool_BinaryDetection(t *testing.T) { /* ... */ }
func TestReadTool_ImageFile(t *testing.T) { /* ... */ }
func TestReadTool_EnvBlocking(t *testing.T) { /* ... */ }

func TestWriteTool_NewFile(t *testing.T) { /* ... */ }
func TestWriteTool_Overwrite(t *testing.T) { /* ... */ }
func TestWriteTool_CreateDirs(t *testing.T) { /* ... */ }

func TestEditTool_ExactMatch(t *testing.T) { /* ... */ }
func TestEditTool_FuzzyMatch(t *testing.T) { /* ... */ }
func TestEditTool_ReplaceAll(t *testing.T) { /* ... */ }
func TestEditTool_LineEndings(t *testing.T) { /* ... */ }

func TestBashTool_Execute(t *testing.T) { /* ... */ }
func TestBashTool_Timeout(t *testing.T) { /* ... */ }
func TestBashTool_Abort(t *testing.T) { /* ... */ }
func TestBashTool_OutputTruncation(t *testing.T) { /* ... */ }

func TestGlobTool_Pattern(t *testing.T) { /* ... */ }
func TestGrepTool_Regex(t *testing.T) { /* ... */ }
```

### Acceptance Criteria

- [ ] All 11 core tools implemented
- [ ] Tool registry supports dynamic registration
- [ ] Edit tool passes fuzzy matching tests
- [ ] Bash tool respects timeout and abort signals
- [ ] Read tool detects binary files correctly
- [ ] Glob/Grep use ripgrep for performance
- [ ] All tools emit appropriate events
- [ ] Test coverage >85% for tool package
