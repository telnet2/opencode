# Go LLM SDK Evaluation for OpenCode Server

## Executive Summary

This document evaluates Go LLM frameworks as potential replacements for the Vercel AI SDK in the OpenCode server implementation. We analyzed two major frameworks:

1. **Google ADK-Go** (`google.golang.org/adk`) - Agent Development Kit
2. **CloudWeGo Eino** (`github.com/cloudwego/eino`) - LLM Application Framework

**Recommendation**: **Eino is the recommended choice** for the OpenCode Go implementation. It provides comprehensive multi-provider support, streaming, tool calling, MCP integration, and agent orchestration - all features needed for OpenCode.

---

## Framework Comparison Overview

| Feature | Vercel AI SDK | ADK-Go | Eino |
|---------|---------------|--------|------|
| **Multi-Provider Support** | 10+ providers | Gemini only | 10+ providers |
| **Streaming** | streamText() | GenerateContent(stream) | Stream() |
| **Tool Calling** | tool() | functiontool.New() | InvokableTool |
| **MCP Integration** | @ai-sdk/mcp | mcptoolset | officialmcp |
| **Agent Framework** | External | Built-in | ReAct, Workflows |
| **Graph Orchestration** | External | Basic | Chain, Graph, Workflow |
| **Cache Control** | Built-in | None | Built-in (Claude) |
| **Extended Thinking** | Built-in | None | Built-in (Claude) |
| **AWS Bedrock** | Built-in | None | Built-in (Claude) |
| **Callbacks/Tracing** | Basic | Callbacks | Comprehensive aspects |
| **Production Ready** | Yes | Alpha | Yes (ByteDance) |

---

## 1. Eino Framework Analysis

### 1.1 Why Eino?

Eino (pronounced "I know") is developed by ByteDance/CloudWeGo and provides:

- **Comprehensive provider support**: OpenAI, Claude, Gemini, Ollama, DeepSeek, Qwen, and more
- **Production-tested**: Used in ByteDance production systems
- **Go-idiomatic**: Follows Go conventions with strong type checking
- **Feature-rich**: Streaming, tools, MCP, agents, graph orchestration
- **Active development**: Regular updates and community support

### 1.2 Eino Architecture

```
github.com/cloudwego/eino/
├── schema/              # Core types (Message, Tool, Stream)
├── components/
│   ├── model/          # ChatModel interface
│   └── tool/           # Tool interfaces
├── compose/            # Graph orchestration
├── flow/agent/         # Agent implementations
│   └── react/          # ReAct agent
├── callbacks/          # Aspect-oriented handlers
└── adk/                # Agent Development Kit

github.com/cloudwego/eino-ext/
├── components/model/   # Provider implementations
│   ├── openai/         # OpenAI/Azure
│   ├── claude/         # Anthropic/Bedrock
│   ├── gemini/         # Google
│   ├── ollama/         # Ollama
│   ├── deepseek/       # DeepSeek
│   └── qwen/           # Alibaba Qwen
├── components/tool/
│   └── mcp/            # MCP integration
└── callbacks/          # Tracing handlers (Langfuse)
```

### 1.3 Key Interfaces

```go
// ChatModel - Core model interface
type BaseChatModel interface {
    Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)
}

// ToolCallingChatModel - Model with tool support
type ToolCallingChatModel interface {
    BaseChatModel
    WithTools(tools []*schema.ToolInfo) (ToolCallingChatModel, error)
}

// Tool - Tool interface
type InvokableTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}
```

---

## 2. Feature Parity Analysis

### 2.1 Multi-Provider Support

**Vercel AI SDK:**
```typescript
import { createAnthropic } from "@ai-sdk/anthropic"
import { createOpenAI } from "@ai-sdk/openai"
const model = createAnthropic({ apiKey })("claude-sonnet-4")
```

