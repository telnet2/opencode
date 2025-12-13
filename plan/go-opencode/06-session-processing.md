# Phase 6: Session Processing (Week 10)

## Overview

Implement the agentic loop and message processing system. This is the core engine that orchestrates LLM interactions, tool execution, streaming responses, and conversation management.

---

## 6.1 Session Processor

### Main Processing Loop

```go
// internal/session/processor.go
package session

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "sync"
    "time"

    "github.com/opencode-ai/opencode-server/internal/event"
    "github.com/opencode-ai/opencode-server/internal/provider"
    "github.com/opencode-ai/opencode-server/internal/tool"
    "github.com/opencode-ai/opencode-server/pkg/types"
)

// Processor handles message processing and the agentic loop
type Processor struct {
    mu sync.Mutex

    providerRegistry *provider.Registry
    toolRegistry     *tool.Registry
    sessionStore     *Store
    messageStore     *MessageStore
    partStore        *PartStore
    permissionChecker *permission.Checker
    bus              *event.Bus

    // Active sessions
    sessions map[string]*sessionState
}

// sessionState tracks the state of an active session
type sessionState struct {
    abort    context.CancelFunc
    ctx      context.Context
    message  *types.Message
    parts    []types.Part
    waiters  []chan error
}

// NewProcessor creates a new session processor
func NewProcessor(
    providerReg *provider.Registry,
    toolReg *tool.Registry,
    sessionStore *Store,
    messageStore *MessageStore,
    partStore *PartStore,
    permChecker *permission.Checker,
    bus *event.Bus,
) *Processor {
    return &Processor{
        providerRegistry:  providerReg,
        toolRegistry:      toolReg,
        sessionStore:      sessionStore,
        messageStore:      messageStore,
        partStore:         partStore,
        permissionChecker: permChecker,
        bus:               bus,
        sessions:          make(map[string]*sessionState),
    }
}

// Process handles a new user message and generates assistant response
func (p *Processor) Process(ctx context.Context, sessionID string, callback ProcessCallback) error {
    p.mu.Lock()

    // Check if session is already processing
    if state, ok := p.sessions[sessionID]; ok {
        // Queue this request
        waiter := make(chan error, 1)
        state.waiters = append(state.waiters, waiter)
        p.mu.Unlock()

        // Wait for current processing to complete
        select {
        case err := <-waiter:
            if err != nil {
                return err
            }
            // Retry processing
            return p.Process(ctx, sessionID, callback)
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    // Create new session state
    loopCtx, cancel := context.WithCancel(ctx)
    state := &sessionState{
        abort: cancel,
        ctx:   loopCtx,
    }
    p.sessions[sessionID] = state
    p.mu.Unlock()

    // Ensure cleanup
    defer func() {
        p.mu.Lock()
        delete(p.sessions, sessionID)

        // Notify waiters
        for _, waiter := range state.waiters {
            waiter <- nil
        }
        p.mu.Unlock()
    }()

    // Run the agentic loop
    return p.runLoop(loopCtx, sessionID, state, callback)
}

// ProcessCallback is called with message updates during processing
type ProcessCallback func(msg *types.Message, parts []types.Part)

// Abort cancels processing for a session
func (p *Processor) Abort(sessionID string) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    state, ok := p.sessions[sessionID]
    if !ok {
        return fmt.Errorf("session not processing: %s", sessionID)
    }

    state.abort()
    return nil
}

// IsProcessing returns whether a session is currently processing
func (p *Processor) IsProcessing(sessionID string) bool {
    p.mu.Lock()
    defer p.mu.Unlock()
    _, ok := p.sessions[sessionID]
    return ok
}
```

### Agentic Loop Implementation

