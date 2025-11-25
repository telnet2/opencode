# OpenCode Hooks System

## Overview

OpenCode provides a powerful hooks system for extending and customizing behavior. The system has **two parallel approaches**:

1. **Configuration-Based Hooks** - Simple shell command execution triggered by events
2. **Plugin-Based Hooks** - Advanced system integration with modification and veto capabilities

### Goals

1. **Extensibility**: Allow users to customize OpenCode behavior without modifying core code
2. **Event-Driven**: React to system events like file edits, session completion, tool execution
3. **Veto Capability**: Enable hooks to approve, modify, or deny operations
4. **Plugin Support**: Provide a rich plugin interface for advanced integrations

### Non-Goals

- Replacing core OpenCode functionality
- Providing hooks for every internal operation
- Supporting synchronous blocking hooks that could freeze the UI

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      OpenCode Core                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐          ┌──────────────────────────────┐ │
│  │ Configuration    │          │ Plugin System                │ │
│  │ Hooks            │          │                              │ │
│  │                  │          │  ┌────────────────────────┐  │ │
│  │ • file_edited    │          │  │ permission.ask         │  │ │
│  │ • session_       │          │  │ tool.execute.before    │  │ │
│  │   completed      │          │  │ tool.execute.after     │  │ │
│  │                  │          │  │ chat.message           │  │ │
│  │ (Shell Commands) │          │  │ chat.params            │  │ │
│  │                  │          │  │ event                  │  │ │
│  └──────────────────┘          │  │ config                 │  │ │
│                                │  └────────────────────────┘  │ │
│                                └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## Configuration-Based Hooks (Experimental)

These hooks are defined in your `opencode.json` or `opencode.jsonc` configuration file and execute shell commands in response to events.

### Location

- **Go SDK**: `/packages/sdk/go/config.go` (lines 705-795)
- **TypeScript Config**: `/packages/opencode/src/config/config.ts`

### Available Hooks

| Hook | Trigger | Use Case |
|------|---------|----------|
| `file_edited` | When files matching a pattern are edited | Run formatters, linters |
| `session_completed` | When a session completes | Auto-commit, cleanup |

### Configuration Structure

```typescript
{
  experimental: {
    hook: {
      file_edited: {
        [pattern: string]: Array<{
          command: string[]
          environment?: Record<string, string>
        }>
      }
      session_completed: Array<{
        command: string[]
        environment?: Record<string, string>
      }>
    }
  }
}
```

### Example Configuration

```json
{
  "experimental": {
    "hook": {
      "file_edited": {
        "*.ts": [
          {
            "command": ["bun", "run", "format", "{file}"],
            "environment": {
              "NODE_ENV": "development"
            }
          }
        ],
        "*.tsx": [
          {
            "command": ["eslint", "--fix", "{file}"]
          }
        ],
        "*.py": [
          {
            "command": ["black", "{file}"]
          },
          {
            "command": ["isort", "{file}"]
          }
        ],
        "*.go": [
          {
            "command": ["gofmt", "-w", "{file}"]
          }
        ]
      },
      "session_completed": [
        {
          "command": ["git", "add", "."],
          "environment": {
            "GIT_AUTHOR_NAME": "OpenCode"
          }
        },
        {
          "command": ["git", "commit", "-m", "Auto-commit from OpenCode session"]
        }
      ]
    }
  }
}
```

### Limitations

- **No Veto Capability**: These hooks execute after the operation completes and cannot cancel it
- **Fire and Forget**: Exit codes are not checked; commands run asynchronously
- **Shell Execution**: Commands run in a shell environment on the host machine

---

## Plugin-Based Hooks

Plugin hooks provide advanced integration capabilities with the ability to modify data and veto operations.

### Location

- **Plugin Interface**: `/packages/plugin/src/index.ts` (lines 138-172)
- **Hook Triggering**: `/packages/opencode/src/plugin/index.ts` (lines 55-90)

### Available Hooks

| Hook | Can Veto? | Description |
|------|-----------|-------------|
| `permission.ask` | **Yes** | Control whether operations are allowed or denied |
| `tool.execute.before` | **Modify** | Modify tool arguments before execution |
| `tool.execute.after` | No | React to tool results after execution |
| `chat.message` | **Modify** | Transform incoming messages and parts |
| `chat.params` | **Modify** | Adjust LLM parameters (temperature, topP, etc.) |
| `event` | No | Subscribe to all system events |
| `config` | No | Initialize plugin with configuration |
| `tool` | No | Register custom tool definitions |
| `auth` | No | Provide authentication handlers |

