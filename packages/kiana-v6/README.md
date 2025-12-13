# Kiana v6

A powerful headless coding agent built with Vercel AI SDK v6. Kiana can execute code, manipulate files, search the web, and much more through a rich set of tools.

## Features

- 🤖 **Headless Operation** - Runs non-interactively with sensible defaults
- 🔧 **Rich Tool Set** - File operations, bash execution, web search, and more
- 🔌 **MCP Support** - Connect to external MCP servers for extended functionality
- 📝 **Interactive Mode** - REPL-style interface with intuitive keybindings
- 🌊 **Streaming** - Real-time streaming responses with SSE protocol
- 💾 **Session Persistence** - Save and restore conversation state
- 📊 **Multiple Output Modes** - Human-readable or SSE format for integration

## Installation

```bash
npm install kiana-v6
# or
pnpm add kiana-v6
```

## Quick Start

### 1. Create a Config File

Generate a template configuration:

```bash
npx kiana-v6 --create-config > kiana.jsonc
```

Edit `kiana.jsonc` with your API key:

```jsonc
{
  "provider": {
    "type": "anthropic",
    "apiKey": "YOUR_API_KEY",
    "model": "claude-sonnet-4-20250514"
  }
}
```

### 2. Run Interactive Mode

```bash
npx kiana-v6 -i
```

### 3. Send a Prompt

```bash
npx kiana-v6 -H -p "List all TypeScript files in this project"
```

## Usage Modes

### Interactive Mode (Recommended)

```bash
kiana-v6 -i
```

Keybindings:
- **Enter** - Send message
- **Ctrl+J** - Insert newline
- **ESC ESC** - Cancel current operation (double-tap within 2s)
- **Ctrl+C** - Exit

### Single Prompt Mode

```bash
# Human-readable output
kiana-v6 -H -p "Your prompt here"

# JSON/SSE output
kiana-v6 -p "Your prompt here"
```

### JSON Protocol Mode

For integration with other tools:

```bash
echo '{"type":"message","text":"Your prompt"}' | kiana-v6
```

## Built-in Tools

Kiana comes with a comprehensive set of tools:

### File Operations
- **read** - Read file contents
- **write** - Write files (with safety checks)
- **edit** - Make precise edits with unified diffs
- **glob** - Find files by pattern
- **grep** - Search file contents with regex
- **list** - List directory contents

### Execution
- **bash** - Execute shell commands safely

### Web & Search
- **webfetch** - Fetch and convert web pages to markdown
- **websearch** - Search the web with Exa AI
- **codesearch** - Search for code examples and documentation

### Planning
- **todowrite** - Create and manage task lists
- **todoread** - Read current tasks

### Subagents
- **task** - Launch specialized subagents for complex tasks

## Model Context Protocol (MCP) Support

Kiana supports the Model Context Protocol, allowing you to connect to external MCP servers:

```jsonc
{
  "provider": { ... },
  "mcpServers": [
    {
      "name": "filesystem",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "."],
      "env": {}
    },
    {
      "name": "github",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token"
      }
    }
  ]
}
```

See [MCP_SUPPORT.md](./docs/MCP_SUPPORT.md) for detailed documentation.

## Configuration

### Provider Options

#### Anthropic
```jsonc
{
  "provider": {
    "type": "anthropic",
    "apiKey": "sk-ant-...",
    "model": "claude-sonnet-4-20250514"
  }
}
```

#### OpenAI
```jsonc
{
  "provider": {
    "type": "openai",
    "apiKey": "sk-...",
    "model": "gpt-4o"
  }
}
```

#### OpenAI-Compatible
```jsonc
{
  "provider": {
    "type": "openai-compatible",
    "apiKey": "...",
    "model": "...",
    "baseUrl": "https://api.example.com/v1"
  }
}
```

#### Google
```jsonc
{
  "provider": {
    "type": "google",
    "apiKey": "...",
    "model": "gemini-2.0-flash-exp"
  }
}
```

### Advanced Options

```jsonc
{
  "systemPrompt": "Custom system instructions...",
  "workingDirectory": "/path/to/project",
  "tools": ["read", "write", "bash"],  // Tool whitelist
  "maxSteps": 50,                      // Max agent loop iterations
  "maxRetries": 5,                     // Max API retries
  "streaming": true                     // Enable streaming
}
```

## Programmatic Usage

```typescript
import { CodingAgent, createLanguageModel } from "kiana-v6"

const agent = new CodingAgent({
  model: createLanguageModel({
    type: "anthropic",
    apiKey: process.env.ANTHROPIC_API_KEY!,
    model: "claude-sonnet-4-20250514",
  }),
  workingDirectory: process.cwd(),
})

// Subscribe to stream events
agent.onStream((part) => {
  if (part.type === "text-delta") {
    process.stdout.write(part.delta)
  }
})

// Generate response
const result = await agent.generate({
  prompt: "List all TypeScript files",
})

console.log(result.text)

// Cleanup
await agent.cleanup()
```

### Streaming

```typescript
const result = await agent.stream({
  prompt: "Your prompt here",
})

// Consume text stream
for await (const chunk of result.textStream) {
  process.stdout.write(chunk)
}

// Wait for completion
const fullText = await result.text
```

### Session Persistence

```typescript
const agent = new CodingAgent({
  model,
  sessionDir: "./sessions/my-session",
})

// Session is automatically saved after each interaction
// Load existing session by using the same sessionDir
```

## Stream Event Types

Kiana emits structured events compatible with AI SDK UI:

- **text-delta** - Streaming text chunks
- **tool-input-available** - Tool being called with arguments
- **tool-output-available** - Tool execution result
- **tool-output-error** - Tool execution error
- **finish** - Generation complete
- **data-session** - Session metadata
- **data-todo** - Task list updates

## CLI Options

```bash
kiana-v6 [options]

Options:
  -c, --config <path>    Path to config file (default: ./kiana.jsonc)
  -p, --prompt <text>    Send a single prompt and exit
  -i, --interactive      Interactive REPL mode
  -s, --session <dir>    Session directory for persistence
  -l, --log <file>       Log all events to file (JSONL)
  -H                     Human-readable output
  -v, --verbose          Show verbose output (tool I/O)
  --create-config        Generate config template
  -h, --help             Show help
```

## Examples

### Code Analysis
```bash
kiana-v6 -H -p "Analyze the TypeScript code in src/ and suggest improvements"
```

### Automated Refactoring
```bash
kiana-v6 -H -p "Refactor all uses of var to const/let in src/"
```

### Documentation Generation
```bash
kiana-v6 -H -p "Generate API documentation for the exported functions"
```

### Git Operations
```bash
kiana-v6 -H -p "Show me uncommitted changes and create a meaningful commit"
```

### Web Research
```bash
kiana-v6 -H -p "Search for best practices for React Server Components and summarize"
```

## Architecture

Kiana is built on several key components:

- **Agent** - `CodingAgent` class manages the agent loop and tool execution
- **Tools** - Extensible tool system with Zod validation
- **Streaming** - Real-time event emission with SSE protocol
- **MCP** - Model Context Protocol integration for external tools
- **Config** - JSONC-based configuration with validation

## Development

```bash
# Install dependencies
pnpm install

# Build
pnpm build

# Watch mode
pnpm dev

# Type check
pnpm typecheck
```

## Contributing

Contributions are welcome! Areas for improvement:

- Additional tool implementations
- More MCP server integrations
- Enhanced error handling
- Performance optimizations
- Documentation improvements

## License

MIT

## Links

- [MCP Documentation](./docs/MCP_SUPPORT.md)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Vercel AI SDK](https://sdk.vercel.ai/)