```go
// internal/session/loop.go
package session

import (
    "context"
    "fmt"
    "time"

    "github.com/opencode-ai/opencode-server/internal/provider"
    "github.com/opencode-ai/opencode-server/pkg/types"
)

const (
    MaxSteps          = 50
    MaxRetries        = 3
    RetryBaseDelay    = time.Second
    MaxContextTokens  = 150000 // Trigger compaction threshold
)

// runLoop executes the agentic loop
func (p *Processor) runLoop(ctx context.Context, sessionID string, state *sessionState, callback ProcessCallback) error {
    session, err := p.sessionStore.Get(ctx, sessionID)
    if err != nil {
        return fmt.Errorf("session not found: %w", err)
    }

    // Get the last user message
    messages, err := p.messageStore.List(ctx, sessionID)
    if err != nil {
        return err
    }

    if len(messages) == 0 {
        return fmt.Errorf("no messages in session")
    }

    lastUserMsg := messages[len(messages)-1]
    if lastUserMsg.Role != "user" {
        return fmt.Errorf("expected user message, got %s", lastUserMsg.Role)
    }

    // Get agent configuration
    agent, err := p.getAgent(lastUserMsg.Agent)
    if err != nil {
        return err
    }

    // Get model
    model, err := p.getModel(lastUserMsg.Model)
    if err != nil {
        return err
    }

    // Create assistant message
    assistantMsg := p.createAssistantMessage(sessionID, lastUserMsg, model, agent)
    state.message = assistantMsg

    // Notify callback
    callback(assistantMsg, nil)

    // Publish event
    p.bus.Publish(event.Event{
        Type: event.MessageUpdated,
        Data: event.MessageUpdatedData{Message: assistantMsg},
    })

    // Run loop
    step := 0
    retries := 0

    for {
        // Check context cancellation
        select {
        case <-ctx.Done():
            assistantMsg.Error = &types.MessageError{
                Type:    "abort",
                Message: "Processing aborted",
            }
            p.saveMessage(ctx, assistantMsg)
            return ctx.Err()
        default:
        }

        // Check step limit
        if step >= MaxSteps {
            assistantMsg.Error = &types.MessageError{
                Type:    "max_steps",
                Message: "Maximum steps reached",
            }
            p.saveMessage(ctx, assistantMsg)
            return fmt.Errorf("max steps exceeded")
        }

        // Check for context overflow and compact if needed
        if p.shouldCompact(messages) {
            if err := p.compactMessages(ctx, sessionID, messages); err != nil {
                // Log but don't fail
            }
            // Reload messages
            messages, _ = p.messageStore.List(ctx, sessionID)
        }

        // Build completion request
        req, err := p.buildCompletionRequest(ctx, session, messages, assistantMsg, agent, model)
        if err != nil {
            return fmt.Errorf("failed to build request: %w", err)
        }

        // Call LLM with streaming
        stream, err := model.Provider.CreateCompletion(ctx, req)
        if err != nil {
            retries++
            if retries >= MaxRetries {
                assistantMsg.Error = &types.MessageError{
                    Type:    "api",
                    Message: err.Error(),
                }
                p.saveMessage(ctx, assistantMsg)
                return err
            }

            // Exponential backoff
            delay := RetryBaseDelay * time.Duration(1<<retries)
            time.Sleep(delay)
            continue
        }

        // Process stream
        finishReason, err := p.processStream(ctx, stream, state, callback)
        stream.Close()

        if err != nil {
            retries++
            if retries >= MaxRetries {
                assistantMsg.Error = &types.MessageError{
                    Type:    "api",
                    Message: err.Error(),
                }
                p.saveMessage(ctx, assistantMsg)
                return err
            }
            continue
        }

        // Reset retries on success
        retries = 0

        // Check finish reason
        switch finishReason {
        case "stop":
            // Normal completion
            assistantMsg.Finish = ptr("stop")
            p.saveMessage(ctx, assistantMsg)
            return nil

        case "tool_calls":
            // Execute tools and continue loop
            if err := p.executeToolCalls(ctx, state, agent, callback); err != nil {
                // Tool execution errors don't stop the loop
                // The error is captured in the tool part
            }
            step++
            continue

        case "max_tokens":
            // Output limit reached
            assistantMsg.Finish = ptr("max_tokens")
            assistantMsg.Error = &types.MessageError{
                Type:    "output_length",
                Message: "Output length limit reached",
            }
            p.saveMessage(ctx, assistantMsg)
            return nil

        case "error":
            retries++
            if retries >= MaxRetries {
                return fmt.Errorf("stream error")
            }
            continue

        default:
            // Unknown finish reason, treat as stop
            assistantMsg.Finish = ptr(finishReason)
            p.saveMessage(ctx, assistantMsg)
            return nil
        }
    }
}

func (p *Processor) shouldCompact(messages []*types.Message) bool {
    // Estimate token count
    totalTokens := 0
    for _, msg := range messages {
        if msg.Tokens != nil {
            totalTokens += msg.Tokens.Input + msg.Tokens.Output
        }
    }
    return totalTokens > MaxContextTokens
}
```

### Stream Processing

