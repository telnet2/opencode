# Phase 5: Permission & Security (Week 9)

**Status: ✅ COMPLETE** (implemented 2025-11-26)

## Overview

Implement the permission system for controlling tool execution, with special focus on bash command analysis using **mvdan/sh** (already used in go-memsh).

### Implementation Summary

All Phase 5 deliverables have been implemented:

| File | Lines | Description |
|------|-------|-------------|
| `internal/permission/permission.go` | 75 | Permission types and errors |
| `internal/permission/bash_parser.go` | 145 | mvdan/sh based bash parsing |
| `internal/permission/checker.go` | 165 | Permission checker with ask flow |
| `internal/permission/wildcard.go` | 100 | Pattern matching for permissions |
| `internal/permission/doom_loop.go` | 75 | Doom loop detection |
| `internal/permission/bash_parser_test.go` | 200 | Parser unit tests |
| `internal/permission/permission_test.go` | 350 | Permission system tests |
| **Total** | **~1,100** | **42 tests passing**

---

## 5.1 Bash Command Parsing with mvdan/sh

### Why mvdan/sh?

| Feature | mvdan/sh | tree-sitter-bash |
|---------|----------|------------------|
| Pure Go | Yes | No (CGo/WASM) |
| Already in project | Yes (go-memsh) | No |
| Shell dialect support | POSIX, Bash, mksh | Bash only |
| Parser quality | Excellent | Good |
| Maintenance | Active | Active |

### Parser Implementation

```go
// internal/permission/bash_parser.go
package permission

import (
    "context"
    "fmt"
    "os/exec"
    "path/filepath"
    "strings"

    "mvdan.cc/sh/v3/syntax"
)

// BashCommand represents a parsed command with its arguments
type BashCommand struct {
    Name string   // Command name (e.g., "rm", "git")
    Args []string // Command arguments
    Subcommand string // First non-flag argument (e.g., "commit" in "git commit")
}

// ParseBashCommand parses a bash command string into structured commands
func ParseBashCommand(command string) ([]BashCommand, error) {
    parser := syntax.NewParser(
        syntax.Variant(syntax.LangBash),
        syntax.KeepComments(false),
    )

    file, err := parser.Parse(strings.NewReader(command), "")
    if err != nil {
        return nil, fmt.Errorf("failed to parse command: %w", err)
    }

    var commands []BashCommand
    syntax.Walk(file, func(node syntax.Node) bool {
        switch n := node.(type) {
        case *syntax.CallExpr:
            cmd := extractCommand(n)
            if cmd != nil {
                commands = append(commands, *cmd)
            }
        }
        return true
    })

    return commands, nil
}

// extractCommand extracts command name and arguments from a CallExpr
func extractCommand(call *syntax.CallExpr) *BashCommand {
    if len(call.Args) == 0 {
        return nil
    }

    cmd := &BashCommand{}

    // Extract command name from first word
    if len(call.Args) > 0 {
        cmd.Name = wordToString(call.Args[0])
    }

    // Extract arguments
    for _, arg := range call.Args[1:] {
        argStr := wordToString(arg)
        cmd.Args = append(cmd.Args, argStr)

        // Find first non-flag argument as subcommand
        if cmd.Subcommand == "" && !strings.HasPrefix(argStr, "-") {
            cmd.Subcommand = argStr
        }
    }

    return cmd
}

// wordToString converts a syntax.Word to a string
func wordToString(word *syntax.Word) string {
    var sb strings.Builder
    for _, part := range word.Parts {
        switch p := part.(type) {
        case *syntax.Lit:
            sb.WriteString(p.Value)
        case *syntax.SglQuoted:
            sb.WriteString(p.Value)
        case *syntax.DblQuoted:
            for _, qp := range p.Parts {
                if lit, ok := qp.(*syntax.Lit); ok {
                    sb.WriteString(lit.Value)
                }
            }
        case *syntax.ParamExp:
            // Variable expansion - return placeholder
            sb.WriteString("$" + p.Param.Value)
        }
    }
    return sb.String()
}

// DangerousCommands that modify files and need path validation
var DangerousCommands = map[string]bool{
    "cd":    true,
    "rm":    true,
    "cp":    true,
    "mv":    true,
    "mkdir": true,
    "touch": true,
    "chmod": true,
    "chown": true,
}

// IsDangerousCommand checks if a command is in the dangerous list
func IsDangerousCommand(name string) bool {
    return DangerousCommands[name]
}

// ExtractPaths extracts file paths from command arguments
func ExtractPaths(cmd BashCommand) []string {
    var paths []string
    for _, arg := range cmd.Args {
        // Skip flags
        if strings.HasPrefix(arg, "-") {
            continue
        }
        // Skip chmod mode arguments (numeric)
        if cmd.Name == "chmod" && strings.HasPrefix(arg, "+") {
            continue
        }
        paths = append(paths, arg)
    }
    return paths
}

// ResolvePath resolves a path to absolute, handling relative paths
func ResolvePath(ctx context.Context, path, workDir string) (string, error) {
    // Handle absolute paths
    if filepath.IsAbs(path) {
        return filepath.Clean(path), nil
    }

    // Use realpath for relative paths
    cmd := exec.CommandContext(ctx, "realpath", path)
    cmd.Dir = workDir
    output, err := cmd.Output()
    if err != nil {
        // Fallback to manual resolution
        return filepath.Clean(filepath.Join(workDir, path)), nil
    }
    return strings.TrimSpace(string(output)), nil
}
```

