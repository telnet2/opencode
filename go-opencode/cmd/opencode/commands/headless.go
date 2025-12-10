package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/opencode-ai/opencode/internal/headless"
	"github.com/spf13/cobra"
)

var (
	// Headless mode flags
	headlessPrompt       string
	headlessWorkDir      string
	headlessAutoApprove  bool
	headlessOutputFormat string
	headlessTimeout      string
	headlessMaxSteps     int
	headlessStdin        bool
	headlessNoSave       bool
	headlessSessionID    string
	headlessContinue     bool
	headlessFiles        []string
	headlessSystemPrompt string
	headlessQuiet        bool
	headlessVerbose      bool
	headlessAgent        string
	headlessTitle        string
)

var headlessCmd = &cobra.Command{
	Use:   "headless [prompt...]",
	Short: "Run OpenCode in headless mode",
	Long: `Run OpenCode in headless mode without interactive TUI.

Headless mode executes a prompt and outputs results to stdout. All events are
streamed in the specified format (text, json, or jsonl).

Examples:
  # Simple prompt
  opencode headless "Fix the bug in main.go"

  # Auto-approve all tool executions
  opencode headless --yolo "Refactor the authentication module"

  # With timeout and JSON output
  opencode headless -o json -t 5m "Run tests and fix failures"

  # Read prompt from stdin
  echo "Fix linting errors" | opencode headless --stdin

  # Continue previous session
  opencode headless -c "Now add tests for what you just implemented"

  # With context files
  opencode headless -f spec.md -f api.yaml "Implement the API from spec"

  # Stream JSONL events for programmatic consumption
  opencode headless -o jsonl "Implement feature X" | jq -r '.type'`,
	RunE: runHeadless,
}

func init() {
	// Prompt input
	headlessCmd.Flags().StringVarP(&headlessPrompt, "prompt", "p", "", "Prompt/instruction to execute")
	headlessCmd.Flags().BoolVar(&headlessStdin, "stdin", false, "Read prompt from stdin")
	headlessCmd.Flags().StringArrayVarP(&headlessFiles, "file", "f", nil, "File(s) to attach as context")

	// Working directory and session
	headlessCmd.Flags().StringVarP(&headlessWorkDir, "workdir", "w", "", "Working directory")
	headlessCmd.Flags().StringVarP(&headlessSessionID, "session", "s", "", "Continue existing session ID")
	headlessCmd.Flags().BoolVarP(&headlessContinue, "continue", "c", false, "Continue the last session")
	headlessCmd.Flags().BoolVar(&headlessNoSave, "no-save", false, "Don't persist session (ephemeral)")
	headlessCmd.Flags().StringVar(&headlessTitle, "title", "", "Session title")

	// Tool permissions
	headlessCmd.Flags().BoolVar(&headlessAutoApprove, "auto-approve", false, "Auto-approve all tool executions")
	headlessCmd.Flags().BoolVar(&headlessAutoApprove, "yolo", false, "Alias for --auto-approve")

	// Output format
	headlessCmd.Flags().StringVarP(&headlessOutputFormat, "output-format", "o", "text", "Output format: text, json, jsonl")
	headlessCmd.Flags().BoolVarP(&headlessQuiet, "quiet", "q", false, "Suppress progress output, only show result")
	headlessCmd.Flags().BoolVarP(&headlessVerbose, "verbose", "v", false, "Show all events (with jsonl format)")

	// Execution limits
	headlessCmd.Flags().StringVarP(&headlessTimeout, "timeout", "t", "30m", "Maximum execution time (e.g., 5m, 1h)")
	headlessCmd.Flags().IntVar(&headlessMaxSteps, "max-steps", 50, "Maximum agentic loop iterations")

	// Agent and model
	headlessCmd.Flags().StringVar(&headlessAgent, "agent", "", "Agent to use")
	headlessCmd.Flags().StringVar(&headlessSystemPrompt, "system-prompt", "", "Custom system prompt file")
}

func runHeadless(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workDir, err := GetWorkDir(headlessWorkDir)
	if err != nil {
		return err
	}

	// Parse timeout
	timeout, err := time.ParseDuration(headlessTimeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	// Parse output format
	var outputFormat headless.OutputFormat
	switch strings.ToLower(headlessOutputFormat) {
	case "text":
		outputFormat = headless.OutputText
	case "json":
		outputFormat = headless.OutputJSON
	case "jsonl":
		outputFormat = headless.OutputJSONL
	default:
		return fmt.Errorf("invalid output format: %s (must be text, json, or jsonl)", headlessOutputFormat)
	}

	// Build prompt from args if not provided via flag
	prompt := headlessPrompt
	if prompt == "" && len(args) > 0 {
		prompt = strings.Join(args, " ")
	}

	// Validate that we have a prompt source
	if prompt == "" && !headlessStdin && !headlessContinue && headlessSessionID == "" {
		return fmt.Errorf("prompt required. Provide via argument, --prompt flag, or --stdin")
	}

	// Get global model override
	model := GetGlobalModel()

	// Create headless config
	cfg := &headless.Config{
		Prompt:       prompt,
		WorkDir:      workDir,
		AutoApprove:  headlessAutoApprove,
		OutputFormat: outputFormat,
		Timeout:      timeout,
		MaxSteps:     headlessMaxSteps,
		ReadStdin:    headlessStdin,
		NoSave:       headlessNoSave,
		SessionID:    headlessSessionID,
		ContinueLast: headlessContinue,
		Files:        headlessFiles,
		SystemPrompt: headlessSystemPrompt,
		Quiet:        headlessQuiet,
		Verbose:      headlessVerbose,
		Model:        model,
		Agent:        headlessAgent,
		Title:        headlessTitle,
	}

	// Create and run headless runner
	runner := headless.NewRunner(cfg)
	result, err := runner.Run(cmd.Context(), os.Stdout)

	// Exit with appropriate code
	if result != nil {
		os.Exit(int(result.ExitCode))
	}

	if err != nil {
		return err
	}

	return nil
}
