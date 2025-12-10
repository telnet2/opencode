package headless

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/opencode-ai/opencode/internal/agent"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/executor"
	"github.com/opencode-ai/opencode/internal/mcp"
	"github.com/opencode-ai/opencode/internal/permission"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/opencode-ai/opencode/internal/session"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/internal/tool"
	"github.com/opencode-ai/opencode/pkg/types"
)

// Runner executes prompts in headless mode.
type Runner struct {
	config    *Config
	appConfig *types.Config
	printer   *Printer
	storage   *storage.Storage

	providerReg *provider.Registry
	toolReg     *tool.Registry
	agentReg    *agent.Registry
	permChecker PermissionCheckerInterface
	mcpClient   *mcp.Client
	processor   *session.Processor

	defaultProviderID string
	defaultModelID    string
}

// NewRunner creates a new headless runner.
func NewRunner(cfg *Config) *Runner {
	return &Runner{
		config: cfg,
	}
}

// Run executes the headless session and returns the result.
func (r *Runner) Run(ctx context.Context, writer io.Writer) (*Result, error) {
	// Create printer for output
	r.printer = NewPrinter(writer, r.config.OutputFormat, r.config.Quiet, r.config.Verbose)
	r.printer.Subscribe()
	defer r.printer.Unsubscribe()

	// Initialize all components
	if err := r.initialize(ctx); err != nil {
		r.printer.SetResult("error", ExitError, "", err)
		return r.printer.GetResult(), err
	}

	// Clean up MCP client on exit
	if r.mcpClient != nil {
		defer r.mcpClient.Close()
	}

	// Get or build the prompt
	prompt, err := r.getPrompt()
	if err != nil {
		r.printer.SetResult("error", ExitInvalidInput, "", err)
		return r.printer.GetResult(), err
	}

	if prompt == "" {
		err := errors.New("prompt is required")
		r.printer.SetResult("error", ExitInvalidInput, "", err)
		return r.printer.GetResult(), err
	}

	// Create or continue session
	sessionID, err := r.getOrCreateSession(ctx)
	if err != nil {
		r.printer.SetResult("error", ExitSessionNotFound, "", err)
		return r.printer.GetResult(), err
	}
	r.printer.SetSessionID(sessionID)

	// Set model info
	r.printer.SetModel(fmt.Sprintf("%s/%s", r.defaultProviderID, r.defaultModelID))

	// Add user message to session
	if err := r.addUserMessage(ctx, sessionID, prompt); err != nil {
		r.printer.SetResult("error", ExitError, "", err)
		return r.printer.GetResult(), err
	}

	// Create context with timeout
	runCtx := ctx
	var cancel context.CancelFunc
	if r.config.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	// Create agent configuration
	agentCfg := r.createAgent()

	// Process callback for streaming output
	var finalMessage string
	callback := func(msg *types.Message, parts []types.Part) {
		if msg.Tokens != nil {
			r.printer.SetTokens(msg.Tokens)
		}
		for _, part := range parts {
			if textPart, ok := part.(*types.TextPart); ok {
				finalMessage = textPart.Text
			}
		}
	}

	// Run the agentic loop
	err = r.processor.Process(runCtx, sessionID, agentCfg, callback)

	// Handle result based on error type
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			r.printer.SetResult("timeout", ExitTimeout, finalMessage, err)
			return r.printer.GetResult(), err
		}
		if permission.IsRejectedError(err) {
			r.printer.SetResult("permission_denied", ExitPermissionDenied, finalMessage, err)
			return r.printer.GetResult(), err
		}
		r.printer.SetResult("error", ExitError, finalMessage, err)
		return r.printer.GetResult(), err
	}

	r.printer.SetResult("success", ExitSuccess, finalMessage, nil)

	// Print final result if JSON format
	r.printer.PrintFinalResult()

	return r.printer.GetResult(), nil
}

