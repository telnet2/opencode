package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/opencode-ai/opencode/internal/permission"
)

const (
	DefaultBashTimeout = 120 * time.Second
	MaxBashTimeout     = 10 * time.Minute
	MaxOutputLength    = 30000
	SigkillTimeout     = 200 * time.Millisecond
)

const bashDescription = `Executes a bash command in a persistent shell session.

Usage:
- Command is required
- Optional timeout in milliseconds (max 600000)
- Provide a brief description of what the command does
- Output is captured from stdout and stderr
- Commands are run with process group for proper cleanup`

// BashTool implements shell command execution.
type BashTool struct {
	workDir     string
	shell       string
	permChecker *permission.Checker
	permissions map[string]permission.PermissionAction // bash command patterns
	externalDir permission.PermissionAction           // action for external directory access
}

// BashInput represents the input for the bash tool.
type BashInput struct {
	Command     string `json:"command"`
	Timeout     int    `json:"timeout,omitempty"` // milliseconds
	Description string `json:"description"`
}

// BashToolOption configures the bash tool.
type BashToolOption func(*BashTool)

// WithPermissionChecker sets the permission checker for the bash tool.
func WithPermissionChecker(checker *permission.Checker) BashToolOption {
	return func(t *BashTool) {
		t.permChecker = checker
	}
}

// WithBashPermissions sets the bash command permission patterns.
func WithBashPermissions(perms map[string]permission.PermissionAction) BashToolOption {
	return func(t *BashTool) {
		t.permissions = perms
	}
}

// WithExternalDirAction sets the action for external directory access.
func WithExternalDirAction(action permission.PermissionAction) BashToolOption {
	return func(t *BashTool) {
		t.externalDir = action
	}
}

