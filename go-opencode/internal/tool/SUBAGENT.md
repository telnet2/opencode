# Subagent System Documentation

This document describes the subagent (task agent) system in go-opencode and compares it with the TypeScript implementation in packages/opencode.

## Table of Contents

1. [Overview](#overview)
2. [Available Agents](#available-agents)
3. [Task Tool Usage](#task-tool-usage)
4. [Parallel Execution](#parallel-execution)
5. [Architecture](#architecture)
6. [TS vs Go Implementation Comparison](#ts-vs-go-implementation-comparison)
7. [Orchestration Patterns](#orchestration-patterns)
8. [Future Roadmap](#future-roadmap)

---

## Overview

The subagent system allows the primary agent to spawn specialized child agents to handle complex, multi-step tasks autonomously. Each subagent runs in its own session context with specific tools and permissions.

### Key Concepts

- **Primary Agent**: The main agent (e.g., `build`, `plan`) that interacts with the user
- **Subagent**: A specialized agent spawned via the `task` tool to handle specific tasks
- **Child Session**: Each subagent execution creates a new session linked to the parent
- **Task Tool**: The tool that enables spawning subagents

### When to Use Subagents

- **Complex research tasks** requiring multiple search iterations
- **Codebase exploration** that needs deep file traversal
- **Parallel workloads** that can be executed concurrently
- **Isolated analysis** where you want a fresh context

---

## Available Agents

### Primary Agents (Cannot be used as subagents)

#### `build`
- **Mode**: Primary
- **Purpose**: Main agent for executing tasks, writing code, and making changes
- **Permissions**: Full access to all tools
- **Use Case**: Default agent for user interactions

#### `plan`
- **Mode**: Primary
- **Purpose**: Planning and analysis without making changes
- **Permissions**:
  - Edit: Denied
  - Bash: Read-only commands only (git, grep, find, ls, etc.)
  - Write: Denied
- **Use Case**: Safe exploration and planning mode

### Subagents (Available via Task tool)

#### `general`
- **Mode**: Subagent
- **Description**: General-purpose agent for researching complex questions and executing multi-step tasks. Use this agent to execute multiple units of work in parallel.
- **Permissions**: Full access (edit, bash, webfetch allowed)
- **Disabled Tools**: `todoread`, `todowrite`, `task` (prevents recursion)
- **Best For**:
  - Complex multi-step research
  - Tasks requiring file modifications
  - Parallel execution of independent work units

#### `explore`
- **Mode**: Subagent
- **Description**: Fast agent specialized for exploring codebases. Use for finding files by patterns, searching code for keywords, or answering questions about the codebase.
- **Custom Prompt**: File search specialist with guidelines for efficient exploration
- **Permissions**: Full access but edit/write disabled
- **Disabled Tools**: `todoread`, `todowrite`, `edit`, `write`, `task`
- **Thoroughness Levels**:
  - `quick`: Basic searches, first-level matches
  - `medium`: Moderate exploration, follow references
  - `very thorough`: Comprehensive analysis across multiple locations
- **Best For**:
  - Finding files by glob patterns (e.g., `src/**/*.tsx`)
  - Searching code for keywords or patterns
  - Understanding codebase structure
  - Answering "where is X defined?" questions

---

## Task Tool Usage

### Basic Invocation

```json
{
  "description": "Find auth handlers",
  "prompt": "Search the codebase for all authentication-related handlers and middleware. Report file paths and a brief description of each.",
  "subagentType": "explore"
}
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `description` | string | Yes | Short 3-5 word description of the task |
| `prompt` | string | Yes | Detailed task instructions for the subagent |
| `subagentType` | string | Yes | Agent to use: `general`, `explore`, or `plan` |
| `model` | string | No | Optional model override: `sonnet`, `opus`, `haiku` |
| `resume` | string | No | Optional session ID to resume from (TS only) |

### Response Format

```json
{
  "title": "Completed: Find auth handlers",
  "output": "Found 5 authentication handlers:\n1. /src/auth/login.go:45 - LoginHandler\n...",
  "metadata": {
    "subagent": "explore",
    "status": "completed",
    "sessionID": "01HXYZ..."
  }
}
```

---

## Parallel Execution

### How It Works

The LLM can invoke multiple task tools in a single response. Each task runs independently in its own session.

```
Primary Agent Response:
├── Task 1: [explore] "Find all API endpoints"
├── Task 2: [explore] "Find all database models"
└── Task 3: [general] "Analyze error handling patterns"
```

### Concurrency Model

#### TypeScript Implementation
- Uses `Promise.all()` for parallel execution
- Each task gets independent context with unique `callID`
- Progress updates via event bus (real-time)
- Supports up to 10 parallel tasks via Batch tool

#### Go Implementation
- Each task executed via goroutines (when triggered by LLM)
- Independent sessions with separate processor instances
- Results collected after all tasks complete
- No explicit concurrency limit (bounded by LLM tool calls)

### Example: Parallel Codebase Analysis

```
User: "Analyze this codebase architecture"

Primary Agent spawns:
1. Task(explore): "Find all entry points (main.go, cmd/)"
2. Task(explore): "Find configuration handling"
3. Task(explore): "Find database/storage layer"
4. Task(explore): "Find API/HTTP handlers"
5. Task(general): "Analyze dependency injection patterns"

Results aggregated by primary agent into comprehensive response.
```

### Best Practices for Parallel Execution

1. **Independent Tasks**: Ensure tasks don't depend on each other's results
2. **Specific Prompts**: Give each subagent focused, unambiguous instructions
3. **Appropriate Agent**: Use `explore` for read-only, `general` for modifications
4. **Result Aggregation**: Primary agent should synthesize subagent outputs

---

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Primary Session                          │
│  ┌─────────────┐    ┌──────────────┐    ┌───────────────┐  │
│  │   User      │───▶│   Processor  │───▶│  Tool Registry │  │
│  │   Message   │    │   (Loop)     │    │               │  │
│  └─────────────┘    └──────────────┘    └───────┬───────┘  │
│                            │                     │          │
│                            ▼                     ▼          │
│                     ┌──────────────┐    ┌───────────────┐  │
│                     │   LLM Call   │    │   Task Tool   │  │
│                     └──────────────┘    └───────┬───────┘  │
└─────────────────────────────────────────────────┼──────────┘
                                                  │
                    ┌─────────────────────────────┼─────────────────────────────┐
                    │                             ▼                             │
                    │              ┌─────────────────────────────┐              │
                    │              │     SubagentExecutor        │              │
                    │              │  - Creates child session    │              │
                    │              │  - Converts agent config    │              │
                    │              │  - Runs processor loop      │              │
                    │              └─────────────┬───────────────┘              │
                    │                            │                              │
          ┌─────────┴────────┐         ┌────────┴────────┐         ┌──────────┴─────────┐
          ▼                  ▼         ▼                 ▼         ▼                    ▼
   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
   │ Child Session│   │ Child Session│   │ Child Session│   │ Child Session│
   │  (explore)   │   │  (explore)   │   │  (general)   │   │    ...       │
   │              │   │              │   │              │   │              │
   │ ┌──────────┐ │   │ ┌──────────┐ │   │ ┌──────────┐ │   │              │
   │ │Processor │ │   │ │Processor │ │   │ │Processor │ │   │              │
   │ │  Loop    │ │   │ │  Loop    │ │   │ │  Loop    │ │   │              │
   │ └──────────┘ │   │ └──────────┘ │   │ └──────────┘ │   │              │
   └──────────────┘   └──────────────┘   └──────────────┘   └──────────────┘
```

### Session Hierarchy

```
Session: "Main conversation" (ID: 01HX...)
├── Message: User "Analyze the codebase"
├── Message: Assistant (spawns tasks)
│   ├── ToolPart: task (explore) → Child Session 01HY...
│   ├── ToolPart: task (explore) → Child Session 01HZ...
│   └── ToolPart: task (general) → Child Session 01HA...
└── Message: Assistant (aggregated response)

Child Session: 01HY... (ParentID: 01HX...)
├── Message: User (subagent prompt)
└── Message: Assistant (subagent response)
```

### Key Files

| File | Purpose |
|------|---------|
| `internal/agent/agent.go` | Agent definitions and permissions |
| `internal/agent/registry.go` | Agent registration and lookup |
| `internal/tool/task.go` | Task tool implementation |
| `internal/tool/subagent_executor.go` | Subagent execution logic |
| `internal/tool/registry.go` | Tool registry with task tool support |
| `internal/session/processor.go` | Agentic loop processor |

---

## TS vs Go Implementation Comparison

### Feature Matrix

| Feature | TypeScript | Go | Notes |
|---------|------------|-----|-------|
| Basic subagent execution | ✅ | ✅ | Both create child sessions |
| Agent registry | ✅ | ✅ | Identical agent definitions |
| Custom agent prompts | ✅ | ✅ | Explore agent has custom prompt |
| Permission system | ✅ | ✅ | Bash patterns, edit control |
| Session hierarchy | ✅ | ✅ | ParentID linking |
| Session resume | ✅ | ❌ | TS supports `session_id` param |
| Real-time progress | ✅ | ❌ | TS uses event bus subscription |
| Metadata callback | ✅ | ❌ | TS updates parent with progress |
| Workflow orchestration | ✅ | ❌ | TS has full workflow DSL |
| Parallel steps | ✅ | ❌ | TS has explicit parallel execution |
| Conditional branching | ✅ | ❌ | TS supports if/then/else workflows |
| Loop steps | ✅ | ❌ | TS supports while/until loops |
| Pause/Resume | ✅ | ❌ | TS has human-in-the-loop |
| Batch tool | ✅ | ❌ | TS can batch 10 parallel tools |

### Implementation Differences

#### Session Creation

**TypeScript:**
```typescript
const session = await Session.create({
  parentID: ctx.sessionID,
  title: `${params.description} (@${agent.name} subagent)`,
})
```

**Go:**
```go
session := &types.Session{
    ID:        ulid.Make().String(),
    ParentID:  &parentSessionID,
    Title:     fmt.Sprintf("Subtask: %s", agentName),
    // ...
}
```

#### Progress Tracking

**TypeScript:**
```typescript
const unsub = Bus.subscribe(MessageV2.Event.PartUpdated, async (evt) => {
    if (evt.properties.part.sessionID !== session.id) return
    parts[evt.properties.part.id] = evt.properties.part
    ctx.metadata({
        summary: Object.values(parts).sort(...)
    })
})
```

**Go:**
```go
// Currently no real-time progress tracking
// Results returned only after completion
```

#### Agent Config Conversion

**TypeScript:**
```typescript
// Direct use of agent config
const tools = {
    todowrite: false,
    todoread: false,
    task: false,
    ...agent.tools,
}
```

**Go:**
```go
func convertToSessionAgent(a *agent.Agent) *session.Agent {
    // Converts agent.Agent to session.Agent
    // Maps tools, permissions, and settings
}
```

---

## Orchestration Patterns

### Pattern 1: Fan-Out / Fan-In

Use multiple subagents for parallel exploration, then aggregate results.

```
Primary Agent:
1. Spawn N explore agents for different aspects
2. Wait for all to complete
3. Synthesize findings into coherent response

Example:
- Task 1: Find all REST endpoints
- Task 2: Find all GraphQL resolvers
- Task 3: Find all WebSocket handlers
→ Aggregate into "API Surface Analysis"
```

### Pattern 2: Specialist Delegation

Route specific subtasks to the most appropriate agent.

```
User: "Review and improve error handling"

Primary Agent:
1. Task(explore): "Find all error handling patterns"
2. Based on findings...
3. Task(general): "Refactor error handling in auth module"
4. Task(general): "Add error handling to database layer"
```

### Pattern 3: Iterative Refinement

Use subagents for progressive deepening.

```
Round 1: Task(explore, quick): "Find main components"
Round 2: Task(explore, thorough): "Deep dive into auth component"
Round 3: Task(general): "Implement improvements to auth"
```

### Pattern 4: Verification Pipeline

Use subagents to verify work.

```
1. Task(general): "Implement feature X"
2. Task(explore): "Verify feature X implementation"
3. Task(general): "Fix issues found in verification"
```

### TypeScript-Only: Workflow Orchestration

The TS implementation supports declarative workflows:

```typescript
const workflow = {
  steps: [
    { id: "analyze", type: "agent", agent: "explore", input: "..." },
    { id: "implement", type: "agent", agent: "general", dependsOn: ["analyze"] },
    { id: "verify", type: "parallel", steps: ["test", "lint"], maxConcurrency: 2 },
    { id: "review", type: "pause", message: "Please review changes" },
  ],
  orchestrator: {
    mode: "guided",
    onError: "pause",
    maxRetries: 3,
  }
}
```

---

## Future Roadmap

### Planned Go Implementation

#### Phase 1: Core Improvements
- [ ] Add session resume via `session_id` parameter
- [ ] Implement real-time progress events
- [ ] Add metadata callback to tool context
- [ ] Implement tool restriction in subagents

#### Phase 2: Orchestration
- [ ] Define workflow types and schema
- [ ] Implement workflow executor
- [ ] Add parallel step execution
- [ ] Add conditional branching

#### Phase 3: Advanced Features
- [ ] Implement pause/resume mechanism
- [ ] Add loop steps (while/until)
- [ ] Implement transform steps
- [ ] Add batch tool for parallel tool execution

### Contributing

To contribute to the subagent system:

1. Agent definitions: `internal/agent/agent.go`
2. Task tool: `internal/tool/task.go`
3. Executor: `internal/tool/subagent_executor.go`
4. Tests: Add tests in `*_test.go` files

---

## Appendix: Agent Permission Reference

### Bash Command Patterns (Plan Agent)

| Pattern | Permission | Purpose |
|---------|------------|---------|
| `git diff*`, `git log*`, `git status*` | Allow | Git read operations |
| `grep*`, `rg*` | Allow | Search tools |
| `find *` | Allow | File finding |
| `find * -delete*`, `find * -exec*` | Ask | Dangerous find flags |
| `ls*`, `cat*`, `head*`, `tail*` | Allow | File reading |
| `tree*` | Allow | Directory visualization |
| `*` | Ask | All other commands |

### Tool Availability by Agent

| Tool | build | plan | general | explore |
|------|-------|------|---------|---------|
| read | ✅ | ✅ | ✅ | ✅ |
| write | ✅ | ❌ | ✅ | ❌ |
| edit | ✅ | ❌ | ✅ | ❌ |
| bash | ✅ | ✅* | ✅ | ✅ |
| glob | ✅ | ✅ | ✅ | ✅ |
| grep | ✅ | ✅ | ✅ | ✅ |
| task | ✅ | ✅ | ❌ | ❌ |
| todowrite | ✅ | ❌ | ❌ | ❌ |
| todoread | ✅ | ❌ | ❌ | ❌ |
| webfetch | ✅ | ✅ | ✅ | ✅ |

*Plan agent has restricted bash (read-only commands only)
