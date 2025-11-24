# System Prompt Construction Analysis

This document provides a comprehensive analysis of the system prompt sent to LLMs in OpenCode, including the template processing and final prompt structure.

## Table of Contents

1. [Overview](#1-overview)
2. [Prompt Template Files](#2-prompt-template-files)
3. [Construction Process](#3-construction-process)
4. [Complete Anthropic Prompt](#4-complete-anthropic-prompt)
5. [Environment Context Template](#5-environment-context-template)
6. [Custom Instructions Loading](#6-custom-instructions-loading)
7. [Final Message Structure](#7-final-message-structure)
8. [Complete Example](#8-complete-example)
9. [Model-Specific Variations](#9-model-specific-variations)

---

## 1. Overview

The system prompt is constructed from multiple components assembled in a specific order:

1. **Provider Header** (Anthropic only)
2. **Base Prompt** (model-specific)
3. **Environment Context** (dynamic)
4. **Custom Instructions** (user-defined)

The final prompt is optimized into **2 system messages** for caching efficiency.

---

## 2. Prompt Template Files

**Location**: `packages/opencode/src/session/prompt/`

### Main Prompts

| File | Model | Lines | Purpose |
|------|-------|-------|---------|
| `anthropic.txt` | Claude models | 106 | Main coding assistant prompt |
| `beast.txt` | GPT-4/o1/o3 | lengthy | Autonomous problem-solving |
| `gemini.txt` | Gemini | 156 | Gemini-specific instructions |
| `qwen.txt` | Other models | minimal | Concise responses |
| `polaris.txt` | Polaris-alpha | - | Polaris-specific |
| `codex.txt` | GPT-5 | 319 | Detailed workflows |

### Utility Prompts

| File | Purpose |
|------|---------|
| `anthropic_spoof.txt` | Anthropic provider header |
| `summarize.txt` | Conversation summaries |
| `compaction.txt` | Context compression |
| `title.txt` | Thread title generation |
| `plan.txt` | Read-only phase constraint |
| `build-switch.txt` | Plan/build agent switching |

---

## 3. Construction Process

### Entry Point

**File**: `packages/opencode/src/session/prompt.ts` (lines 621-641)

```typescript
async function resolveSystemPrompt(input: {
  system?: string
  agent: Agent.Info
  providerID: string
  modelID: string
}) {
  let system = SystemPrompt.header(input.providerID)           // Step 1
  system.push(
    ...(() => {
      if (input.system) return [input.system]                  // Step 2a
      if (input.agent.prompt) return [input.agent.prompt]      // Step 2b
      return SystemPrompt.provider(input.modelID)              // Step 2c
    })(),
  )
  system.push(...(await SystemPrompt.environment()))           // Step 3
  system.push(...(await SystemPrompt.custom()))                // Step 4

  // Optimization: Combine into 2 messages for caching
  const [first, ...rest] = system
  system = [first, rest.join("\n")]
  return system
}
```

### Step-by-Step Assembly

**Step 1: Provider Header**

**File**: `packages/opencode/src/session/system.ts` (lines 22-25)

```typescript
export function header(providerID: string) {
  if (providerID.includes("anthropic")) return [PROMPT_ANTHROPIC_SPOOF.trim()]
  return []
}
```

Only Anthropic provider gets: `"You are Claude Code, Anthropic's official CLI for Claude."`

**Step 2: Base Prompt Selection**

**File**: `packages/opencode/src/session/system.ts` (lines 27-34)

```typescript
export function provider(modelID: string) {
  if (modelID.includes("gpt-5")) return [PROMPT_CODEX]
  if (modelID.includes("gpt-") || modelID.includes("o1") || modelID.includes("o3"))
    return [PROMPT_BEAST]
  if (modelID.includes("gemini-")) return [PROMPT_GEMINI]
  if (modelID.includes("claude")) return [PROMPT_ANTHROPIC]
  if (modelID.includes("polaris-alpha")) return [PROMPT_POLARIS]
  return [PROMPT_ANTHROPIC_WITHOUT_TODO]  // Default (qwen.txt)
}
```

Priority order:
1. Custom system override (`input.system`)
2. Agent-specific prompt (`input.agent.prompt`)
3. Model-specific default

**Step 3: Environment Context** (see [Section 5](#5-environment-context-template))

**Step 4: Custom Instructions** (see [Section 6](#6-custom-instructions-loading))

---

## 4. Complete Anthropic Prompt

**File**: `packages/opencode/src/session/prompt/anthropic.txt`

This is the main prompt for Claude models (106 lines):

```
You are OpenCode, the best coding agent on the planet.

You are an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

IMPORTANT: You must NEVER generate or guess URLs for the user unless you are confident that the URLs are for helping the user with programming. You may use URLs provided by the user in their messages or local files.

If the user asks for help or wants to give feedback inform them of the following:
- ctrl+p to list available actions
- To give feedback, users should report the issue at
  https://github.com/sst/opencode

When the user directly asks about OpenCode (eg. "can OpenCode do...", "does OpenCode have..."), or asks in second person (eg. "are you able...", "can you do..."), or asks how to use a specific OpenCode feature (eg. implement a hook, write a slash command, or install an MCP server), use the WebFetch tool to gather information to answer the question from OpenCode docs. The list of available docs is available at https://opencode.ai/docs

# Tone and style
- Only use emojis if the user explicitly requests it. Avoid using emojis in all communication unless asked.
- Your output will be displayed on a command line interface. Your responses should be short and concise. You can use Github-flavored markdown for formatting, and will be rendered in a monospace font using the CommonMark specification.
- Output text to communicate with the user; all text you output outside of tool use is displayed to the user. Only use tools to complete tasks. Never use tools like Bash or code comments as means to communicate with the user during the session.
- NEVER create files unless they're absolutely necessary for achieving your goal. ALWAYS prefer editing an existing file to creating a new one. This includes markdown files.

# Professional objectivity
Prioritize technical accuracy and truthfulness over validating the user's beliefs. Focus on facts and problem-solving, providing direct, objective technical info without any unnecessary superlatives, praise, or emotional validation. It is best for the user if OpenCode honestly applies the same rigorous standards to all ideas and disagrees when necessary, even if it may not be what the user wants to hear. Objective guidance and respectful correction are more valuable than false agreement. Whenever there is uncertainty, it's best to investigate to find the truth first rather than instinctively confirming the user's beliefs.

# Task Management
You have access to the TodoWrite tools to help you manage and plan tasks. Use these tools VERY frequently to ensure that you are tracking your tasks and giving the user visibility into your progress.
These tools are also EXTREMELY helpful for planning tasks, and for breaking down larger complex tasks into smaller steps. If you do not use this tool when planning, you may forget to do important tasks - and that is unacceptable.

It is critical that you mark todos as completed as soon as you are done with a task. Do not batch up multiple tasks before marking them as completed.

Examples:

<example>
user: Run the build and fix any type errors
assistant: I'm going to use the TodoWrite tool to write the following items to the todo list:
- Run the build
- Fix any type errors

I'm now going to run the build using Bash.

Looks like I found 10 type errors. I'm going to use the TodoWrite tool to write 10 items to the todo list.

marking the first todo as in_progress

Let me start working on the first item...

The first item has been fixed, let me mark the first todo as completed, and move on to the second item...
..
..
</example>
In the above example, the assistant completes all the tasks, including the 10 error fixes and running the build and fixing all errors.

<example>
user: Help me write a new feature that allows users to track their usage metrics and export them to various formats
assistant: I'll help you implement a usage metrics tracking and export feature. Let me first use the TodoWrite tool to plan this task.
Adding the following todos to the todo list:
1. Research existing metrics tracking in the codebase
2. Design the metrics collection system
3. Implement core metrics tracking functionality
4. Create export functionality for different formats

Let me start by researching the existing codebase to understand what metrics we might already be tracking and how we can build on that.

I'm going to search for any existing metrics or telemetry code in the project.

I've found some existing telemetry code. Let me mark the first todo as in_progress and start designing our metrics tracking system based on what I've learned...

[Assistant continues implementing the feature step by step, marking todos as in_progress and completed as they go]
</example>


# Doing tasks
The user will primarily request you perform software engineering tasks. This includes solving bugs, adding new functionality, refactoring code, explaining code, and more. For these tasks the following steps are recommended:
-
- Use the TodoWrite tool to plan the task if required

- Tool results and user messages may include <system-reminder> tags. <system-reminder> tags contain useful information and reminders. They are automatically added by the system, and bear no direct relation to the specific tool results or user messages in which they appear.


# Tool usage policy
- When doing file search, prefer to use the Task tool in order to reduce context usage.
- You should proactively use the Task tool with specialized agents when the task at hand matches the agent's description.

- When WebFetch returns a message about a redirect to a different host, you should immediately make a new WebFetch request with the redirect URL provided in the response.
- You can call multiple tools in a single response. If you intend to call multiple tools and there are no dependencies between them, make all independent tool calls in parallel. Maximize use of parallel tool calls where possible to increase efficiency. However, if some tool calls depend on previous calls to inform dependent values, do NOT call these tools in parallel and instead call them sequentially. For instance, if one operation must complete before another starts, run these operations sequentially instead. Never use placeholders or guess missing parameters in tool calls.
- If the user specifies that they want you to run tools "in parallel", you MUST send a single message with multiple tool use content blocks. For example, if you need to launch multiple agents in parallel, send a single message with multiple Task tool calls.
- Use specialized tools instead of bash commands when possible, as this provides a better user experience. For file operations, use dedicated tools: Read for reading files instead of cat/head/tail, Edit for editing instead of sed/awk, and Write for creating files instead of cat with heredoc or echo redirection. Reserve bash tools exclusively for actual system commands and terminal operations that require shell execution. NEVER use bash echo or other command-line tools to communicate thoughts, explanations, or instructions to the user. Output all communication directly in your response text instead.
- VERY IMPORTANT: When exploring the codebase to gather context or to answer a question that is not a needle query for a specific file/class/function, it is CRITICAL that you use the Task tool instead of running search commands directly.
<example>
user: Where are errors from the client handled?
assistant: [Uses the Task tool to find the files that handle client errors instead of using Glob or Grep directly]
</example>
<example>
user: What is the codebase structure?
assistant: [Uses the Task tool]
</example>

IMPORTANT: Always use the TodoWrite tool to plan and track tasks throughout the conversation.

# Code References

When referencing specific functions or pieces of code include the pattern `file_path:line_number` to allow the user to easily navigate to the source code location.

<example>
user: Where are errors from the client handled?
assistant: Clients are marked as failed in the `connectToServer` function in src/services/process.ts:712.
</example>
```

---

## 5. Environment Context Template

**File**: `packages/opencode/src/session/system.ts` (lines 36-59)

```typescript
export async function environment() {
  const project = Instance.project
  return [
    [
      `Here is some useful information about the environment you are running in:`,
      `<env>`,
      `  Working directory: ${Instance.directory}`,
      `  Is directory a git repo: ${project.vcs === "git" ? "yes" : "no"}`,
      `  Platform: ${process.platform}`,
      `  Today's date: ${new Date().toDateString()}`,
      `</env>`,
      `<files>`,
      `  ${
        project.vcs === "git"
          ? await Ripgrep.tree({
              cwd: Instance.directory,
              limit: 200,
            })
          : ""
      }`,
      `</files>`,
    ].join("\n"),
  ]
}
```

### Variables Substituted

| Variable | Source | Example |
|----------|--------|---------|
| `${Instance.directory}` | Current working directory | `/home/user/myproject` |
| `${project.vcs === "git" ? "yes" : "no"}` | Git status | `yes` |
| `${process.platform}` | OS platform | `linux`, `darwin`, `win32` |
| `${new Date().toDateString()}` | Current date | `Sun Nov 24 2024` |
| File tree | Ripgrep.tree (limit 200) | Indented file listing |

### Example Output

```
Here is some useful information about the environment you are running in:
<env>
  Working directory: /home/user/myproject
  Is directory a git repo: yes
  Platform: linux
  Today's date: Sun Nov 24 2024
</env>
<files>
  myproject/
    .git/
    src/
      main.go
      handlers/
        api.go
      models/
        user.go
    go.mod
    go.sum
    README.md
</files>
```

---

## 6. Custom Instructions Loading

**File**: `packages/opencode/src/session/system.ts` (lines 61-118)

### Search Paths

**Local files** (project-specific, searched in order):
1. `AGENTS.md`
2. `CLAUDE.md`
3. `CONTEXT.md` (deprecated)

**Global files** (user-level, searched in order):
1. `~/.opencode/AGENTS.md` (Global.Path.config)
2. `~/.claude/CLAUDE.md`

### Loading Logic

```typescript
export async function custom() {
  const config = await Config.get()
  const paths = new Set<string>()

  // Search for local rule files (first match wins per category)
  for (const localRuleFile of LOCAL_RULE_FILES) {
    const matches = await Filesystem.findUp(localRuleFile, Instance.directory, Instance.worktree)
    if (matches.length > 0) {
      matches.forEach((path) => paths.add(path))
      break
    }
  }

  // Search for global rule files
  for (const globalRuleFile of GLOBAL_RULE_FILES) {
    if (await Bun.file(globalRuleFile).exists()) {
      paths.add(globalRuleFile)
      break
    }
  }

  // Config-based instructions
  if (config.instructions) {
    for (let instruction of config.instructions) {
      if (instruction.startsWith("~/")) {
        instruction = path.join(os.homedir(), instruction.slice(2))
      }
      // ... glob pattern resolution
    }
  }

  // Format each instruction
  const found = Array.from(paths).map((p) =>
    Bun.file(p)
      .text()
      .then((x) => "Instructions from: " + p + "\n" + x),
  )
  return Promise.all(found)
}
```

### Output Format

Each instruction file is prefixed with its source:

```
Instructions from: /home/user/myproject/AGENTS.md
## Project Guidelines

- Use Go idioms and error handling patterns
- Write table-driven tests
- Run `go fmt` before committing

Instructions from: /home/user/.claude/CLAUDE.md
## Personal Preferences

- Always explain your reasoning
- Prefer simple solutions
```

---

## 7. Final Message Structure

### Message Assembly

**File**: `packages/opencode/src/session/prompt.ts` (lines 559-581)

```typescript
messages: [
  ...system.map(
    (x): ModelMessage => ({
      role: "system",
      content: x,
    }),
  ),
  ...MessageV2.toModelMessage(
    msgs.filter(...)  // Conversation history
  ),
]
```

### Structure

The final system prompt is **2 messages** (for caching optimization):

**Message 1 (System)**:
- Provider header (Anthropic only)
- Base prompt (anthropic.txt, beast.txt, etc.)

**Message 2 (System)**:
- Environment context
- Custom instructions (joined with `\n`)

**Messages 3+ (User/Assistant)**:
- Converted conversation history

---

## 8. Complete Example

For a Claude model working on a Go project:

### System Message 1

```
You are Claude Code, Anthropic's official CLI for Claude.
You are OpenCode, the best coding agent on the planet.

You are an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

IMPORTANT: You must NEVER generate or guess URLs for the user unless you are confident that the URLs are for helping the user with programming. You may use URLs provided by the user in their messages or local files.

If the user asks for help or wants to give feedback inform them of the following:
- ctrl+p to list available actions
- To give feedback, users should report the issue at
  https://github.com/sst/opencode

[... rest of anthropic.txt - 106 lines total ...]

IMPORTANT: Always use the TodoWrite tool to plan and track tasks throughout the conversation.

# Code References

When referencing specific functions or pieces of code include the pattern `file_path:line_number` to allow the user to easily navigate to the source code location.

<example>
user: Where are errors from the client handled?
assistant: Clients are marked as failed in the `connectToServer` function in src/services/process.ts:712.
</example>
```

### System Message 2

```
Here is some useful information about the environment you are running in:
<env>
  Working directory: /home/user/mygoproject
  Is directory a git repo: yes
  Platform: linux
  Today's date: Sun Nov 24 2024
</env>
<files>
  mygoproject/
    .git/
    cmd/
      server/
        main.go
    internal/
      handler/
        handler.go
      service/
        service.go
    pkg/
      utils/
        helpers.go
    go.mod
    go.sum
    main.go
    README.md
    Dockerfile
    .gitignore
</files>

Instructions from: /home/user/mygoproject/AGENTS.md
## Go Development Guidelines

- Follow standard Go project layout (cmd/, internal/, pkg/)
- Use `go fmt` for formatting
- Handle errors explicitly, don't ignore them
- Use table-driven tests
- Run `go vet` before committing

Instructions from: /home/user/.claude/CLAUDE.md
## Personal Preferences

- Always explain your reasoning
- Show file paths with line numbers
```

---

## 9. Model-Specific Variations

### Anthropic (Claude)

- **Header**: "You are Claude Code, Anthropic's official CLI for Claude."
- **Base**: anthropic.txt (full TodoWrite instructions)
- **Focus**: Task management, tool parallelism, code references

### OpenAI (GPT-4, o1, o3)

- **Header**: None
- **Base**: beast.txt
- **Focus**: Autonomous problem-solving, extensive research, rigorous testing

### OpenAI (GPT-5)

- **Header**: None
- **Base**: codex.txt
- **Focus**: Detailed workflows, sandbox/approvals, AGENTS.md spec

### Google (Gemini)

- **Header**: None
- **Base**: gemini.txt
- **Focus**: Gemini-specific capabilities

### Other Models

- **Header**: None
- **Base**: qwen.txt (minimal)
- **Focus**: Concise responses (1-3 sentences), safety warnings

---

## Key Implementation Details

### Caching Optimization

The system prompt is limited to 2 messages:
- First message: Raw header + base prompt
- Second message: Combined environment + custom instructions

This enables prompt caching at LLM provider level.

### Dynamic File References

**File**: `packages/opencode/src/session/prompt.ts` (lines 145-191)

Prompts support `[[file:path]]` syntax for dynamic file inclusion.

### Agent Overrides

Agents can completely replace the base prompt with their own via the `agent.prompt` field.

---

## Files Summary

| File | Lines | Purpose |
|------|-------|---------|
| `src/session/system.ts` | 1-146 | Core system prompt assembly |
| `src/session/prompt.ts` | 621-641 | resolveSystemPrompt function |
| `src/session/prompt/anthropic.txt` | 106 | Claude base prompt |
| `src/session/prompt/beast.txt` | - | GPT base prompt |
| `src/session/prompt/anthropic_spoof.txt` | 1 | Anthropic header |