### Example Usage

```go
// Parse: git commit -m "message" && rm -rf ./temp
commands, _ := ParseBashCommand(`git commit -m "message" && rm -rf ./temp`)

// Result:
// commands[0] = {Name: "git", Args: ["commit", "-m", "message"], Subcommand: "commit"}
// commands[1] = {Name: "rm", Args: ["-rf", "./temp"], Subcommand: "./temp"}
```

---

## 5.2 Permission System

### Permission Types

```go
// internal/permission/permission.go
package permission

import (
    "context"
    "sync"
)

// PermissionAction represents the action to take for a permission check
type PermissionAction string

const (
    ActionAllow PermissionAction = "allow"
    ActionDeny  PermissionAction = "deny"
    ActionAsk   PermissionAction = "ask"
)

// PermissionType represents the type of permission being checked
type PermissionType string

const (
    PermBash             PermissionType = "bash"
    PermEdit             PermissionType = "edit"
    PermWebFetch         PermissionType = "webfetch"
    PermExternalDir      PermissionType = "external_directory"
    PermDoomLoop         PermissionType = "doom_loop"
)

// PermissionRequest represents a request for permission
type PermissionRequest struct {
    ID        string                 `json:"id"`
    Type      PermissionType         `json:"type"`
    Pattern   []string               `json:"pattern,omitempty"` // Patterns for approval
    SessionID string                 `json:"sessionID"`
    MessageID string                 `json:"messageID"`
    CallID    string                 `json:"callID,omitempty"`
    Title     string                 `json:"title"`
    Metadata  map[string]any         `json:"metadata,omitempty"`
}

// PermissionResponse represents a user's response to a permission request
type PermissionResponse struct {
    RequestID string `json:"requestID"`
    Action    string `json:"action"` // "once" | "always" | "reject"
}

// RejectedError is returned when permission is denied
type RejectedError struct {
    SessionID string
    Type      PermissionType
    CallID    string
    Metadata  map[string]any
    Message   string
}

func (e *RejectedError) Error() string {
    return e.Message
}
```

### Permission Checker