**Eino Equivalent:**
```go
import (
    "github.com/cloudwego/eino-ext/components/model/claude"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

// Claude/Anthropic
claudeModel, err := claude.NewChatModel(ctx, &claude.Config{
    APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
    Model:     "claude-sonnet-4-20250514",
    MaxTokens: 8192,
})

// Claude via AWS Bedrock
bedrockModel, err := claude.NewChatModel(ctx, &claude.Config{
    ByBedrock:       true,
    Region:          "us-east-1",
    AccessKey:       os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
    Model:           "anthropic.claude-sonnet-4-20250514-v1:0",
    MaxTokens:       8192,
})

// OpenAI
openaiModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})

// Azure OpenAI
azureModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
    ByAzure:    true,
    APIKey:     os.Getenv("AZURE_OPENAI_API_KEY"),
    BaseURL:    "https://your-resource.openai.azure.com",
    APIVersion: "2024-02-15-preview",
    Model:      "gpt-4o",
})
```

### 2.2 Streaming Text Generation

**Vercel AI SDK:**
```typescript
const result = await streamText({
    model: languageModel,
    messages: messages,
    tools: tools,
    maxOutputTokens: 32000,
})

for await (const chunk of result.textStream) {
    process.stdout.write(chunk)
}
```

**Eino Equivalent:**
```go
// Stream generates streaming response
stream, err := model.Stream(ctx, messages, model.WithMaxTokens(32000))
if err != nil {
    return err
}
defer stream.Close()

for {
    msg, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    fmt.Print(msg.Content)
}

// Or use ConcatMessageStream to get full message
fullMsg, err := schema.ConcatMessageStream(stream)
```

### 2.3 Tool Calling

**Vercel AI SDK:**
```typescript
import { tool, jsonSchema } from "ai"

const readTool = tool({
    id: "read",
    description: "Read a file",
    inputSchema: jsonSchema({
        type: "object",
        properties: { file_path: { type: "string" } },
        required: ["file_path"],
    }),
    async execute(args) {
        return { output: await fs.readFile(args.file_path, "utf-8") }
    },
})
```

**Eino Equivalent:**
```go
import (
    "github.com/cloudwego/eino/components/tool"
    toolutils "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/schema"
)

// Using InvokableLambda for simple tools
type ReadArgs struct {
    FilePath string `json:"file_path" jsonschema:"description=The file path to read"`
}

readTool := toolutils.InvokableLambda(func(ctx context.Context, args *ReadArgs) (string, error) {
    content, err := os.ReadFile(args.FilePath)
    if err != nil {
        return "", err
    }
    return string(content), nil
}, toolutils.WithToolName("read"), toolutils.WithToolDesc("Read a file"))

// Or implement the interface directly
type ReadTool struct{}

func (t *ReadTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "read",
        Desc: "Read a file from the filesystem",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "file_path": {
                Type:     schema.String,
                Desc:     "The absolute path to the file to read",
                Required: true,
            },
        }),
    }, nil
}

func (t *ReadTool) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
    var args ReadArgs
    if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
        return "", err
    }
    content, err := os.ReadFile(args.FilePath)
    if err != nil {
        return "", err
    }
    return string(content), nil
}

// Bind tools to model
modelWithTools, err := model.WithTools([]*schema.ToolInfo{toolInfo})
```

### 2.4 Cache Control (Anthropic Ephemeral Cache)

**Vercel AI SDK:**
```typescript
const result = await streamText({
    model,
    messages,
    providerOptions: {
        anthropic: { cacheControl: { type: "ephemeral" } }
    }
})
```

**Eino Equivalent:**
```go
import "github.com/cloudwego/eino-ext/components/model/claude"

// Eino's Claude implementation supports automatic cache control
// Set breakpoints on messages via Extra field
msg := &schema.Message{
    Role:    schema.System,
    Content: systemPrompt,
    Extra: map[string]any{
        claude.ExtraKeyBreakpoint: true, // Mark for ephemeral caching
    },
}

// Or enable auto-caching for system messages and tools
model.Stream(ctx, messages, claude.WithEnableAutoCache(true))
```

### 2.5 Extended Thinking (Claude)

**Vercel AI SDK:**
```typescript
const result = await streamText({
    model,
    messages,
    providerOptions: {
        anthropic: { thinking: { type: "enabled", budgetTokens: 10000 } }
    }
})
```

