# Kiana - Minimal Headless Coding Agent

A standalone TypeScript package providing a headless coding agent library + CLI wrapper with zero dependencies on `packages/opencode`.

## Installation

```bash
npm install
npm run build
```

## Usage

### CLI

```bash
# Create config template
kiana --create-config ./kiana.jsonc

# Run with prompt (exits after completion)
kiana --config ./kiana.jsonc -p "list files in src directory"

# Run with session persistence
kiana --config ./kiana.jsonc --session ./my-session -p "analyze this codebase"

# Human-readable output (streaming text)
kiana --config ./kiana.jsonc -H -p "explain main.ts"

# Interactive mode (stdin/stdout JSON protocol)
kiana --config ./kiana.jsonc
```

### Library

```typescript
import { createSession, loadConfig } from 'kiana';

const config = loadConfig('./kiana.jsonc');
const session = await createSession(config);

session.onEvent((event) => {
  console.log(event.type, event.properties);
});

await session.sendMessage('List files in src directory');
```

## Configuration

```jsonc
{
  "provider": {
    "type": "openai-compatible",  // "anthropic" | "openai" | "openai-compatible" | "google"
    "apiKey": "YOUR_API_KEY",
    "model": "your-model-id",
    "baseUrl": "https://api.example.com/v1"  // Required for openai-compatible
  },
  "streaming": true,    // Set to false for better token counting with some providers
  "maxRetries": 5       // Retry count for rate limit errors (exponential backoff)
}
```

## Implementation Notes

### Token Usage in Streaming Mode

For OpenAI-compatible providers, streaming mode requires `stream_options: { include_usage: true }` to get token counts. This is enabled by setting `includeUsage: true` when creating the provider:

```typescript
const compatible = createOpenAICompatible({
  name: "openai-compatible",
  apiKey: config.apiKey,
  baseURL: config.baseUrl,
  includeUsage: true,  // Enables token counting in streaming mode
});
```

After streaming completes, use `await stream.totalUsage` to get accumulated token counts:

```typescript
const stream = streamText({ model, messages, tools });
for await (const chunk of stream.fullStream) {
  // Process chunks...
}
const totalUsage = await stream.totalUsage;  // { inputTokens, outputTokens, ... }
```

### Session Persistence & Resumption

When building AI SDK messages from persisted session data, tool calls and results must follow AI SDK v6 format:

**Tool Call (in assistant message):**
```typescript
{
  type: "tool-call",
  toolCallId: "call-123",
  toolName: "bash",
  input: JSON.stringify({ command: "ls" }),  // Stringified JSON, not object!
}
```

**Tool Result (in tool message):**
```typescript
{
  type: "tool-result",
  toolCallId: "call-123",
  toolName: "bash",
  output: {
    type: "text",      // or "error-text" for errors
    value: "file1.ts\nfile2.ts",
  },
}
```

### Tool Call Argument Formatting (LLM Quirk)

Some OpenAI-compatible LLMs (e.g., ByteDance/Volcano models) occasionally output tool call arguments as double-stringified JSON, causing validation errors:

```
Invalid input: expected object, received string
"{\"command\":\"git status\"}"
```

**Why this happens:** This is a model-level behavior issue, not a framework bug. Claude models and most OpenAI models correctly output tool arguments as objects, but some models occasionally stringify them twice.

**Why langchain/langgraph doesn't have this issue:** Projects using langchain (like kiana-agent) typically use Claude models which follow the tool format correctly. The issue is model-specific, not framework-specific.

**Our mitigation strategy (defense in depth):**

1. **System prompt guidance** - Explicit instructions in the system prompt:
   ```
   CRITICAL: When calling tools, pass arguments as a proper JSON object (not a string).
   Example good: {"command": "git status", "description": "Check git status"}
   Example bad: "{\"command\": \"git status\"}"
   ```

2. **Defensive JSON.parse in tool execution** - The `defineTool` wrapper in `src/tool/tool.ts` automatically handles double-stringified args:
   ```typescript
   // Normalize args: some models erroneously stringify tool inputs
   let normalizedArgs: unknown = args
   if (typeof args === "string") {
     try {
       normalizedArgs = JSON.parse(args)
     } catch {
       // Fall through to validation error
       normalizedArgs = args
     }
   }
   ```

This two-layer approach ensures robustness: the system prompt prevents most issues, and the defensive parsing catches any that slip through

### Retry Logic

The AI SDK provides built-in retry with exponential backoff for rate limit errors. Configure via `maxRetries` (default: 5):

```typescript
streamText({
  model,
  messages,
  tools,
  maxRetries: config.maxRetries,  // Respects Retry-After headers
});
```

## Architecture

```
src/
├── index.ts           # Library exports
├── session.ts         # Session management & agent loop
├── config.ts          # Config loading (JSONC)
├── event.ts           # Event bus (typed events)
├── provider.ts        # LLM provider setup (AI SDK 6)
├── cli.ts             # CLI wrapper
├── tool/              # Tool implementations
│   ├── bash.ts
│   ├── read.ts
│   ├── write.ts
│   ├── edit.ts
│   ├── glob.ts
│   ├── grep.ts
│   ├── list.ts
│   ├── webfetch.ts
│   ├── websearch.ts
│   ├── codesearch.ts
│   ├── todo.ts
│   └── task.ts
└── types/             # Type definitions
    ├── session.ts
    ├── message.ts
    ├── part.ts
    └── event.ts
```

## Event Types

Wire format: `{"type": "...", "properties": {...}}`

- `session.created` / `session.updated` / `session.idle`
- `message.created` / `message.updated`
- `message.part.updated` (text, tool, step-start, step-finish)
- `todo.updated`

## License

MIT