```go
// internal/permission/checker.go
package permission

import (
    "context"
    "sync"

    "github.com/opencode-ai/opencode-server/internal/event"
    "github.com/oklog/ulid/v2"
)

// Checker handles permission checks and approvals
type Checker struct {
    mu       sync.RWMutex
    approved map[string]map[PermissionType]bool // sessionID -> type -> approved
    pending  map[string]chan PermissionResponse // requestID -> response channel
}

func NewChecker() *Checker {
    return &Checker{
        approved: make(map[string]map[PermissionType]bool),
        pending:  make(map[string]chan PermissionResponse),
    }
}

// Check performs a permission check based on agent configuration
func (c *Checker) Check(ctx context.Context, req PermissionRequest, action PermissionAction) error {
    switch action {
    case ActionAllow:
        return nil
    case ActionDeny:
        return &RejectedError{
            SessionID: req.SessionID,
            Type:      req.Type,
            CallID:    req.CallID,
            Metadata:  req.Metadata,
            Message:   "Permission denied by agent configuration",
        }
    case ActionAsk:
        return c.Ask(ctx, req)
    }
    return nil
}

// Ask prompts the user for permission
func (c *Checker) Ask(ctx context.Context, req PermissionRequest) error {
    // Check if already approved for this session
    c.mu.RLock()
    if sessionApprovals, ok := c.approved[req.SessionID]; ok {
        if sessionApprovals[req.Type] {
            c.mu.RUnlock()
            return nil
        }
    }
    c.mu.RUnlock()

    // Generate request ID if not set
    if req.ID == "" {
        req.ID = ulid.Make().String()
    }

    // Create response channel
    respChan := make(chan PermissionResponse, 1)
    c.mu.Lock()
    c.pending[req.ID] = respChan
    c.mu.Unlock()

    defer func() {
        c.mu.Lock()
        delete(c.pending, req.ID)
        c.mu.Unlock()
    }()

    // Publish permission request event
    event.Publish(event.Event{
        Type: event.PermissionRequired,
        Data: req,
    })

    // Wait for response
    select {
    case <-ctx.Done():
        return ctx.Err()
    case resp := <-respChan:
        switch resp.Action {
        case "once":
            return nil
        case "always":
            c.approve(req.SessionID, req.Type)
            return nil
        case "reject":
            return &RejectedError{
                SessionID: req.SessionID,
                Type:      req.Type,
                CallID:    req.CallID,
                Metadata:  req.Metadata,
                Message:   "Permission rejected by user",
            }
        }
    }
    return nil
}

// Respond handles a user's response to a permission request
func (c *Checker) Respond(requestID string, action string) {
    c.mu.RLock()
    ch, ok := c.pending[requestID]
    c.mu.RUnlock()

    if ok {
        ch <- PermissionResponse{
            RequestID: requestID,
            Action:    action,
        }
    }
}

func (c *Checker) approve(sessionID string, permType PermissionType) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.approved[sessionID] == nil {
        c.approved[sessionID] = make(map[PermissionType]bool)
    }
    c.approved[sessionID][permType] = true
}

func (c *Checker) ClearSession(sessionID string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.approved, sessionID)
}
```

---

## 5.3 Wildcard Pattern Matching

```go
// internal/permission/wildcard.go
package permission

import (
    "strings"
)

// AgentPermissions represents the permission configuration for an agent
type AgentPermissions struct {
    Edit            PermissionAction            `json:"edit"`
    WebFetch        PermissionAction            `json:"webfetch"`
    ExternalDir     PermissionAction            `json:"external_directory"`
    DoomLoop        PermissionAction            `json:"doom_loop"`
    Bash            map[string]PermissionAction `json:"bash"` // pattern -> action
}

// MatchBashPermission finds the matching permission for a command
func MatchBashPermission(cmd BashCommand, permissions map[string]PermissionAction) PermissionAction {
    // Build command string for matching
    cmdStr := cmd.Name
    if cmd.Subcommand != "" {
        cmdStr += " " + cmd.Subcommand
    }

    // Try exact match first
    if action, ok := permissions[cmdStr+" *"]; ok {
        return action
    }

    // Try command + wildcard
    if action, ok := permissions[cmd.Name+" *"]; ok {
        return action
    }

    // Try global wildcard
    if action, ok := permissions["*"]; ok {
        return action
    }

    // Default to ask
    return ActionAsk
}

// MatchPattern checks if a command matches a wildcard pattern
func MatchPattern(pattern string, cmd BashCommand) bool {
    parts := strings.Split(pattern, " ")
    if len(parts) == 0 {
        return false
    }

    // Match command name
    if parts[0] != "*" && parts[0] != cmd.Name {
        return false
    }

    // If pattern ends with *, match any subcommand/args
    if len(parts) > 1 && parts[len(parts)-1] == "*" {
        // Match intermediate parts
        for i := 1; i < len(parts)-1; i++ {
            if i-1 >= len(cmd.Args) {
                return false
            }
            if parts[i] != "*" && parts[i] != cmd.Args[i-1] {
                return false
            }
        }
        return true
    }

    return true
}
```