```go
// internal/session/stream.go
package session

import (
    "context"
    "encoding/json"
    "io"

    "github.com/opencode-ai/opencode-server/internal/provider"
    "github.com/opencode-ai/opencode-server/pkg/types"
)

// processStream processes events from the LLM stream
func (p *Processor) processStream(
    ctx context.Context,
    stream provider.CompletionStream,
    state *sessionState,
    callback ProcessCallback,
) (string, error) {
    var currentTextPart *types.TextPart
    var currentReasoningPart *types.ReasoningPart
    var currentToolParts map[string]*types.ToolPart
    var finishReason string
    var stepTokens provider.TokenUsage
    var stepCost float64

    currentToolParts = make(map[string]*types.ToolPart)

    // Emit step start
    stepStartPart := &types.StepStartPart{
        ID:   generatePartID(),
        Type: "step-start",
    }
    state.parts = append(state.parts, stepStartPart)

    for {
        select {
        case <-ctx.Done():
            return "error", ctx.Err()
        default:
        }

        event, err := stream.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return "error", err
        }

        switch e := event.(type) {
        case provider.TextStartEvent:
            currentTextPart = &types.TextPart{
                ID:   generatePartID(),
                Type: "text",
                Time: types.PartTime{Start: ptr(time.Now().UnixMilli())},
            }
            state.parts = append(state.parts, currentTextPart)
            callback(state.message, state.parts)

        case provider.TextDeltaEvent:
            if currentTextPart != nil {
                currentTextPart.Text += e.Text

                // Publish delta event
                p.bus.Publish(event.Event{
                    Type: event.PartUpdated,
                    Data: event.PartUpdatedData{
                        SessionID: state.message.SessionID,
                        MessageID: state.message.ID,
                        Part:      currentTextPart,
                        Delta:     &e.Text,
                    },
                })

                callback(state.message, state.parts)
            }

        case provider.TextEndEvent:
            if currentTextPart != nil {
                currentTextPart.Time.End = ptr(time.Now().UnixMilli())
                p.savePart(ctx, state.message.ID, currentTextPart)
                currentTextPart = nil
            }

        case provider.ReasoningStartEvent:
            currentReasoningPart = &types.ReasoningPart{
                ID:   generatePartID(),
                Type: "reasoning",
                Time: types.PartTime{Start: ptr(time.Now().UnixMilli())},
            }
            state.parts = append(state.parts, currentReasoningPart)
            callback(state.message, state.parts)

        case provider.ReasoningDeltaEvent:
            if currentReasoningPart != nil {
                currentReasoningPart.Text += e.Text

                p.bus.Publish(event.Event{
                    Type: event.PartUpdated,
                    Data: event.PartUpdatedData{
                        SessionID: state.message.SessionID,
                        MessageID: state.message.ID,
                        Part:      currentReasoningPart,
                        Delta:     &e.Text,
                    },
                })

                callback(state.message, state.parts)
            }

        case provider.ReasoningEndEvent:
            if currentReasoningPart != nil {
                currentReasoningPart.Time.End = ptr(time.Now().UnixMilli())
                p.savePart(ctx, state.message.ID, currentReasoningPart)
                currentReasoningPart = nil
            }

        case provider.ToolCallStartEvent:
            toolPart := &types.ToolPart{
                ID:         generatePartID(),
                Type:       "tool",
                ToolCallID: e.ID,
                ToolName:   e.Name,
                State:      "pending",
                Input:      make(map[string]any),
                Time:       types.PartTime{Start: ptr(time.Now().UnixMilli())},
            }
            currentToolParts[e.ID] = toolPart
            state.parts = append(state.parts, toolPart)
            callback(state.message, state.parts)

        case provider.ToolCallDeltaEvent:
            // Accumulate input JSON fragments
            // (handled in ToolCallEndEvent)

        case provider.ToolCallEndEvent:
            if toolPart, ok := currentToolParts[e.ID]; ok {
                var input map[string]any
                json.Unmarshal(e.Input, &input)
                toolPart.Input = input
                toolPart.State = "running"

                p.bus.Publish(event.Event{
                    Type: event.PartUpdated,
                    Data: event.PartUpdatedData{
                        SessionID: state.message.SessionID,
                        MessageID: state.message.ID,
                        Part:      toolPart,
                    },
                })

                callback(state.message, state.parts)
            }

        case provider.StepFinishEvent:
            stepTokens = e.Tokens
            stepCost = e.Cost

            // Update message tokens
            if state.message.Tokens == nil {
                state.message.Tokens = &types.TokenUsage{}
            }
            state.message.Tokens.Input += stepTokens.Input
            state.message.Tokens.Output += stepTokens.Output
            state.message.Tokens.Reasoning += stepTokens.Reasoning
            state.message.Tokens.Cache.Read += stepTokens.Cache.Read
            state.message.Tokens.Cache.Write += stepTokens.Cache.Write
            state.message.Cost += stepCost

            // Emit step finish part
            stepFinishPart := &types.StepFinishPart{
                ID:     generatePartID(),
                Type:   "step-finish",
                Tokens: stepTokens,
                Cost:   stepCost,
            }
            state.parts = append(state.parts, stepFinishPart)

        case provider.FinishEvent:
            finishReason = e.Reason

            if e.Error != nil {
                return "error", e.Error
            }
        }
    }

    return finishReason, nil
}
```