// NewBashTool creates a new bash tool.
func NewBashTool(workDir string, opts ...BashToolOption) *BashTool {
	shell := detectShell()
	t := &BashTool{
		workDir:     workDir,
		shell:       shell,
		permissions: make(map[string]permission.PermissionAction),
		externalDir: permission.ActionAsk,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
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
				"description": "Optional timeout in milliseconds (max 600000)"
			},
			"description": {
				"type": "string",
				"description": "Brief description of what this command does"
			}
		},
		"required": ["command", "description"]
	}`)
}

func (t *BashTool) Execute(ctx context.Context, input json.RawMessage, toolCtx *Context) (*Result, error) {
	var params BashInput
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Check permissions if checker is configured
	if t.permChecker != nil && toolCtx != nil {
		if err := t.checkPermissions(ctx, params.Command, toolCtx); err != nil {
			return nil, err
		}
	}

	// Calculate timeout
	timeout := DefaultBashTimeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Millisecond
		if timeout > MaxBashTimeout {
			timeout = MaxBashTimeout
		}
	}

	// Create command with context
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, t.shell, "/c", params.Command)
	} else {
		cmd = exec.CommandContext(cmdCtx, t.shell, "-c", params.Command)
	}

	// Set working directory
	if toolCtx != nil && toolCtx.WorkDir != "" {
		cmd.Dir = toolCtx.WorkDir
	} else if t.workDir != "" {
		cmd.Dir = t.workDir
	}

	cmd.Env = os.Environ()

	// Set process group for Unix (allows killing child processes)
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// Initialize metadata
	if toolCtx != nil {
		toolCtx.SetMetadata(params.Description, map[string]any{
			"output":      "",
			"description": params.Description,
		})
	}

	// Run command and capture output
	output, err := cmd.CombinedOutput()
	timedOut := cmdCtx.Err() == context.DeadlineExceeded

	// Truncate output if needed
	result := string(output)
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

	// Add error message if command failed
	if err != nil && !timedOut {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			result += fmt.Sprintf("\n\nError: %v", err)
		}
	}

	title := params.Description
	if title == "" {
		title = "Run command"
	}

	return &Result{
		Title:  title,
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

func (t *BashTool) EinoTool() einotool.InvokableTool {
	return &einoToolWrapper{tool: t}
}

// checkPermissions validates bash command permissions.
func (t *BashTool) checkPermissions(ctx context.Context, command string, toolCtx *Context) error {
	// Parse the command
	commands, err := permission.ParseBashCommand(command)
	if err != nil {
		// If we can't parse, default to asking
		return t.permChecker.Ask(ctx, permission.Request{
			Type:      permission.PermBash,
			Pattern:   []string{command},
			SessionID: toolCtx.SessionID,
			MessageID: toolCtx.MessageID,
			CallID:    toolCtx.CallID,
			Title:     command,
			Metadata: map[string]any{
				"command":      command,
				"parse_failed": true,
			},
		})
	}

	// Determine working directory
	workDir := t.workDir
	if toolCtx.WorkDir != "" {
		workDir = toolCtx.WorkDir
	}

	var askPatterns []string

	for _, cmd := range commands {
		// Check for dangerous commands (file operations)
		if permission.IsDangerousCommand(cmd.Name) {
			paths := permission.ExtractPaths(cmd)
			for _, p := range paths {
				resolved, err := permission.ResolvePath(ctx, p, workDir)
				if err != nil {
					continue
				}

				// Check if path is outside working directory
				if !permission.IsWithinDir(resolved, workDir) {
					switch t.externalDir {
					case permission.ActionDeny:
						return &permission.RejectedError{
							SessionID: toolCtx.SessionID,
							Type:      permission.PermExternalDir,
							CallID:    toolCtx.CallID,
							Message:   fmt.Sprintf("Command references paths outside of %s", workDir),
							Metadata: map[string]any{
								"command": command,
								"path":    resolved,
							},
						}
					case permission.ActionAsk:
						err := t.permChecker.Ask(ctx, permission.Request{
							Type:      permission.PermExternalDir,
							Pattern:   []string{filepath.Dir(resolved), filepath.Join(filepath.Dir(resolved), "*")},
							SessionID: toolCtx.SessionID,
							MessageID: toolCtx.MessageID,
							CallID:    toolCtx.CallID,
							Title:     fmt.Sprintf("Command references paths outside of %s", workDir),
							Metadata: map[string]any{
								"command": command,
								"path":    resolved,
							},
						})
						if err != nil {
							return err
						}
					}
					// ActionAllow - continue
				}
			}
		}

		// Skip "cd" after path validation
		if cmd.Name == "cd" {
			continue
		}

		// Check bash permission patterns
		action := permission.MatchBashPermission(cmd, t.permissions)
		switch action {
		case permission.ActionDeny:
			return &permission.RejectedError{
				SessionID: toolCtx.SessionID,
				Type:      permission.PermBash,
				CallID:    toolCtx.CallID,
				Message:   fmt.Sprintf("Command not allowed: %s", cmd.Name),
				Metadata: map[string]any{
					"command":     command,
					"permissions": t.permissions,
				},
			}
		case permission.ActionAsk:
			// Build pattern for approval
			pattern := permission.BuildPattern(cmd)
			askPatterns = append(askPatterns, pattern)
		}
		// ActionAllow - continue
	}

	// Ask for all collected patterns at once
	if len(askPatterns) > 0 {
		// Deduplicate patterns
		seen := make(map[string]bool)
		uniquePatterns := make([]string, 0, len(askPatterns))
		for _, p := range askPatterns {
			if !seen[p] {
				seen[p] = true
				uniquePatterns = append(uniquePatterns, p)
			}
		}

		return t.permChecker.Ask(ctx, permission.Request{
			Type:      permission.PermBash,
			Pattern:   uniquePatterns,
			SessionID: toolCtx.SessionID,
			MessageID: toolCtx.MessageID,
			CallID:    toolCtx.CallID,
			Title:     command,
			Metadata: map[string]any{
				"command":  command,
				"patterns": uniquePatterns,
			},
		})
	}

	return nil
}