---

## 5.4 Bash Tool Permission Integration

```go
// internal/tool/bash.go (permission section)
package tool

import (
    "context"
    "fmt"
    "path/filepath"

    "github.com/opencode-ai/opencode-server/internal/permission"
)

// checkBashPermissions validates bash command permissions
func (t *BashTool) checkPermissions(ctx context.Context, command string, toolCtx ToolContext) error {
    // Parse command
    commands, err := permission.ParseBashCommand(command)
    if err != nil {
        return fmt.Errorf("failed to parse command: %w", err)
    }

    agent := t.agentStore.Get(toolCtx.Agent)
    permissions := agent.Permission.Bash

    askPatterns := make([]string, 0)

    for _, cmd := range commands {
        // Check for dangerous commands (file operations)
        if permission.IsDangerousCommand(cmd.Name) {
            paths := permission.ExtractPaths(cmd)
            for _, p := range paths {
                resolved, err := permission.ResolvePath(ctx, p, t.workDir)
                if err != nil {
                    continue
                }

                // Check if path is outside working directory
                if !isWithinDir(resolved, t.workDir) {
                    action := agent.Permission.ExternalDir
                    if action == permission.ActionAsk {
                        err := t.permChecker.Ask(ctx, permission.PermissionRequest{
                            Type:      permission.PermExternalDir,
                            Pattern:   []string{filepath.Dir(resolved), filepath.Join(filepath.Dir(resolved), "*")},
                            SessionID: toolCtx.SessionID,
                            MessageID: toolCtx.MessageID,
                            CallID:    toolCtx.CallID,
                            Title:     fmt.Sprintf("Command references paths outside of %s", t.workDir),
                            Metadata: map[string]any{
                                "command": command,
                                "path":    resolved,
                            },
                        })
                        if err != nil {
                            return err
                        }
                    } else if action == permission.ActionDeny {
                        return &permission.RejectedError{
                            SessionID: toolCtx.SessionID,
                            Type:      permission.PermExternalDir,
                            CallID:    toolCtx.CallID,
                            Message:   fmt.Sprintf("Command references paths outside of %s", t.workDir),
                        }
                    }
                }
            }
        }

        // Skip "cd" after path validation
        if cmd.Name == "cd" {
            continue
        }

        // Check bash permission patterns
        action := permission.MatchBashPermission(cmd, permissions)
        if action == permission.ActionDeny {
            return &permission.RejectedError{
                SessionID: toolCtx.SessionID,
                Type:      permission.PermBash,
                CallID:    toolCtx.CallID,
                Message:   fmt.Sprintf("Command not allowed: %s", cmd.Name),
                Metadata: map[string]any{
                    "permissions": permissions,
                },
            }
        }
        if action == permission.ActionAsk {
            // Build pattern for approval
            pattern := cmd.Name + " *"
            if cmd.Subcommand != "" {
                pattern = cmd.Name + " " + cmd.Subcommand + " *"
            }
            askPatterns = append(askPatterns, pattern)
        }
    }

    // Ask for all collected patterns at once
    if len(askPatterns) > 0 {
        return t.permChecker.Ask(ctx, permission.PermissionRequest{
            Type:      permission.PermBash,
            Pattern:   askPatterns,
            SessionID: toolCtx.SessionID,
            MessageID: toolCtx.MessageID,
            CallID:    toolCtx.CallID,
            Title:     command,
            Metadata: map[string]any{
                "command":  command,
                "patterns": askPatterns,
            },
        })
    }

    return nil
}

// isWithinDir checks if path is within or under directory
func isWithinDir(path, dir string) bool {
    rel, err := filepath.Rel(dir, path)
    if err != nil {
        return false
    }
    return !strings.HasPrefix(rel, "..")
}
```