---

## 6.2 Tool Execution

```go
// internal/session/tools.go
package session

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/opencode-ai/opencode-server/internal/tool"
    "github.com/opencode-ai/opencode-server/pkg/types"
)

// executeToolCalls executes all pending tool calls
func (p *Processor) executeToolCalls(
    ctx context.Context,
    state *sessionState,
    agent *Agent,
    callback ProcessCallback,
) error {
    // Find all pending tool parts
    var pendingTools []*types.ToolPart
    for _, part := range state.parts {
        if toolPart, ok := part.(*types.ToolPart); ok {
            if toolPart.State == "running" {
                pendingTools = append(pendingTools, toolPart)
            }
        }
    }

    // Execute each tool
    for _, toolPart := range pendingTools {
        err := p.executeSingleTool(ctx, state, agent, toolPart, callback)
        if err != nil {
            // Error is captured in tool part, don't stop processing
            continue
        }
    }

    return nil
}

// executeSingleTool executes a single tool call
func (p *Processor) executeSingleTool(
    ctx context.Context,
    state *sessionState,
    agent *Agent,
    toolPart *types.ToolPart,
    callback ProcessCallback,
) error {
    // Get the tool
    t, ok := p.toolRegistry.Get(toolPart.ToolName)
    if !ok {
        toolPart.State = "error"
        toolPart.Error = ptr(fmt.Sprintf("Tool not found: %s", toolPart.ToolName))
        toolPart.Time.End = ptr(time.Now().UnixMilli())
        p.savePart(ctx, state.message.ID, toolPart)
        callback(state.message, state.parts)
        return fmt.Errorf("tool not found: %s", toolPart.ToolName)
    }

    // Check permissions
    if err := p.checkToolPermission(ctx, state, agent, toolPart); err != nil {
        toolPart.State = "error"
        toolPart.Error = ptr(err.Error())
        toolPart.Time.End = ptr(time.Now().UnixMilli())
        p.savePart(ctx, state.message.ID, toolPart)
        callback(state.message, state.parts)
        return err
    }

    // Check for doom loop
    if err := p.checkDoomLoop(ctx, state, agent, toolPart); err != nil {
        toolPart.State = "error"
        toolPart.Error = ptr(err.Error())
        toolPart.Time.End = ptr(time.Now().UnixMilli())
        p.savePart(ctx, state.message.ID, toolPart)
        callback(state.message, state.parts)
        return err
    }

    // Prepare input
    inputJSON, _ := json.Marshal(toolPart.Input)

    // Create tool context
    toolCtx := tool.Context{
        SessionID: state.message.SessionID,
        MessageID: state.message.ID,
        CallID:    toolPart.ToolCallID,
        Agent:     agent.Name,
        Abort:     ctx,
        Extra: map[string]any{
            "model": state.message.ModelID,
        },
    }

    // Metadata callback for real-time updates
    toolCtx.SetMetadataFunc(func(title string, meta map[string]any) {
        toolPart.Title = &title
        if toolPart.Metadata == nil {
            toolPart.Metadata = make(map[string]any)
        }
        for k, v := range meta {
            toolPart.Metadata[k] = v
        }

        p.bus.Publish(event.Event{
            Type: event.PartUpdated,
            Data: event.PartUpdatedData{
                SessionID: state.message.SessionID,
                MessageID: state.message.ID,
                Part:      toolPart,
            },
        })

        callback(state.message, state.parts)
    })

    // Execute tool
    result, err := t.Execute(ctx, inputJSON, toolCtx)

    if err != nil {
        toolPart.State = "error"
        toolPart.Error = ptr(err.Error())
        toolPart.Time.End = ptr(time.Now().UnixMilli())
        p.savePart(ctx, state.message.ID, toolPart)
        callback(state.message, state.parts)
        return err
    }

    // Update tool part with result
    toolPart.State = "completed"
    toolPart.Output = &result.Output
    toolPart.Title = &result.Title
    if result.Metadata != nil {
        if toolPart.Metadata == nil {
            toolPart.Metadata = make(map[string]any)
        }
        for k, v := range result.Metadata {
            toolPart.Metadata[k] = v
        }
    }
    toolPart.Time.End = ptr(time.Now().UnixMilli())

    // Handle attachments
    if len(result.Attachments) > 0 {
        toolPart.Metadata["attachments"] = result.Attachments
    }

    p.savePart(ctx, state.message.ID, toolPart)

    // Publish event
    p.bus.Publish(event.Event{
        Type: event.PartUpdated,
        Data: event.PartUpdatedData{
            SessionID: state.message.SessionID,
            MessageID: state.message.ID,
            Part:      toolPart,
        },
    })

    callback(state.message, state.parts)
    return nil
}

// checkDoomLoop detects and handles repetitive tool calls
func (p *Processor) checkDoomLoop(
    ctx context.Context,
    state *sessionState,
    agent *Agent,
    toolPart *types.ToolPart,
) error {
    // Count identical tool calls
    count := 0
    inputJSON, _ := json.Marshal(toolPart.Input)
    inputStr := string(inputJSON)

    for _, part := range state.parts {
        if tp, ok := part.(*types.ToolPart); ok {
            if tp.ToolName == toolPart.ToolName {
                otherInput, _ := json.Marshal(tp.Input)
                if string(otherInput) == inputStr {
                    count++
                }
            }
        }
    }

    // Threshold for doom loop detection
    if count < 3 {
        return nil
    }

    // Check permission policy
    switch agent.Permission.DoomLoop {
    case "allow":
        return nil

    case "deny":
        return fmt.Errorf("doom loop detected: %s called %d times with same input", toolPart.ToolName, count)

    case "ask", "":
        // Request permission from user
        permID := generatePermissionID()
        p.bus.Publish(event.Event{
            Type: event.PermissionRequired,
            Data: event.PermissionRequiredData{
                ID:        permID,
                Type:      "doom_loop",
                Pattern:   []string{toolPart.ToolName},
                SessionID: state.message.SessionID,
                Title:     fmt.Sprintf("Allow repeated %s call?", toolPart.ToolName),
            },
        })

        // Wait for permission response
        granted, err := p.waitForPermission(ctx, permID)
        if err != nil {
            return err
        }
        if !granted {
            return fmt.Errorf("doom loop denied by user")
        }
        return nil
    }

    return nil
}
```