// initialize sets up all required components.
func (r *Runner) initialize(ctx context.Context) error {
	// Ensure paths exist
	paths := config.GetPaths()
	if err := paths.EnsurePaths(); err != nil {
		return fmt.Errorf("failed to ensure paths: %w", err)
	}

	// Load configuration
	appConfig, err := config.Load(r.config.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	r.appConfig = appConfig

	// Override model if specified
	if r.config.Model != "" {
		r.appConfig.Model = r.config.Model
	}

	// Parse default provider and model
	r.parseModel()

	// Initialize storage
	if r.config.NoSave {
		// Use ephemeral storage (memory-based or temp directory)
		tempDir, err := os.MkdirTemp("", "opencode-headless-*")
		if err != nil {
			return fmt.Errorf("failed to create temp storage: %w", err)
		}
		r.storage = storage.New(tempDir)
	} else {
		r.storage = storage.New(paths.StoragePath())
	}

	// Initialize providers
	providerReg, err := provider.InitializeProviders(ctx, r.appConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}
	r.providerReg = providerReg

	// Initialize tool registry
	r.toolReg = tool.DefaultRegistry(r.config.WorkDir, r.storage)

	// Initialize agent registry
	r.agentReg = agent.NewRegistry()
	r.toolReg.RegisterTaskTool(r.agentReg)

	// Initialize MCP if configured
	if r.appConfig.MCP != nil && len(r.appConfig.MCP) > 0 {
		r.mcpClient = mcp.NewClient()
		for name, cfg := range r.appConfig.MCP {
			enabled := cfg.Enabled == nil || *cfg.Enabled
			mcpCfg := &mcp.Config{
				Enabled:     enabled,
				Type:        mcp.TransportType(cfg.Type),
				URL:         cfg.URL,
				Headers:     cfg.Headers,
				Command:     cfg.Command,
				Environment: cfg.Environment,
				Timeout:     cfg.Timeout,
			}
			if err := r.mcpClient.AddServer(ctx, name, mcpCfg); err != nil {
				// Log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: MCP server %s failed: %v\n", name, err)
				continue
			}
		}
		mcp.RegisterMCPTools(r.mcpClient, r.toolReg)
	}

	// Initialize permission checker
	if r.config.AutoApprove {
		r.permChecker = NewAutoApproveChecker(r.config.Verbose)
	} else {
		r.permChecker = &StandardCheckerWrapper{permission.NewChecker()}
	}

	// Create subagent executor for task tool
	subagentExecutor := executor.NewSubagentExecutor(executor.SubagentExecutorConfig{
		Storage:           r.storage,
		ProviderRegistry:  r.providerReg,
		ToolRegistry:      r.toolReg,
		PermissionChecker: permission.NewChecker(), // Subagents use standard checker
		AgentRegistry:     r.agentReg,
		WorkDir:           r.config.WorkDir,
		DefaultProviderID: r.defaultProviderID,
		DefaultModelID:    r.defaultModelID,
	})
	r.toolReg.SetTaskExecutor(subagentExecutor)

	// Create processor
	r.processor = session.NewProcessor(
		r.providerReg,
		r.toolReg,
		r.storage,
		permission.NewChecker(), // The processor needs the real checker for interface compatibility
		r.defaultProviderID,
		r.defaultModelID,
	)

	return nil
}

// parseModel parses the model string into provider and model IDs.
func (r *Runner) parseModel() {
	model := r.appConfig.Model
	if model == "" {
		r.defaultProviderID = "anthropic"
		r.defaultModelID = "claude-sonnet-4-20250514"
		return
	}

	parts := strings.SplitN(model, "/", 2)
	if len(parts) == 2 {
		r.defaultProviderID = parts[0]
		r.defaultModelID = parts[1]
	} else {
		r.defaultProviderID = "anthropic"
		r.defaultModelID = model
	}
}

// getPrompt retrieves the prompt from various sources.
func (r *Runner) getPrompt() (string, error) {
	var prompt string

	// Read from stdin if specified
	if r.config.ReadStdin {
		reader := bufio.Reader{}
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		prompt = strings.Join(lines, "\n")
		_ = reader // Unused, just for clarity
	}

	// Override with direct prompt if provided
	if r.config.Prompt != "" {
		if prompt != "" {
			// Combine stdin and prompt
			prompt = r.config.Prompt + "\n\n" + prompt
		} else {
			prompt = r.config.Prompt
		}
	}

	// Attach file contents if specified
	if len(r.config.Files) > 0 {
		var fileContent strings.Builder
		for _, file := range r.config.Files {
			content, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("failed to read file %s: %w", file, err)
			}
			fileContent.WriteString(fmt.Sprintf("\n\n--- File: %s ---\n%s", file, string(content)))
		}
		prompt = prompt + fileContent.String()
	}

	return strings.TrimSpace(prompt), nil
}