### Hook Interface

```typescript
export interface Hooks {
  // System hooks
  event?: (input: { event: Event }) => Promise<void>
  config?: (input: Config) => Promise<void>
  tool?: { [key: string]: ToolDefinition }
  auth?: AuthHook

  // Data transformation hooks
  "chat.message"?: (
    input: { sessionID: string; agent?: string; model?; messageID?: string },
    output: { message: UserMessage; parts: Part[] }
  ) => Promise<void>

  "chat.params"?: (
    input: { sessionID: string; agent; model; provider; message },
    output: { temperature; topP; options }
  ) => Promise<void>

  // Permission hooks (CAN VETO)
  "permission.ask"?: (
    input: Permission,
    output: { status: "ask" | "deny" | "allow" }
  ) => Promise<void>

  // Tool execution hooks
  "tool.execute.before"?: (
    input: { tool; sessionID; callID },
    output: { args }
  ) => Promise<void>

  "tool.execute.after"?: (
    input: { tool; sessionID; callID },
    output: { title; output; metadata }
  ) => Promise<void>
}
```

---

## Veto Capability

### Can Hooks Veto Operations?

**Yes**, but only through specific plugin hooks:

#### 1. `permission.ask` Hook (Primary Veto Mechanism)

This hook can explicitly allow or deny operations by setting `output.status`:

```typescript
"permission.ask": async (input, output) => {
  // Deny dangerous bash commands
  if (input.tool === "bash" && input.args?.command?.includes("rm -rf /")) {
    output.status = "deny"
    return
  }

  // Allow safe operations
  if (input.tool === "read") {
    output.status = "allow"
    return
  }

  // Ask user for everything else (default behavior)
  output.status = "ask"
}
```

**Status Values:**
- `"allow"` - Permit the operation without user confirmation
- `"deny"` - **VETO** - Block the operation entirely
- `"ask"` - Prompt the user for confirmation (default)

#### 2. `tool.execute.before` Hook (Modify/Sanitize)

While not a direct veto, this hook can modify arguments to prevent dangerous operations:

```typescript
"tool.execute.before": async (input, output) => {
  // Sanitize file paths
  if (input.tool === "write") {
    output.args.path = sanitizePath(output.args.path)
  }

  // Remove dangerous flags
  if (input.tool === "bash") {
    output.args.command = output.args.command.replace(/--force/g, "")
  }
}
```

#### 3. HTTP Middleware (Request-Level Veto)

For SDK-level control, middleware can intercept and cancel HTTP requests:

**Location**: `/packages/sdk/go/option/middleware.go` (lines 76-92)

```go
myMiddleware := func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
  // Veto certain requests
  if shouldBlock(req) {
    return nil, errors.New("request blocked by middleware")
  }

  return next(req)
}

client := opencode.NewClient(
  option.WithMiddleware(myMiddleware),
)
```

---

## Creating Custom Plugins

### Plugin Structure

**Location**: `/packages/plugin/src/index.ts` (lines 19-27)

```typescript
export type PluginInput = {
  client: ReturnType<typeof createOpencodeClient>
  project: Project
  directory: string
  worktree: string
  $: BunShell
}

export type Plugin = (input: PluginInput) => Promise<Hooks>
```

### Example: Security Plugin

```typescript
import type { Plugin, Hooks } from "@opencode/plugin"

const securityPlugin: Plugin = async (input) => {
  const blockedPatterns = [
    /rm\s+-rf\s+\//,
    />\s*\/dev\/sd/,
    /mkfs\./,
    /dd\s+if=/,
  ]

  return {
    // Initialize with config
    config: async (config) => {
      console.log("Security plugin initialized")
    },

    // Veto dangerous operations
    "permission.ask": async (input, output) => {
      if (input.tool === "bash") {
        const command = input.args?.command || ""

        for (const pattern of blockedPatterns) {
          if (pattern.test(command)) {
            console.warn(`Blocked dangerous command: ${command}`)
            output.status = "deny"
            return
          }
        }
      }
    },

    // Sanitize tool arguments
    "tool.execute.before": async (input, output) => {
      if (input.tool === "write") {
        // Prevent writing to system directories
        const path = output.args?.path || ""
        if (path.startsWith("/etc/") || path.startsWith("/usr/")) {
          throw new Error("Cannot write to system directories")
        }
      }
    },

    // Audit tool executions
    "tool.execute.after": async (input, output) => {
      console.log(`[AUDIT] Tool: ${input.tool}, Call: ${input.callID}`)
    },
  }
}

export default securityPlugin
```