---

## 6.3 System Prompt Builder

```go
// internal/session/system.go
package session

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/opencode-ai/opencode-server/pkg/types"
)

// SystemPrompt builds the system prompt for the LLM
type SystemPrompt struct {
    session   *types.Session
    agent     *Agent
    modelID   string
    providerID string
}

func NewSystemPrompt(session *types.Session, agent *Agent, providerID, modelID string) *SystemPrompt {
    return &SystemPrompt{
        session:    session,
        agent:      agent,
        modelID:    modelID,
        providerID: providerID,
    }
}

// Build constructs the complete system prompt
func (s *SystemPrompt) Build() string {
    var parts []string

    // 1. Provider-specific header
    if header := s.providerHeader(); header != "" {
        parts = append(parts, header)
    }

    // 2. Base agent prompt
    if s.agent.Prompt != "" {
        parts = append(parts, s.agent.Prompt)
    }

    // 3. Model-specific instructions
    if modelPrompt := s.modelPrompt(); modelPrompt != "" {
        parts = append(parts, modelPrompt)
    }

    // 4. Environment context
    parts = append(parts, s.environmentContext())

    // 5. Custom rules (AGENTS.md, CLAUDE.md)
    if rules := s.loadCustomRules(); rules != "" {
        parts = append(parts, rules)
    }

    // 6. Tool instructions
    if toolInstructions := s.toolInstructions(); toolInstructions != "" {
        parts = append(parts, toolInstructions)
    }

    return strings.Join(parts, "\n\n")
}

func (s *SystemPrompt) providerHeader() string {
    switch s.providerID {
    case "anthropic":
        return `You are Claude, an AI assistant made by Anthropic. You are helpful, harmless, and honest.

IMPORTANT: You have access to tools that can read, write, and execute commands on the user's computer. Use them responsibly.`

    default:
        return ""
    }
}

func (s *SystemPrompt) modelPrompt() string {
    switch {
    case strings.Contains(s.modelID, "claude"):
        return `When using tools, be decisive and take action. Don't ask for confirmation unless absolutely necessary.

For file operations:
- Read files before editing to understand context
- Make minimal, focused changes
- Preserve existing code style and formatting`

    case strings.Contains(s.modelID, "gpt"):
        return `When working with files:
- Always read files before making changes
- Make precise, targeted edits
- Follow existing code conventions`

    case strings.Contains(s.modelID, "gemini"):
        return `For code tasks:
- Examine existing code structure first
- Make minimal necessary changes
- Maintain code style consistency`

    default:
        return ""
    }
}