// getOrCreateSession gets an existing session or creates a new one.
func (r *Runner) getOrCreateSession(ctx context.Context) (string, error) {
	// Continue existing session
	if r.config.SessionID != "" {
		// Verify session exists
		var sess types.Session
		if err := r.storage.Get(ctx, []string{"session", r.config.SessionID}, &sess); err != nil {
			return "", fmt.Errorf("session not found: %s", r.config.SessionID)
		}
		return r.config.SessionID, nil
	}

	// Continue last session
	if r.config.ContinueLast {
		sessions, err := r.storage.List(ctx, []string{"session"})
		if err != nil {
			return "", fmt.Errorf("failed to list sessions: %w", err)
		}
		if len(sessions) > 0 {
			return sessions[len(sessions)-1], nil
		}
		// No existing sessions, create new
	}

	// Create new session
	return r.createSession(ctx)
}

// createSession creates a new session.
func (r *Runner) createSession(ctx context.Context) (string, error) {
	sessionID := fmt.Sprintf("sess_%s", ulid.Make().String())

	title := r.config.Title
	if title == "" {
		title = "Headless Session"
	}

	sess := &types.Session{
		ID:        sessionID,
		Directory: r.config.WorkDir,
		Title:     title,
		Time: types.SessionTime{
			Created: time.Now().UnixMilli(),
		},
		Summary: types.SessionSummary{},
	}

	// Save session
	if err := r.storage.Put(ctx, []string{"session", sessionID}, sess); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Publish session created event
	event.PublishSync(event.Event{
		Type: event.SessionCreated,
		Data: event.SessionCreatedData{Info: sess},
	})

	return sessionID, nil
}

// addUserMessage adds the user message to the session.
func (r *Runner) addUserMessage(ctx context.Context, sessionID string, content string) error {
	msgID := ulid.Make().String()
	now := time.Now().UnixMilli()

	msg := &types.Message{
		ID:        msgID,
		SessionID: sessionID,
		Role:      "user",
		Time: types.MessageTime{
			Created: now,
		},
	}

	// Save message
	if err := r.storage.Put(ctx, []string{"message", sessionID, msgID}, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Create and save text part
	partID := ulid.Make().String()
	textPart := &types.TextPart{
		ID:        partID,
		Type:      "text",
		MessageID: msgID,
		Text:      content,
	}

	if err := r.storage.Put(ctx, []string{"part", msgID, partID}, textPart); err != nil {
		return fmt.Errorf("failed to save part: %w", err)
	}

	// Publish message created event
	event.PublishSync(event.Event{
		Type: event.MessageCreated,
		Data: event.MessageCreatedData{Info: msg},
	})

	return nil
}

// createAgent creates the agent configuration for the session.
func (r *Runner) createAgent() *session.Agent {
	agentCfg := session.DefaultAgent()

	if r.config.Agent != "" {
		agentCfg.Name = r.config.Agent
	}

	// Load system prompt if specified
	if r.config.SystemPrompt != "" {
		data, err := os.ReadFile(r.config.SystemPrompt)
		if err == nil {
			agentCfg.Prompt = string(data)
		}
	}

	// Set max steps
	if r.config.MaxSteps > 0 {
		agentCfg.MaxSteps = r.config.MaxSteps
	}

	return agentCfg
}

// StandardCheckerWrapper wraps the standard permission checker to implement PermissionCheckerInterface.
type StandardCheckerWrapper struct {
	*permission.Checker
}

// Check delegates to the underlying checker.
func (w *StandardCheckerWrapper) Check(ctx context.Context, req permission.Request, action permission.PermissionAction) error {
	return w.Checker.Check(ctx, req, action)
}

// Ask delegates to the underlying checker.
func (w *StandardCheckerWrapper) Ask(ctx context.Context, req permission.Request) error {
	return w.Checker.Ask(ctx, req)
}