**Eino Equivalent:**
```go
import "github.com/cloudwego/eino-ext/components/model/claude"

// Configure thinking in model creation
model, err := claude.NewChatModel(ctx, &claude.Config{
    APIKey:    apiKey,
    Model:     "claude-sonnet-4-20250514",
    MaxTokens: 16000,
    Thinking: &claude.Thinking{
        Enable:       true,
        BudgetTokens: 10000,
    },
})

// Or configure per-request
model.Stream(ctx, messages, claude.WithThinking(&claude.Thinking{
    Enable:       true,
    BudgetTokens: 10000,
}))

// Access thinking content from response
msg, _ := model.Generate(ctx, messages)
thinking, ok := claude.GetThinking(msg)
// msg.ReasoningContent also contains thinking
```

### 2.6 MCP Integration

**Vercel AI SDK:**
```typescript
import { experimental_createMCPClient } from "@ai-sdk/mcp"

const mcpClient = await experimental_createMCPClient({
    name: "server",
    transport: new StdioClientTransport({ command: "npx", args: ["-y", "server"] }),
})
const tools = await mcpClient.tools()
```

**Eino Equivalent:**
```go
import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
)

// Create MCP client session
cli, err := mcp.NewStdioClient(ctx, mcp.StdioClientParams{
    Command: "npx",
    Args:    []string{"-y", "server"},
})
if err != nil {
    return err
}
defer cli.Close()

// Initialize connection
_, err = cli.Initialize(ctx, &mcp.InitializeParams{
    ClientInfo: mcp.ClientInfo{Name: "opencode", Version: "1.0"},
})
if err != nil {
    return err
}

// Get tools from MCP server
tools, err := officialmcp.GetTools(ctx, &officialmcp.Config{
    Cli: cli,
    // Optionally filter tools
    ToolNameList: []string{"read_file", "write_file"},
    // Optional result handler
    ToolCallResultHandler: func(ctx context.Context, name string, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
        // Custom processing
        return result, nil
    },
})
```

### 2.7 ReAct Agent

**Eino provides a built-in ReAct agent:**

```go
import (
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/flow/agent/react"
)

// Create ReAct agent
agent, err := react.NewAgent(ctx, &react.AgentConfig{
    ToolCallingModel: model,
    ToolsConfig: compose.ToolsNodeConfig{
        Tools: []tool.BaseTool{readTool, writeTool, bashTool},
    },
    MaxStep: 20,
    MessageModifier: func(ctx context.Context, msgs []*schema.Message) []*schema.Message {
        // Add system prompt
        return append([]*schema.Message{schema.SystemMessage(systemPrompt)}, msgs...)
    },
    // Tools that return directly without model loop
    ToolReturnDirectly: map[string]struct{}{
        "final_answer": {},
    },
})

// Generate response
response, err := agent.Generate(ctx, []*schema.Message{
    schema.UserMessage("Help me write a function that calculates fibonacci"),
})

// Or stream
stream, err := agent.Stream(ctx, messages)
```

### 2.8 Graph Orchestration

**Eino provides powerful graph orchestration:**

```go
import "github.com/cloudwego/eino/compose"

// Create a graph for complex workflows
graph := compose.NewGraph[map[string]any, *schema.Message]()

// Add nodes
graph.AddChatTemplateNode("template", chatTemplate)
graph.AddChatModelNode("model", chatModel)
graph.AddToolsNode("tools", toolsNode)
graph.AddLambdaNode("converter", convertFunc)

// Add edges
graph.AddEdge(compose.START, "template")
graph.AddEdge("template", "model")
graph.AddBranch("model", branch) // Conditional branching
graph.AddEdge("tools", "converter")
graph.AddEdge("converter", compose.END)

// Compile and run
runnable, err := graph.Compile(ctx)
result, err := runnable.Invoke(ctx, input)
```

### 2.9 Callbacks/Aspects for Tracing