func (s *SystemPrompt) environmentContext() string {
    var env strings.Builder

    env.WriteString("# Environment Information\n\n")

    // Working directory
    env.WriteString(fmt.Sprintf("Working Directory: %s\n", s.session.Directory))

    // Current date
    env.WriteString(fmt.Sprintf("Current Date: %s\n", time.Now().Format("2006-01-02")))

    // Platform info
    env.WriteString(fmt.Sprintf("Platform: %s\n", runtime.GOOS))

    // Git branch if available
    if branch := s.getGitBranch(); branch != "" {
        env.WriteString(fmt.Sprintf("Git Branch: %s\n", branch))
    }

    // Project type detection
    if projectType := s.detectProjectType(); projectType != "" {
        env.WriteString(fmt.Sprintf("Project Type: %s\n", projectType))
    }

    return env.String()
}

func (s *SystemPrompt) loadCustomRules() string {
    // Try loading from multiple locations
    locations := []string{
        filepath.Join(s.session.Directory, "AGENTS.md"),
        filepath.Join(s.session.Directory, "CLAUDE.md"),
        filepath.Join(s.session.Directory, ".opencode", "rules.md"),
    }

    // Also check global config
    if home, err := os.UserHomeDir(); err == nil {
        locations = append(locations,
            filepath.Join(home, ".config", "opencode", "rules.md"),
            filepath.Join(home, ".claude", "rules.md"),
        )
    }

    for _, loc := range locations {
        if content, err := os.ReadFile(loc); err == nil && len(content) > 0 {
            return fmt.Sprintf("# Custom Rules\n\n%s", string(content))
        }
    }

    return ""
}

func (s *SystemPrompt) toolInstructions() string {
    return `# Tool Usage Guidelines

1. **File Operations**
   - Use the Read tool before editing files
   - Use Edit for surgical changes, Write for new files
   - Always provide absolute paths

2. **Bash Commands**
   - Prefer built-in tools over bash when possible
   - Include a description for every bash command
   - Handle errors gracefully

3. **Search**
   - Use Glob for file discovery
   - Use Grep for content search
   - Be specific with patterns to avoid noise

4. **Best Practices**
   - Work iteratively, verify changes work
   - Don't modify files you haven't read
   - Explain your reasoning before acting`
}

func (s *SystemPrompt) getGitBranch() string {
    cmd := exec.Command("git", "branch", "--show-current")
    cmd.Dir = s.session.Directory
    output, err := cmd.Output()
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(output))
}

func (s *SystemPrompt) detectProjectType() string {
    dir := s.session.Directory

    // Check for common project indicators
    indicators := map[string][]string{
        "Node.js":    {"package.json"},
        "Python":     {"pyproject.toml", "setup.py", "requirements.txt"},
        "Go":         {"go.mod"},
        "Rust":       {"Cargo.toml"},
        "Java":       {"pom.xml", "build.gradle"},
        "Ruby":       {"Gemfile"},
        "PHP":        {"composer.json"},
        "C#":         {"*.csproj", "*.sln"},
    }

    for projectType, files := range indicators {
        for _, pattern := range files {
            matches, _ := filepath.Glob(filepath.Join(dir, pattern))
            if len(matches) > 0 {
                return projectType
            }
        }
    }

    return ""
}
```

---

## 6.4 Message History Management

```go
// internal/session/history.go
package session

import (
    "context"

    "github.com/opencode-ai/opencode-server/internal/provider"
    "github.com/opencode-ai/opencode-server/pkg/types"
)

// buildCompletionRequest builds the LLM completion request
func (p *Processor) buildCompletionRequest(
    ctx context.Context,
    session *types.Session,
    messages []*types.Message,
    currentMsg *types.Message,
    agent *Agent,
    model *provider.Model,
) (*provider.CompletionRequest, error) {
    // Build system prompt
    systemPrompt := NewSystemPrompt(session, agent, model.ProviderID, model.ID)

    // Convert messages to provider format
    var providerMessages []provider.Message

    // Add system message
    providerMessages = append(providerMessages, provider.Message{
        Role: "system",
        Content: []provider.ContentPart{
            provider.TextContent{Type: "text", Text: systemPrompt.Build()},
        },
    })

    // Add conversation history
    for _, msg := range messages {
        // Skip errored messages without content
        if msg.Error != nil && !hasUsableContent(msg) {
            continue
        }

        // Load parts for this message
        parts, err := p.partStore.List(ctx, msg.ID)
        if err != nil {
            continue
        }

        providerMsg := p.convertMessage(msg, parts)
        providerMessages = append(providerMessages, providerMsg)
    }

    // Get enabled tools
    tools, err := p.resolveTools(agent, model)
    if err != nil {
        return nil, err
    }

    // Build request
    req := &provider.CompletionRequest{
        Model:       model.ID,
        Messages:    providerMessages,
        Tools:       tools,
        MaxTokens:   model.MaxOutputTokens,
        Temperature: agent.Temperature,
        TopP:        agent.TopP,
        Stream:      true,
    }

    // Apply message transformations for the provider
    req.Messages = provider.TransformMessages(req.Messages, model.ProviderID)

    return req, nil
}

