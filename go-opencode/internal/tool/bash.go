package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
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
	workDir string
	shell   string
}

// BashInput represents the input for the bash tool.
type BashInput struct {
	Command     string `json:"command"`
	Timeout     int    `json:"timeout,omitempty"` // milliseconds
	Description string `json:"description"`
}

// NewBashTool creates a new bash tool.
func NewBashTool(workDir string) *BashTool {
	shell := detectShell()
	return &BashTool{
		workDir: workDir,
		shell:   shell,
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

func (t *BashTool) ID() string          { return "Bash" }
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
