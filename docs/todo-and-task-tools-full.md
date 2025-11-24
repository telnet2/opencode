# OpenCode Todo and Task Tools - Comprehensive Documentation

## Overview

OpenCode provides two primary tool systems for task management and delegation:

1. **Todo Tools** (`TodoWrite` and `TodoRead`) - For tracking and managing tasks within a session
2. **Task Tool** - For launching autonomous subagents to handle complex multi-step tasks

---

## Table of Contents

- [Todo Tools](#todo-tools)
  - [Data Model](#data-model)
  - [Tool Definitions](#tool-definitions)
  - [Usage Guidelines](#usage-guidelines)
  - [Storage Design](#storage-design)
- [Task Tool](#task-tool)
  - [Definition & Architecture](#definition--architecture)
  - [Usage Guidelines](#task-usage-guidelines)
- [System Integration](#system-integration)
- [File References](#file-references)

---

## Todo Tools

### Data Model

**File**: `packages/opencode/src/session/todo.ts:6-14`

```typescript
export const Info = z.object({
  content: z.string().describe("Brief description of the task"),
  status: z.string().describe("pending, in_progress, completed, cancelled"),
  priority: z.string().describe("high, medium, low"),
  id: z.string().describe("Unique identifier"),
})
```

### Tool Definitions

#### TodoWriteTool (`packages/opencode/src/tool/todo.ts:6-24`)

- **Parameters**: `todos` array with content, status, priority, id
- **Returns**: Count of incomplete todos, JSON output, metadata
- **Side Effects**: Persists to storage, publishes bus event

#### TodoReadTool (`packages/opencode/src/tool/todo.ts:26-39`)

- **Parameters**: None
- **Returns**: Current todo list
- **Side Effects**: None (read-only)

### Usage Guidelines

**Source**: `packages/opencode/src/tool/todowrite.txt`

#### ✅ When to Use

1. Complex multi-step tasks (3+ steps)
2. Non-trivial tasks requiring planning
3. User explicitly requests it
4. Multiple tasks provided by user
5. After receiving new instructions
6. After completing tasks (mark complete)
7. When starting work (mark in_progress)

#### ❌ When NOT to Use

1. Single straightforward task
2. Trivial task
3. <3 trivial steps
4. Purely conversational

#### Task Management Rules

1. **Status Tracking**: Update real-time, mark complete immediately
2. **Single Focus**: Only ONE task in_progress at a time
3. **Sequential Work**: Complete current before starting new

### Storage Design

**File**: `packages/opencode/src/session/todo.ts:26-35`

```typescript
export async function update(input: { sessionID: string; todos: Info[] }) {
  await Storage.write(["todo", input.sessionID], input.todos)
  Bus.publish(Event.Updated, input)
}

export async function get(sessionID: string) {
  return Storage.read<Info[]>(["todo", sessionID])
    .then((x) => x || [])
    .catch(() => [])
}
```

**Storage Location**: `~/.opencode/storage/todo/{sessionID}.json`

---

## Task Tool

### Definition & Architecture

**File**: `packages/opencode/src/tool/task.ts`

The Task tool spawns autonomous subagents for complex multi-step tasks.

#### Key Implementation Details

1. **Session Creation** (lines 38-42):
   - Creates child session with parentID
   - Title includes subagent name
   - Can resume existing sessions via session_id parameter

2. **Tool Restrictions** (lines 88-92):
   ```typescript
   tools: {
     todowrite: false,  // Prevent recursive nesting
     todoread: false,
     task: false,
     ...agent.tools,
   }
   ```

3. **Progress Tracking** (lines 55-67):
   - Subscribes to MessageV2.Event.PartUpdated
   - Tracks tool calls in subagent
   - Updates metadata with summary

4. **Cancellation Support** (lines 74-78):
   - Respects abort signals
   - Cleans up listeners

#### Parameters

```typescript
{
  description: string      // Short (3-5 words) description
  prompt: string          // Detailed task instructions
  subagent_type: string   // Agent type to use
  session_id?: string     // Optional: resume existing
}
```

#### Returns

```typescript
{
  title: string                    // Task description
  output: string                   // Agent response + metadata
  metadata: {
    summary: ToolPart[]           // All tool calls
    sessionId: string             // Child session ID
  }
}
```

### Task Usage Guidelines

**Source**: `packages/opencode/src/tool/task.txt`

#### When to Use

- Execute custom slash commands
- Complex multi-step autonomous tasks matching agent descriptions

#### When NOT to Use

- Reading specific file paths (use Read/Glob)
- Searching for specific class definitions (use Glob)
- Searching within 2-3 specific files (use Read)
- Tasks not matching agent descriptions

#### Best Practices

1. **Concurrency**: Launch multiple agents in parallel when possible
2. **Detailed Prompts**: Provide highly detailed task descriptions
3. **Specify Intent**: Clearly state if agent should write code or just research
4. **Trust Results**: Agent outputs should generally be trusted
5. **User Communication**: Summarize results for user (agent output not visible to them)

---

## System Integration

### Event Bus Architecture

**File**: `packages/opencode/src/session/todo.ts:16-24`

```typescript
export const Event = {
  Updated: Bus.event("todo.updated", z.object({
    sessionID: z.string(),
    todos: z.array(Info),
  })),
}
```

- Publishes on every TodoWrite
- Enables real-time UI updates
- Session-scoped events

### Tool Registry

**File**: `packages/opencode/src/tool/registry.ts`

Tools are registered centrally and made available to all agents unless explicitly disabled.

### UI Rendering

**File**: `packages/opencode/src/cli/cmd/tui/routes/session/index.tsx:1596-1622`

```tsx
<For each={props.input.todos ?? []}>
  {(todo) => (
    <text style={{
      fg: todo.status === "in_progress" ? theme.success : theme.textMuted
    }}>
      [{todo.status === "completed" ? "✓" : " "}] {todo.content}
    </text>
  )}
</For>
```

Visual indicators:
- ✓ for completed
- Green color for in_progress
- Muted color for pending

---

## File References

### Core Files

| File | Purpose | Lines |
|------|---------|-------|
| `packages/opencode/src/tool/todo.ts` | Tool definitions | 40 |
| `packages/opencode/src/session/todo.ts` | Data model & storage | 37 |
| `packages/opencode/src/tool/task.ts` | Task tool definition | 116 |
| `packages/opencode/src/storage/storage.ts` | File-based storage | 227 |

### Prompt Files

| File | Purpose | Size |
|------|---------|------|
| `packages/opencode/src/tool/todowrite.txt` | TodoWrite usage guidelines | 8,846 bytes |
| `packages/opencode/src/tool/todoread.txt` | TodoRead usage guidelines | 977 bytes |
| `packages/opencode/src/tool/task.txt` | Task tool guidelines | 3,506 bytes |

### System Prompts

| File | Todo Instructions |
|------|-------------------|
| `packages/opencode/src/session/prompt/anthropic.txt` | ✓ Full instructions |
| `packages/opencode/src/session/prompt/anthropic-20250930.txt` | ✓ Enhanced version |
| `packages/opencode/src/session/prompt/polaris.txt` | ✓ Similar instructions |

---

## Data Flow Diagram

```
┌─────────────┐
│ User Input  │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ TodoWrite/Read  │
│ Tool Execution  │
└──────┬──────────┘
       │
       ├──────────────┐
       ▼              ▼
┌──────────────┐  ┌──────────────┐
│ Todo.update()│  │ Todo.get()   │
│ Todo.get()   │  │              │
└──────┬───────┘  └──────┬───────┘
       │                  │
       ▼                  ▼
┌──────────────────────────────┐
│ Storage.write/read()         │
│ ~/.opencode/storage/todo/    │
│   {sessionID}.json           │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────┐
│ Bus.publish()    │
│ Event.Updated    │
└──────┬───────────┘
       │
       ▼
┌──────────────────────┐
│ Tool Returns         │
│ { title, output,     │
│   metadata: todos }  │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ TUI Renders          │
│ with checkmarks ✓    │
│ and color coding     │
└──────────────────────┘
```

---

## Key Design Decisions

### 1. Session-Scoped Storage
- Each session has independent todo list
- Stored at `~/.opencode/storage/todo/{sessionID}.json`
- Enables parallel sessions without conflicts

### 2. Complete List Replacement
- TodoWrite replaces entire list (not incremental updates)
- Simplifies consistency and reduces edge cases
- Agent is responsible for managing complete state

### 3. Task Tool Restrictions
- Subagents cannot use todowrite, todoread, or task tools
- Prevents recursive nesting and complexity
- Forces clear separation of concerns

### 4. Event-Driven UI Updates
- Bus events enable real-time synchronization
- TUI subscribes to Event.Updated
- No polling required

### 5. Single In-Progress Rule
- Only one task should be in_progress at a time
- Enforces sequential completion
- Prevents context-switching confusion

---

## Common Patterns

### Pattern 1: Multi-Step Task
```typescript
// 1. User provides complex request
// 2. Agent creates todo list with TodoWrite
// 3. Agent marks first task in_progress
// 4. Agent completes first task
// 5. Agent marks first completed, second in_progress
// 6. Repeat until all complete
```

### Pattern 2: Task Delegation
```typescript
// 1. Main agent identifies complex subtask
// 2. Launches Task tool with specific subagent
// 3. Subagent works independently (no nested todos/tasks)
// 4. Results returned to main agent
// 5. Main agent continues with results
```

### Pattern 3: Progress Checking
```typescript
// 1. Agent uses TodoRead at conversation start
// 2. Reviews pending/in_progress items
// 3. Continues where left off
// 4. Marks items completed as work progresses
```

---

*Generated from OpenCode source code analysis*
*Last updated: 2025-11-24*