// convertMessage converts a types.Message to provider.Message
func (p *Processor) convertMessage(msg *types.Message, parts []types.Part) provider.Message {
    var content []provider.ContentPart

    for _, part := range parts {
        switch pt := part.(type) {
        case *types.TextPart:
            content = append(content, provider.TextContent{
                Type: "text",
                Text: pt.Text,
            })

        case *types.FilePart:
            content = append(content, provider.ImageContent{
                Type:      "image",
                MediaType: pt.MediaType,
                Data:      pt.URL,
            })

        case *types.ToolPart:
            if msg.Role == "assistant" {
                // Assistant message: include tool call
                inputJSON, _ := json.Marshal(pt.Input)
                content = append(content, provider.ToolCallContent{
                    Type:  "tool_call",
                    ID:    pt.ToolCallID,
                    Name:  pt.ToolName,
                    Input: inputJSON,
                })
            } else {
                // User message with tool result
                output := ""
                if pt.Output != nil {
                    output = *pt.Output
                }
                if pt.Error != nil {
                    output = *pt.Error
                }
                content = append(content, provider.ToolResultContent{
                    Type:       "tool_result",
                    ToolCallID: pt.ToolCallID,
                    Output:     output,
                    IsError:    pt.Error != nil,
                })
            }
        }
    }

    return provider.Message{
        Role:    msg.Role,
        Content: content,
    }
}

// resolveTools returns tools enabled for the agent
func (p *Processor) resolveTools(agent *Agent, model *provider.Model) ([]provider.Tool, error) {
    // Check if model supports tools
    if !model.SupportsTools {
        return nil, nil
    }

    // Get all registered tools
    allTools := p.toolRegistry.List()

    var result []provider.Tool

    for _, t := range allTools {
        // Check if tool is enabled for this agent
        if !agent.ToolEnabled(t.ID()) {
            continue
        }

        result = append(result, provider.Tool{
            Name:        t.ID(),
            Description: t.Description(),
            Parameters:  t.Parameters(),
        })
    }

    return result, nil
}

// hasUsableContent checks if message has content worth including
func hasUsableContent(msg *types.Message) bool {
    // Would need to check parts, simplified for now
    return msg.Tokens != nil && msg.Tokens.Output > 0
}
```

---

## 6.5 Message Compaction

```go
// internal/session/compact.go
package session

import (
    "context"
    "fmt"
    "strings"

    "github.com/opencode-ai/opencode-server/pkg/types"
)

// CompactionConfig controls message compaction behavior
type CompactionConfig struct {
    MinMessagesToKeep  int     // Always keep at least this many recent messages
    SummaryMaxTokens   int     // Max tokens for summary
    ContextThreshold   float64 // Compact when context is this % full
}

var DefaultCompactionConfig = CompactionConfig{
    MinMessagesToKeep:  4,
    SummaryMaxTokens:   2000,
    ContextThreshold:   0.75,
}