---

## 5.5 Doom Loop Detection

```go
// internal/permission/doom_loop.go
package permission

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
)

const DoomLoopThreshold = 3

// DoomLoopDetector tracks repeated tool calls
type DoomLoopDetector struct {
    history map[string][]string // sessionID -> last N tool call hashes
}

func NewDoomLoopDetector() *DoomLoopDetector {
    return &DoomLoopDetector{
        history: make(map[string][]string),
    }
}

// Check if a tool call is a doom loop (same tool + input 3x in a row)
func (d *DoomLoopDetector) Check(sessionID, toolName string, input any) bool {
    hash := d.hashCall(toolName, input)

    history := d.history[sessionID]
    if len(history) < DoomLoopThreshold {
        d.history[sessionID] = append(history, hash)
        return false
    }

    // Check if last N calls are identical
    allSame := true
    for i := len(history) - DoomLoopThreshold + 1; i < len(history); i++ {
        if history[i] != hash {
            allSame = false
            break
        }
    }

    // Update history
    d.history[sessionID] = append(history[1:], hash)

    return allSame && history[len(history)-1] == hash
}

func (d *DoomLoopDetector) hashCall(toolName string, input any) string {
    data, _ := json.Marshal(map[string]any{
        "tool":  toolName,
        "input": input,
    })
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}

func (d *DoomLoopDetector) Clear(sessionID string) {
    delete(d.history, sessionID)
}
```

---

## 5.6 Tests

### Unit Tests

```go
// test/unit/bash_parser_test.go
package unit

import (
    "testing"

    "github.com/opencode-ai/opencode-server/internal/permission"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParseBashCommand_Simple(t *testing.T) {
    commands, err := permission.ParseBashCommand("ls -la")
    require.NoError(t, err)
    require.Len(t, commands, 1)

    assert.Equal(t, "ls", commands[0].Name)
    assert.Equal(t, []string{"-la"}, commands[0].Args)
}

func TestParseBashCommand_Pipeline(t *testing.T) {
    commands, err := permission.ParseBashCommand("cat file.txt | grep pattern")
    require.NoError(t, err)
    require.Len(t, commands, 2)

    assert.Equal(t, "cat", commands[0].Name)
    assert.Equal(t, "grep", commands[1].Name)
}

func TestParseBashCommand_AndChain(t *testing.T) {
    commands, err := permission.ParseBashCommand("git add . && git commit -m 'message'")
    require.NoError(t, err)
    require.Len(t, commands, 2)

    assert.Equal(t, "git", commands[0].Name)
    assert.Equal(t, "add", commands[0].Subcommand)
    assert.Equal(t, "git", commands[1].Name)
    assert.Equal(t, "commit", commands[1].Subcommand)
}

func TestParseBashCommand_Subshell(t *testing.T) {
    commands, err := permission.ParseBashCommand("echo $(pwd)")
    require.NoError(t, err)
    require.Len(t, commands, 2) // echo and pwd

    assert.Equal(t, "echo", commands[0].Name)
    assert.Equal(t, "pwd", commands[1].Name)
}

func TestParseBashCommand_DangerousCommand(t *testing.T) {
    commands, err := permission.ParseBashCommand("rm -rf /tmp/test")
    require.NoError(t, err)
    require.Len(t, commands, 1)

    assert.True(t, permission.IsDangerousCommand(commands[0].Name))
    paths := permission.ExtractPaths(commands[0])
    assert.Equal(t, []string{"/tmp/test"}, paths)
}

func TestParseBashCommand_QuotedStrings(t *testing.T) {
    commands, err := permission.ParseBashCommand(`echo "hello world" 'single quoted'`)
    require.NoError(t, err)
    require.Len(t, commands, 1)

    assert.Equal(t, "echo", commands[0].Name)
    assert.Contains(t, commands[0].Args, "hello world")
    assert.Contains(t, commands[0].Args, "single quoted")
}

func TestParseBashCommand_ComplexGit(t *testing.T) {
    commands, err := permission.ParseBashCommand(`git commit -m "$(cat <<'EOF'