```go
import "github.com/cloudwego/eino/callbacks"

// Create callback handler
handler := callbacks.NewHandlerBuilder().
    OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
        log.Printf("Starting %s: %v", info.Name, input)
        return ctx
    }).
    OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
        log.Printf("Completed %s: %v", info.Name, output)
        return ctx
    }).
    OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
        log.Printf("Error in %s: %v", info.Name, err)
        return ctx
    }).
    Build()

// Use with model
model.Generate(ctx, messages, model.WithCallbacks(handler))

// Or use with graph
graph.Invoke(ctx, input, compose.WithCallbacks(handler))
```

---

## 3. Implementation Guide for OpenCode

### 3.1 Provider Abstraction Layer

```go
package provider

import (
    "context"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
    "github.com/cloudwego/eino-ext/components/model/claude"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

// Provider wraps an Eino ChatModel with additional metadata
type Provider struct {
    ID          string
    Name        string
    ChatModel   model.ToolCallingChatModel
}

// ProviderConfig holds configuration for creating providers
type ProviderConfig struct {
    ID          string
    Type        string // "anthropic", "openai", "bedrock", etc.
    APIKey      string
    BaseURL     string
    Model       string
    MaxTokens   int

    // Anthropic-specific
    Thinking    *claude.Thinking

    // Bedrock-specific
    Region      string
    Profile     string
}

// Registry manages provider instances
type Registry struct {
    providers map[string]*Provider
}

func NewRegistry() *Registry {
    return &Registry{providers: make(map[string]*Provider)}
}

func (r *Registry) Register(ctx context.Context, cfg ProviderConfig) error {
    var chatModel model.ToolCallingChatModel
    var err error

    switch cfg.Type {
    case "anthropic":
        chatModel, err = claude.NewChatModel(ctx, &claude.Config{
            APIKey:    cfg.APIKey,
            BaseURL:   &cfg.BaseURL,
            Model:     cfg.Model,
            MaxTokens: cfg.MaxTokens,
            Thinking:  cfg.Thinking,
        })

    case "bedrock":
        chatModel, err = claude.NewChatModel(ctx, &claude.Config{
            ByBedrock: true,
            Region:    cfg.Region,
            Profile:   cfg.Profile,
            Model:     cfg.Model,
            MaxTokens: cfg.MaxTokens,
            Thinking:  cfg.Thinking,
        })

    case "openai":
        maxTokens := cfg.MaxTokens
        cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
            APIKey:    cfg.APIKey,
            BaseURL:   cfg.BaseURL,
            Model:     cfg.Model,
            MaxTokens: &maxTokens,
        })
        if err != nil {
            return err
        }
        chatModel = cm

    case "azure":
        maxTokens := cfg.MaxTokens
        cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
            ByAzure:    true,
            APIKey:     cfg.APIKey,
            BaseURL:    cfg.BaseURL,
            APIVersion: "2024-02-15-preview",
            Model:      cfg.Model,
            MaxTokens:  &maxTokens,
        })
        if err != nil {
            return err
        }
        chatModel = cm

    default:
        return fmt.Errorf("unknown provider type: %s", cfg.Type)
    }

    if err != nil {
        return err
    }

    r.providers[cfg.ID] = &Provider{
        ID:        cfg.ID,
        Name:      cfg.Type,
        ChatModel: chatModel,
    }

    return nil
}

func (r *Registry) Get(id string) (*Provider, bool) {
    p, ok := r.providers[id]
    return p, ok
}
```

### 3.2 Tool System Integration