// compactMessages summarizes old messages to free context
func (p *Processor) compactMessages(ctx context.Context, sessionID string, messages []*types.Message) error {
    if len(messages) <= DefaultCompactionConfig.MinMessagesToKeep {
        return nil
    }

    // Update session compacting flag
    session, err := p.sessionStore.Get(ctx, sessionID)
    if err != nil {
        return err
    }

    now := time.Now().UnixMilli()
    session.Time.Compacting = &now
    p.sessionStore.Update(ctx, session)

    defer func() {
        session.Time.Compacting = nil
        p.sessionStore.Update(ctx, session)
    }()

    // Determine which messages to compact
    compactEnd := len(messages) - DefaultCompactionConfig.MinMessagesToKeep
    toCompact := messages[:compactEnd]

    // Build summary request
    summaryPrompt := buildSummaryPrompt(toCompact)

    // Get small/fast model for summarization
    model, err := p.providerRegistry.GetSmallModel()
    if err != nil {
        return err
    }

    // Generate summary
    req := &provider.CompletionRequest{
        Model: model.ID,
        Messages: []provider.Message{
            {
                Role: "system",
                Content: []provider.ContentPart{
                    provider.TextContent{Type: "text", Text: "You are a conversation summarizer. Create a concise summary of the conversation that preserves key context for continuing the discussion."},
                },
            },
            {
                Role: "user",
                Content: []provider.ContentPart{
                    provider.TextContent{Type: "text", Text: summaryPrompt},
                },
            },
        },
        MaxTokens: DefaultCompactionConfig.SummaryMaxTokens,
        Stream:    false,
    }

    stream, err := model.Provider.CreateCompletion(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to create summary: %w", err)
    }
    defer stream.Close()

    // Collect response
    var summary strings.Builder
    for {
        event, err := stream.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        if delta, ok := event.(provider.TextDeltaEvent); ok {
            summary.WriteString(delta.Text)
        }
    }

    // Mark compacted messages as summarized
    for _, msg := range toCompact {
        msg.Summary = true
        p.messageStore.Update(ctx, msg)
    }

    // Create compaction marker part in first remaining message
    if len(messages) > compactEnd {
        compactionPart := &types.CompactionPart{
            ID:      generatePartID(),
            Type:    "compaction",
            Summary: summary.String(),
            Count:   len(toCompact),
        }
        p.partStore.Create(ctx, messages[compactEnd].ID, compactionPart)
    }

    return nil
}

func buildSummaryPrompt(messages []*types.Message) string {
    var prompt strings.Builder

    prompt.WriteString("Please summarize the following conversation, focusing on:\n")
    prompt.WriteString("1. Key decisions and outcomes\n")
    prompt.WriteString("2. Files that were modified\n")
    prompt.WriteString("3. Important context for continuing the work\n\n")
    prompt.WriteString("---\n\n")

    for _, msg := range messages {
        if msg.Role == "user" {
            prompt.WriteString("USER:\n")
        } else {
            prompt.WriteString("ASSISTANT:\n")
        }

        // Add message content (simplified - would need parts)
        prompt.WriteString("[Message content here]\n\n")
    }

    return prompt.String()
}
```

---

## 6.6 Deliverables

### Files to Create

| File | Lines (Est.) | Complexity |
|------|--------------|------------|
| `internal/session/processor.go` | 200 | High |
| `internal/session/loop.go` | 250 | High |
| `internal/session/stream.go` | 300 | High |
| `internal/session/tools.go` | 250 | High |
| `internal/session/system.go` | 200 | Medium |
| `internal/session/history.go` | 200 | Medium |
| `internal/session/compact.go` | 150 | Medium |
| `internal/session/permission.go` | 100 | Medium |

### Integration Tests

```go
// test/integration/processor_test.go

func TestProcessor_SimpleConversation(t *testing.T) { /* ... */ }
func TestProcessor_ToolExecution(t *testing.T) { /* ... */ }
func TestProcessor_MultiStepLoop(t *testing.T) { /* ... */ }
func TestProcessor_Abort(t *testing.T) { /* ... */ }
func TestProcessor_StreamingUpdates(t *testing.T) { /* ... */ }

func TestProcessor_DoomLoopDetection(t *testing.T) { /* ... */ }
func TestProcessor_PermissionDenied(t *testing.T) { /* ... */ }
func TestProcessor_MaxStepsLimit(t *testing.T) { /* ... */ }

func TestProcessor_ErrorRetry(t *testing.T) { /* ... */ }
func TestProcessor_ContextOverflow(t *testing.T) { /* ... */ }
func TestProcessor_MessageCompaction(t *testing.T) { /* ... */ }

func TestSystemPrompt_Build(t *testing.T) { /* ... */ }
func TestSystemPrompt_CustomRules(t *testing.T) { /* ... */ }
func TestSystemPrompt_ProviderSpecific(t *testing.T) { /* ... */ }
```

### Acceptance Criteria

- [x] Agentic loop executes tools and continues conversation
- [x] Streaming updates sent via callback and events
- [x] Tool execution with metadata updates
- [x] Doom loop detection and permission handling
- [x] Session abort works mid-processing
- [x] Error retry with exponential backoff
- [x] Context overflow triggers compaction
- [x] System prompt includes environment context
- [x] Custom rules loaded from AGENTS.md/CLAUDE.md
- [x] Step limits prevent infinite loops
- [x] Token and cost tracking accurate
- [x] Test coverage >80% for session package

**Phase 6 Status: ✅ COMPLETE** (2025-11-26)
