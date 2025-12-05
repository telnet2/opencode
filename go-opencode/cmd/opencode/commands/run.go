package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
	"github.com/spf13/cobra"
)

var (
	runModel        string
	runAgent        string
	runContinue     bool
	runSession      string
	runFormat       string
	runFiles        []string
	runTitle        string
	runPrompt       string
	runPromptFile   string
	runPromptInline string
	runDir          string
)

var runCmd = &cobra.Command{
	Use:   "run [message...]",
	Short: "Start an interactive OpenCode session",
	Long: `Start an interactive OpenCode session with the specified message.

Examples:
  opencode run "Fix the bug in main.go"
  opencode run --model anthropic/claude-sonnet-4 "Explain this code"
  opencode run --continue  # Continue last session
  opencode run --file main.go "Review this file"`,
	RunE: runInteractive,
}

func init() {
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use (provider/model format)")
	runCmd.Flags().StringVar(&runAgent, "agent", "", "Agent to use")
	runCmd.Flags().BoolVarP(&runContinue, "continue", "c", false, "Continue the last session")
	runCmd.Flags().StringVarP(&runSession, "session", "s", "", "Session ID to continue")
	runCmd.Flags().StringVar(&runFormat, "format", "default", "Output format (default|json)")
	runCmd.Flags().StringArrayVarP(&runFiles, "file", "f", nil, "File(s) to attach to message")
	runCmd.Flags().StringVar(&runTitle, "title", "", "Session title")
	runCmd.Flags().StringVar(&runPrompt, "prompt", "", "Custom prompt template")
	runCmd.Flags().StringVar(&runPromptFile, "prompt-file", "", "Custom prompt from file")
	runCmd.Flags().StringVar(&runPromptInline, "prompt-inline", "", "Custom prompt as inline text")
	runCmd.Flags().StringVar(&runDir, "directory", "", "Working directory")
}

func runInteractive(cmd *cobra.Command, args []string) error {
	// Determine working directory
	workDir, err := GetWorkDir(runDir)
	if err != nil {
		return err
	}

	// Initialize paths
	paths := config.GetPaths()
	if err := paths.EnsurePaths(); err != nil {
		return err
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		return err
	}

	// Override model if specified
	if runModel != "" {
		appConfig.Model = runModel
	}

	// Build message from args
	message := strings.Join(args, " ")
	if message == "" && !runContinue && runSession == "" {
		return fmt.Errorf("message required. Usage: opencode run \"your message\"")
	}

	// Initialize storage
	store := storage.New(paths.StoragePath())

	// Initialize providers
	ctx := context.Background()
	providerReg, err := provider.InitializeProviders(ctx, appConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Initialize tool registry
	toolReg := tool.DefaultRegistry(workDir)

	// Initialize permission checker
	permChecker := permission.NewChecker()

	// Handle custom prompt
	var systemPrompt string
	if runPromptFile != "" {
		data, err := os.ReadFile(runPromptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		systemPrompt = string(data)
	} else if runPromptInline != "" {
		systemPrompt = runPromptInline
	} else if runPrompt != "" {
		// Try to read as file first, then use as inline
		if data, err := os.ReadFile(runPrompt); err == nil {
			systemPrompt = string(data)
		} else {
			systemPrompt = runPrompt
		}
	}

	// Handle file attachments - read and include in message
	var fileContent strings.Builder
	for _, file := range runFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}
		fileContent.WriteString(fmt.Sprintf("\n\n--- File: %s ---\n%s", file, string(content)))
	}
	if fileContent.Len() > 0 {
		message = message + fileContent.String()
	}

	// Handle continue/session
	var sessionID string
	if runSession != "" {
		sessionID = runSession
	} else if runContinue {
		// List sessions and get the most recent
		sessions, err := store.List(ctx, []string{"session"})
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		if len(sessions) > 0 {
			sessionID = sessions[len(sessions)-1]
		}
	}

	// Create session ID if not continuing
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess_%d", os.Getpid())
	}

	// Parse default provider and model from config
	var defaultProviderID, defaultModelID string
	if appConfig.Model != "" {
		parts := strings.SplitN(appConfig.Model, "/", 2)
		if len(parts) == 2 {
			defaultProviderID = parts[0]
			defaultModelID = parts[1]
		}
	}

	// Create processor
	processor := session.NewProcessor(providerReg, toolReg, store, permChecker, defaultProviderID, defaultModelID)

	// Create agent configuration
	agentName := runAgent
	if agentName == "" {
		agentName = "default"
	}
	agent := session.DefaultAgent()
	agent.Name = agentName
	agent.Prompt = systemPrompt

	// Process callback
	callback := func(msg *types.Message, parts []types.Part) {
		for _, part := range parts {
			switch p := part.(type) {
			case *types.TextPart:
				fmt.Print(p.Text)
			}
		}
	}

	// Note: User message will be added by the processor
	// The message content is passed through the agent's input

	// Run the agentic loop
	fmt.Printf("Starting session %s...\n", sessionID)
	fmt.Printf("Model: %s\n", appConfig.Model)
	fmt.Printf("Message: %s\n\n", truncate(message, 100))

	if err := processor.Process(ctx, sessionID, agent, callback); err != nil {
		return fmt.Errorf("processing error: %w", err)
	}

	fmt.Println()
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