```go
package tools

import (
    "context"
    "encoding/json"
    "os"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
)

// BaseTool extends Eino's tool interface with OpenCode-specific features
type BaseTool interface {
    tool.InvokableTool

    // Additional OpenCode features
    RequiresPermission() bool
    Category() string
}

// ReadTool implements file reading
type ReadTool struct{}

func (t *ReadTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "Read",
        Desc: "Reads a file from the local filesystem. Returns the file contents with line numbers.",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "file_path": {
                Type:     schema.String,
                Desc:     "The absolute path to the file to read",
                Required: true,
            },
            "offset": {
                Type: schema.Integer,
                Desc: "The line number to start reading from (1-indexed)",
            },
            "limit": {
                Type: schema.Integer,
                Desc: "The number of lines to read",
            },
        }),
    }, nil
}

func (t *ReadTool) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
    var args struct {
        FilePath string `json:"file_path"`
        Offset   int    `json:"offset"`
        Limit    int    `json:"limit"`
    }

    if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
        return "", err
    }

    content, err := os.ReadFile(args.FilePath)
    if err != nil {
        return "", err
    }

    // Apply offset/limit logic
    lines := strings.Split(string(content), "\n")
    if args.Offset > 0 {
        if args.Offset > len(lines) {
            lines = []string{}
        } else {
            lines = lines[args.Offset-1:]
        }
    }
    if args.Limit > 0 && args.Limit < len(lines) {
        lines = lines[:args.Limit]
    }

    // Format with line numbers
    var result strings.Builder
    startLine := 1
    if args.Offset > 0 {
        startLine = args.Offset
    }
    for i, line := range lines {
        fmt.Fprintf(&result, "%d\t%s\n", startLine+i, line)
    }

    return result.String(), nil
}

func (t *ReadTool) RequiresPermission() bool { return false }
func (t *ReadTool) Category() string         { return "filesystem" }

// BashTool implements shell command execution
type BashTool struct {
    sandbox bool
}

func (t *BashTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "Bash",
        Desc: "Executes a bash command in a persistent shell session",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "command": {
                Type:     schema.String,
                Desc:     "The command to execute",
                Required: true,
            },
            "timeout": {
                Type: schema.Integer,
                Desc: "Optional timeout in milliseconds (max 600000)",
            },
        }),
    }, nil
}

func (t *BashTool) InvokableRun(ctx context.Context, argsJSON string, opts ...tool.Option) (string, error) {
    var args struct {
        Command string `json:"command"`
        Timeout int    `json:"timeout"`
    }

    if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
        return "", err
    }

    // Execute command with timeout
    timeout := time.Duration(args.Timeout) * time.Millisecond
    if timeout == 0 {
        timeout = 120 * time.Second
    }

    execCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    cmd := exec.CommandContext(execCtx, "bash", "-c", args.Command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return string(output) + "\n" + err.Error(), nil
    }

    return string(output), nil
}

func (t *BashTool) RequiresPermission() bool { return true }
func (t *BashTool) Category() string         { return "execution" }

// Registry for OpenCode tools
type Registry struct {
    tools map[string]BaseTool
}

func NewRegistry() *Registry {
    return &Registry{tools: make(map[string]BaseTool)}
}

func (r *Registry) Register(t BaseTool) {
    info, _ := t.Info(context.Background())
    r.tools[info.Name] = t
}

func (r *Registry) GetEinoTools(ctx context.Context) []tool.BaseTool {
    result := make([]tool.BaseTool, 0, len(r.tools))
    for _, t := range r.tools {
        result = append(result, t)
    }
    return result
}

func (r *Registry) GetToolInfos(ctx context.Context) ([]*schema.ToolInfo, error) {
    result := make([]*schema.ToolInfo, 0, len(r.tools))
    for _, t := range r.tools {
        info, err := t.Info(ctx)
        if err != nil {
            return nil, err
        }
        result = append(result, info)
    }
    return result, nil
}
```

### 3.3 Session/Message Management

