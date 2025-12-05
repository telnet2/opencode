# Session Package

The session package handles LLM conversation sessions, message processing, and streaming responses.

## Key Components

| File | Description |
|------|-------------|
| `service.go` | Session service interface and implementation |
| `processor.go` | Message processing and LLM interaction |
| `stream.go` | Stream processing for LLM responses |
| `loop.go` | Main processing loop with tool execution |
| `tools.go` | Tool execution and result handling |
| `compact.go` | Context compaction/summarization |
| `title.go` | Session title generation from first message |

## Token Usage Tracking

### Overview

The session package tracks token usage for each LLM response to display context window usage in the TUI sidebar. The TUI calculates context as:

```
total = input + output + reasoning + cache.read + cache.write
```

### Eino Framework Streaming Behavior

The Eino claude framework (`cloudwego/eino-ext/components/model/claude`) sends token usage across multiple stream events:

| Event | Token Data |
|-------|-----------|
| `MessageStartEvent` (first) | `PromptTokens` (input tokens + cache info) |
| `MessageDeltaEvent` (last) | `CompletionTokens` only |

**Important**: `MessageStartEvent` contains:
```go
promptTokens := int(resp.Usage.InputTokens + resp.Usage.CacheReadInputTokens + resp.Usage.CacheCreationInputTokens)
```

While `MessageDeltaEvent` only contains:
```go
Usage: &schema.TokenUsage{
    CompletionTokens: int(e.Usage.OutputTokens),
}
```

### Implementation in stream.go

Because token usage is split across multiple stream events, we **merge** the values by taking the maximum of each field:

```go
// Track token usage across stream events
var inputTokens, completionTokens, cachedTokens int
var hasUsage bool

// In stream loop:
if msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
    usage := msg.ResponseMeta.Usage
    hasUsage = true
    // Take the max of each field (first event has input, last has output)
    if usage.PromptTokens > inputTokens {
        inputTokens = usage.PromptTokens
    }
    if usage.CompletionTokens > completionTokens {
        completionTokens = usage.CompletionTokens
    }
    if usage.PromptTokenDetails.CachedTokens > cachedTokens {
        cachedTokens = usage.PromptTokenDetails.CachedTokens
    }
}
```

This ensures:
- `inputTokens` comes from `MessageStartEvent`
- `completionTokens` comes from `MessageDeltaEvent`
- `cachedTokens` comes from `MessageStartEvent`

### Common Pitfalls

**DO NOT** simply keep the "latest" usage value:
```go
// WRONG: This loses input tokens from the first event
if msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
    finalUsage = msg.ResponseMeta.Usage  // Overwrites good data!
}
```

This will result in `input=0` and `cache.read=0` because the last event (`MessageDeltaEvent`) doesn't include those fields.

## Event Publishing

The session package publishes events via the `event.PublishSync` function. Events are published for:

- Message creation (`message.created`)
- Message updates (`message.updated`)
- Part updates (`message.part.updated`)
- Session status changes (`session.status`)
- Session compaction (`session.compacted`)

### SSE Subscriber Requirements

When using `PublishSync`, subscribers are called synchronously. SSE subscribers must use non-blocking channel sends to avoid deadlocks:

```go
select {
case events <- e:
default:
    logging.Warn().
        Str("eventType", string(e.Type)).
        Msg("SSE event dropped: channel full")
}
```

## Finish Reason Normalization

Different providers return different finish reasons. The session package normalizes them to SDK-compatible format:

| Provider Value | Normalized Value |
|---------------|------------------|
| `tool_use` | `tool-calls` |
| `stop` | `stop` |
| (empty with tool calls) | `tool-calls` |
| (empty without tool calls) | `stop` |

## Step Parts

Each LLM inference step is bracketed with parts:

1. **step-start**: Emitted at the beginning of inference
2. **text/tool/reasoning parts**: Streamed content
3. **step-finish**: Emitted at the end with cost and token info

The `step-finish` part includes the final token usage and finish reason, which the TUI uses for context display.

## Session Title Generation

### Overview

Session titles are automatically generated on the first user message to provide meaningful names in the TUI sidebar (instead of "New Session").

### Implementation in title.go

The `ensureTitle` function:
1. Only runs on first user message (step == 0)
2. Checks if session has no parent AND title is still default ("New Session")
3. Uses the default model to generate a brief title (≤50 chars)
4. Runs asynchronously (goroutine) to not block the response
5. Publishes `session.updated` event so TUI updates immediately

### Title Generation Prompt

The system prompt instructs the LLM to:
- Output ONLY a thread title (single line, ≤50 characters)
- Use -ing verbs for actions (Debugging, Implementing, Analyzing)
- Keep technical terms, numbers, filenames exact
- Remove articles (the, this, my, a, an)

### When Title is Generated

In `loop.go`, after the first step (step == 0) completes successfully:

```go
if step == 0 && userContent != "" {
    go p.ensureTitle(context.Background(), &session, userContent)
}
```

The title generation uses `context.Background()` because:
- It runs in a goroutine that may outlive the HTTP request
- Title generation is non-critical and should not be cancelled

### Default Title Detection

A title is considered "default" if it equals "New Session" or starts with "New Session":

```go
func isDefaultTitle(title string) bool {
    return title == defaultTitlePrefix || strings.HasPrefix(title, defaultTitlePrefix)
}
```