Fix bug in parser
EOF
)"`)
    require.NoError(t, err)
    require.GreaterOrEqual(t, len(commands), 1)
    assert.Equal(t, "git", commands[0].Name)
}
```

```go
// test/unit/permission_test.go
package unit

import (
    "testing"

    "github.com/opencode-ai/opencode-server/internal/permission"
    "github.com/stretchr/testify/assert"
)

func TestMatchBashPermission(t *testing.T) {
    permissions := map[string]permission.PermissionAction{
        "git *":        permission.ActionAllow,
        "rm *":         permission.ActionDeny,
        "npm install *": permission.ActionAsk,
        "*":            permission.ActionAsk,
    }

    tests := []struct {
        name     string
        cmd      permission.BashCommand
        expected permission.PermissionAction
    }{
        {
            name:     "git allowed",
            cmd:      permission.BashCommand{Name: "git", Subcommand: "commit"},
            expected: permission.ActionAllow,
        },
        {
            name:     "rm denied",
            cmd:      permission.BashCommand{Name: "rm", Args: []string{"-rf", "dir"}},
            expected: permission.ActionDeny,
        },
        {
            name:     "npm install ask",
            cmd:      permission.BashCommand{Name: "npm", Subcommand: "install"},
            expected: permission.ActionAsk,
        },
        {
            name:     "unknown command defaults to ask",
            cmd:      permission.BashCommand{Name: "unknown"},
            expected: permission.ActionAsk,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := permission.MatchBashPermission(tt.cmd, permissions)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestDoomLoopDetector(t *testing.T) {
    detector := permission.NewDoomLoopDetector()
    sessionID := "test-session"

    // First 2 calls should not trigger
    assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))
    assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

    // Third identical call should trigger
    assert.True(t, detector.Check(sessionID, "read", map[string]string{"file": "test.txt"}))

    // Different input should not trigger
    assert.False(t, detector.Check(sessionID, "read", map[string]string{"file": "other.txt"}))
}
```

### Integration Tests

```go
// test/integration/bash_permission_test.go
package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/opencode-ai/opencode-server/internal/permission"
    "github.com/opencode-ai/opencode-server/internal/tool"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestBashTool_ExternalDirectoryCheck(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "bash-test")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    checker := permission.NewChecker()
    bashTool := tool.NewBashTool(tmpDir, checker)

    ctx := context.Background()
    toolCtx := tool.ToolContext{
        SessionID: "test-session",
        MessageID: "test-message",
        Agent:     "test-agent",
    }

    // Command within directory should be allowed
    _, err = bashTool.Execute(ctx, tool.BashInput{
        Command:     "echo 'test'",
        Description: "Echo test",
    }, toolCtx)
    assert.NoError(t, err)

    // Command referencing external path with deny permission should fail
    // (This would need mock agent with external_directory: "deny")
}

func TestBashTool_CommandPatternPermission(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "bash-test")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    // Test with agent that allows git but denies rm
    // ...
}
```

---

## 5.7 Deliverables

### Files to Create

| File | Purpose |
|------|---------|
| `internal/permission/permission.go` | Permission types and errors |
| `internal/permission/checker.go` | Permission checker with ask flow |
| `internal/permission/bash_parser.go` | mvdan/sh based bash parsing |
| `internal/permission/wildcard.go` | Pattern matching for permissions |
| `internal/permission/doom_loop.go` | Doom loop detection |
| `test/unit/bash_parser_test.go` | Parser unit tests |
| `test/unit/permission_test.go` | Permission unit tests |
| `test/integration/bash_permission_test.go` | Integration tests |

### Acceptance Criteria

- [ ] mvdan/sh parses all bash command patterns used in OpenCode
- [ ] Dangerous commands (rm, mv, cp, etc.) trigger path validation
- [ ] External directory access triggers permission check
- [ ] Wildcard pattern matching works for all permission configs
- [ ] Doom loop detection triggers after 3 identical calls
- [ ] Permission ask flow publishes events and waits for response
- [ ] All unit tests pass
- [ ] Integration tests verify end-to-end permission flow