```go
package session

import (
    "github.com/cloudwego/eino/schema"
)

// Session represents a conversation session
type Session struct {
    ID        string        `json:"id"`
    ProjectID string        `json:"projectID"`
    Title     string        `json:"title"`
    Messages  []*Message    `json:"messages"`
    CreatedAt int64         `json:"createdAt"`
    UpdatedAt int64         `json:"updatedAt"`
}

// Message wraps Eino's schema.Message with OpenCode metadata
type Message struct {
    *schema.Message

    ID         string         `json:"id"`
    SessionID  string         `json:"sessionID"`
    ParentID   *string        `json:"parentID,omitempty"`

    // OpenCode-specific fields
    Agent      string         `json:"agent,omitempty"`
    ModelID    string         `json:"modelID,omitempty"`
    ProviderID string         `json:"providerID,omitempty"`
    Cost       float64        `json:"cost,omitempty"`

    // Parts for structured content
    Parts      []Part         `json:"parts,omitempty"`

    Time       MessageTime    `json:"time"`
}

// Part represents a message component (text, tool call, etc.)
type Part interface {
    PartType() string
}

// TextPart for text content
type TextPart struct {
    ID   string `json:"id"`
    Type string `json:"type"` // "text"
    Text string `json:"text"`
}

// ToolPart for tool invocations
type ToolPart struct {
    ID     string    `json:"id"`
    Type   string    `json:"type"` // "tool"
    CallID string    `json:"callID"`
    Tool   string    `json:"tool"`
    State  ToolState `json:"state"`
}

// Convert Eino Message to OpenCode Message
func FromEinoMessage(msg *schema.Message, sessionID string) *Message {
    m := &Message{
        Message:   msg,
        ID:        generateID(),
        SessionID: sessionID,
        Time: MessageTime{
            Created: time.Now().UnixMilli(),
        },
    }

    // Extract parts from message content
    if msg.Content != "" {
        m.Parts = append(m.Parts, &TextPart{
            ID:   generateID(),
            Type: "text",
            Text: msg.Content,
        })
    }

    // Extract tool calls
    for _, tc := range msg.ToolCalls {
        m.Parts = append(m.Parts, &ToolPart{
            ID:     generateID(),
            Type:   "tool",
            CallID: tc.ID,
            Tool:   tc.Function.Name,
            State: ToolState{
                Status: "pending",
                Input:  json.RawMessage(tc.Function.Arguments),
            },
        })
    }

    return m
}

// ToEinoMessages converts OpenCode messages for Eino
func ToEinoMessages(messages []*Message) []*schema.Message {
    result := make([]*schema.Message, len(messages))
    for i, m := range messages {
        result[i] = m.Message
    }
    return result
}
```

### 3.4 Main Agent Loop

```go
package agent

import (
    "context"
    "io"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

// Agent handles the main conversation loop
type Agent struct {
    provider *provider.Provider
    tools    *tools.Registry
    session  *session.Session
}

// StreamResponse generates a streaming response
func (a *Agent) StreamResponse(ctx context.Context, userMessage string) (<-chan *StreamEvent, error) {
    events := make(chan *StreamEvent, 100)

    go func() {
        defer close(events)

        // Add user message to session
        userMsg := schema.UserMessage(userMessage)
        a.session.Messages = append(a.session.Messages, session.FromEinoMessage(userMsg, a.session.ID))

        // Get tool infos and bind to model
        toolInfos, err := a.tools.GetToolInfos(ctx)
        if err != nil {
            events <- &StreamEvent{Type: "error", Error: err}
            return
        }

        modelWithTools, err := a.provider.ChatModel.WithTools(toolInfos)
        if err != nil {
            events <- &StreamEvent{Type: "error", Error: err}
            return
        }

        // Convert messages for Eino
        messages := session.ToEinoMessages(a.session.Messages)

        // Agent loop
        for {
            // Stream response from model
            stream, err := modelWithTools.Stream(ctx, messages)
            if err != nil {
                events <- &StreamEvent{Type: "error", Error: err}
                return
            }

            var fullMsg *schema.Message
            for {
                chunk, err := stream.Recv()
                if err == io.EOF {
                    break
                }
                if err != nil {
                    events <- &StreamEvent{Type: "error", Error: err}
                    stream.Close()
                    return
                }

                // Send text chunks
                if chunk.Content != "" {
                    events <- &StreamEvent{Type: "text", Text: chunk.Content}
                }

                // Accumulate full message
                if fullMsg == nil {
                    fullMsg = chunk
                } else {
                    fullMsg, _ = schema.ConcatMessages([]*schema.Message{fullMsg, chunk})
                }
            }
            stream.Close()

            // Add assistant message to session
            a.session.Messages = append(a.session.Messages, session.FromEinoMessage(fullMsg, a.session.ID))
            messages = append(messages, fullMsg)

            // Check for tool calls
            if len(fullMsg.ToolCalls) == 0 {
                // No tool calls, we're done
                events <- &StreamEvent{Type: "complete"}
                return
            }

            // Execute tools
            var toolResults []*schema.Message
            for _, tc := range fullMsg.ToolCalls {
                events <- &StreamEvent{Type: "tool_start", ToolCall: &tc}

                t, ok := a.tools.Get(tc.Function.Name)
                if !ok {
                    result := schema.ToolMessage(
                        fmt.Sprintf("Tool not found: %s", tc.Function.Name),
                        tc.ID,
                    )
                    toolResults = append(toolResults, result)
                    continue
                }

                output, err := t.InvokableRun(ctx, tc.Function.Arguments)
                if err != nil {
                    result := schema.ToolMessage(err.Error(), tc.ID)
                    toolResults = append(toolResults, result)
                } else {
                    result := schema.ToolMessage(output, tc.ID)
                    toolResults = append(toolResults, result)
                }

                events <- &StreamEvent{Type: "tool_end", ToolCall: &tc, Output: output}
            }

            // Add tool results to messages
            for _, tr := range toolResults {
                a.session.Messages = append(a.session.Messages, session.FromEinoMessage(tr, a.session.ID))
                messages = append(messages, tr)
            }
        }
    }()

    return events, nil
}

// StreamEvent represents an event in the response stream
type StreamEvent struct {
    Type     string           `json:"type"` // "text", "tool_start", "tool_end", "error", "complete"
    Text     string           `json:"text,omitempty"`
    ToolCall *schema.ToolCall `json:"toolCall,omitempty"`
    Output   string           `json:"output,omitempty"`
    Error    error            `json:"error,omitempty"`
}
```