### Example: Logging Plugin

```typescript
const loggingPlugin: Plugin = async (input) => {
  const logFile = `${input.directory}/.opencode/audit.log`

  const log = async (message: string) => {
    const timestamp = new Date().toISOString()
    await input.$`echo "${timestamp}: ${message}" >> ${logFile}`
  }

  return {
    event: async ({ event }) => {
      await log(`Event: ${event.type}`)
    },

    "chat.message": async (input, output) => {
      await log(`Message received in session ${input.sessionID}`)
    },

    "tool.execute.before": async (input, output) => {
      await log(`Tool ${input.tool} starting with args: ${JSON.stringify(output.args)}`)
    },

    "tool.execute.after": async (input, output) => {
      await log(`Tool ${input.tool} completed: ${output.title}`)
    },
  }
}

export default loggingPlugin
```

### Example: Auto-Formatter Plugin

```typescript
const formatterPlugin: Plugin = async (input) => {
  return {
    "tool.execute.after": async (toolInput, output) => {
      // Auto-format after file edits
      if (toolInput.tool === "edit" || toolInput.tool === "write") {
        const filePath = output.metadata?.path as string

        if (filePath?.endsWith(".ts") || filePath?.endsWith(".tsx")) {
          await input.$`bunx prettier --write ${filePath}`
        } else if (filePath?.endsWith(".py")) {
          await input.$`black ${filePath}`
        } else if (filePath?.endsWith(".go")) {
          await input.$`gofmt -w ${filePath}`
        }
      }
    },
  }
}

export default formatterPlugin
```

---

## Hook Execution Flow

### How Hooks Are Triggered

**Location**: `/packages/opencode/src/plugin/index.ts` (lines 55-70)

```typescript
export async function trigger<Name extends keyof Hooks>(
  name: Name,
  input: Input,
  output: Output
): Promise<Output> {
  if (!name) return output

  // Hooks execute sequentially
  for (const hook of await state().then((x) => x.hooks)) {
    const fn = hook[name]
    if (!fn) continue
    await fn(input, output)  // Each hook can modify output
  }

  return output
}
```

### Execution Order

1. Hooks are executed **sequentially** in registration order
2. Each hook receives the **modified output** from the previous hook
3. Hooks can read `input` (immutable) and modify `output` (mutable)
4. If a hook throws an error, subsequent hooks are skipped

### Initialization

**Location**: `/packages/opencode/src/plugin/index.ts` (lines 76-90)

```typescript
export async function init() {
  const hooks = await state().then((x) => x.hooks)
  const config = await Config.get()

  // Initialize each hook with config
  for (const hook of hooks) {
    await hook.config?.(config)
  }

  // Subscribe to all system events
  Bus.subscribeAll(async (input) => {
    for (const hook of hooks) {
      hook["event"]?.({ event: input })
    }
  })
}
```

---

## Summary

| Approach | Veto Capable? | Modify Data? | Use Case |
|----------|---------------|--------------|----------|
| Config `file_edited` | No | No | Run formatters/linters after edits |
| Config `session_completed` | No | No | Auto-commit, cleanup tasks |
| Plugin `permission.ask` | **Yes** | No | Block dangerous operations |
| Plugin `tool.execute.before` | No | **Yes** | Sanitize/transform arguments |
| Plugin `tool.execute.after` | No | **Yes** | Audit, post-processing |
| Plugin `chat.message` | No | **Yes** | Transform messages |
| Plugin `chat.params` | No | **Yes** | Adjust LLM parameters |
| HTTP Middleware | **Yes** | **Yes** | Request-level control |

### Quick Reference

- **To veto operations**: Use `permission.ask` hook with `output.status = "deny"`
- **To modify data**: Use `tool.execute.before`, `chat.message`, or `chat.params` hooks
- **To react to events**: Use `event` or `tool.execute.after` hooks
- **To run shell commands**: Use configuration-based hooks (`file_edited`, `session_completed`)

---

## Related Documentation

- [Client-Side Tools](./design/client-side-tools.md) - Design document for client-side tool registration
- [Testing Infrastructure](./testing-infrastructure.md) - How to test hooks and plugins