---

## 4. ADK-Go Analysis (For Reference)

ADK-Go is Google's Agent Development Kit for Go. While it provides useful patterns, it has limitations compared to Eino:

### 4.1 Limitations

| Limitation | Impact |
|------------|--------|
| Gemini only | Cannot use Claude, OpenAI |
| No cache control | Higher API costs |
| No extended thinking | Limited for complex reasoning |
| Less mature | Fewer production deployments |
| Limited streaming | Different patterns than Eino |

### 4.2 Useful Patterns to Borrow

1. **Iterator-based streaming** (`iter.Seq2`)
2. **Agent interface design**
3. **REST API handler patterns** from `adkrest`

---

## 5. Recommendations

### 5.1 Primary Recommendation: Use Eino

Eino provides the most comprehensive feature parity with the Vercel AI SDK:

1. **Multi-provider support** - Claude, OpenAI, Gemini, Ollama, and more
2. **AWS Bedrock** - Native support for Claude on Bedrock
3. **Cache control** - Ephemeral caching for Claude
4. **Extended thinking** - Built-in support for reasoning
5. **MCP integration** - Official MCP SDK integration
6. **ReAct agent** - Built-in agent framework
7. **Graph orchestration** - Complex workflow support
8. **Production-tested** - Used at ByteDance scale

### 5.2 Implementation Phases

**Phase 1: Core Infrastructure**
- Provider registry with Eino models
- Tool system integration
- Session management

**Phase 2: Agent Loop**
- Streaming response handling
- Tool execution loop
- Message history management

**Phase 3: MCP Integration**
- MCP server connections
- Tool discovery and binding
- Custom transports

**Phase 4: Advanced Features**
- ReAct agent for complex tasks
- Graph orchestration for workflows
- Callback handlers for tracing

---

## 6. Conclusion

**Eino is the recommended framework** for implementing the OpenCode server in Go. It provides:

- Near feature parity with Vercel AI SDK
- Production-ready code from ByteDance
- Comprehensive provider support
- Strong type safety and Go idioms
- Active development and community

ADK-Go should be considered for reference patterns but not as the primary framework due to its Gemini-only limitation.

---

*Document Version: 2.0*
*Last Updated: 2025-11-26*
