# Golang Project Prompts Analysis

This document provides a comprehensive analysis of what prompts are sent to the model when working on a Golang project in OpenCode.

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Prompt Construction](#2-system-prompt-construction)
3. [Prompt Components](#3-prompt-components)
4. [Go-Specific Detection](#4-go-specific-detection)
5. [Complete Example Prompt](#5-complete-example-prompt)
6. [Go-Specific Tooling](#6-go-specific-tooling)
7. [Key Finding](#7-key-finding)

---

## 1. Executive Summary

**Important Finding**: There are **NO language-specific instructions for Go** in the system prompts. The model receives:

- Generic coding instructions (same for all languages)
- Project structure (which happens to include `.go` files)
- Custom `AGENTS.md` instructions (if provided by the user)
- Go LSP diagnostics (when errors are reported via gopls)

The model must infer Go best practices from the project context and its training.

---

## 2. System Prompt Construction

### Primary Entry Point

**File**: `packages/opencode/src/session/prompt.ts` (lines 465-470)

```typescript
const system = await resolveSystemPrompt({
  providerID: model.providerID,
  modelID: model.info.id,
  agent,
  system: lastUser.system,
})
```

### Resolution Function

**File**: `packages/opencode/src/session/prompt.ts` (lines 621-641)

```typescript
async function resolveSystemPrompt(input: {
  system?: string
  agent: Agent.Info
  providerID: string
  modelID: string
}) {
  let system = SystemPrompt.header(input.providerID)
  system.push(
    ...(() => {
      if (input.system) return [input.system]
      if (input.agent.prompt) return [input.agent.prompt]
      return SystemPrompt.provider(input.modelID)
    })(),
  )
  system.push(...(await SystemPrompt.environment()))
  system.push(...(await SystemPrompt.custom()))
  // max 2 system prompt messages for caching purposes
  const [first, ...rest] = system
  system = [first, rest.join("\n")]
  return system
}
```

---

## 3. Prompt Components

The final prompt sent consists of these components in order:

### 1. Provider Header

**File**: `packages/opencode/src/session/system.ts` (lines 22-25)

```typescript
export function header(providerID: string) {
  if (providerID.includes("anthropic")) return [PROMPT_ANTHROPIC_SPOOF.trim()]
  return []
}
```

### 2. Model-Specific Base Prompt

**File**: `packages/opencode/src/session/system.ts` (lines 27-34)

```typescript
export function provider(modelID: string) {
  if (modelID.includes("gpt-5")) return [PROMPT_CODEX]
  if (modelID.includes("gpt-") || modelID.includes("o1") || modelID.includes("o3")) return [PROMPT_BEAST]
  if (modelID.includes("gemini-")) return [PROMPT_GEMINI]
  if (modelID.includes("claude")) return [PROMPT_ANTHROPIC]
  if (modelID.includes("polaris-alpha")) return [PROMPT_POLARIS]
  return [PROMPT_ANTHROPIC_WITHOUT_TODO]
}
```

**Prompt files by model**:

| Model | Prompt File | Lines |
|-------|-------------|-------|
| Claude | `src/session/prompt/anthropic.txt` | 106 |
| GPT-4/o1/o3 | `src/session/prompt/beast.txt` | lengthy |
| Gemini | `src/session/prompt/gemini.txt` | 156 |
| Polaris | `src/session/prompt/polaris.txt` | - |
| GPT-5 | `src/session/prompt/codex.txt` | 319 |
| Other | `src/session/prompt/qwen.txt` | - |

### 3. Environment Context

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

For a Go project, this includes:
- Working directory path
- Git repo status
- Platform (Linux/macOS/Windows)
- Current date
- Project file tree (first 200 files/directories)

### 4. Custom Instructions

**File**: `packages/opencode/src/session/system.ts` (lines 71-118)

Custom instructions are loaded from (searched in order):

**Local files** (project-specific):
- `AGENTS.md` - Agent instructions for the repo
- `CLAUDE.md` - Legacy Claude instructions
- `CONTEXT.md` - Deprecated context file

**Global files** (user-level):
- `~/.claude/CLAUDE.md`
- `${Global.Path.config}/AGENTS.md`

---

## 4. Go-Specific Detection

### Go File Detection

**File**: `packages/opencode/src/lsp/language.ts` (line 35)

```typescript
".go": "go",
```

### Go Project Root Detection

**File**: `packages/opencode/src/lsp/server.ts` (lines 211-217)

```typescript
export const Gopls: Info = {
  id: "gopls",
  root: async (file) => {
    const work = await NearestRoot(["go.work"])(file)
    if (work) return work
    return NearestRoot(["go.mod", "go.sum"])(file)
  },
  extensions: [".go"],
  // ...
}
```

Language detection looks for:
1. `go.work` (Go workspace files)
2. `go.mod` + `go.sum` (standard Go module files)

### Gopls Language Server

**File**: `packages/opencode/src/lsp/server.ts` (lines 219-250)

```typescript
async spawn(root) {
  let bin = Bun.which("gopls", {
    PATH: process.env["PATH"] + ":" + Global.Path.bin,
  })
  if (!bin) {
    if (!Bun.which("go")) return
    if (Flag.OPENCODE_DISABLE_LSP_DOWNLOAD) return

    log.info("installing gopls")
    const proc = Bun.spawn({
      cmd: ["go", "install", "golang.org/x/tools/gopls@latest"],
      env: { ...process.env, GOBIN: Global.Path.bin },
      stdout: "pipe",
      stderr: "pipe",
      stdin: "pipe",
    })
    // ... installation logic
  }
  // ...
}
```

Gopls is automatically installed if not present.

---

## 5. Complete Example Prompt

When working on a Go project with Claude Sonnet, the model receives:

### Message 1 (System)

```
<anthropic_spoof_header_if_claude>

You are OpenCode, the best coding agent on the planet.

You are an interactive CLI tool that helps users with software engineering tasks.
Use the instructions below and the tools available to you to assist the user.

... [full anthropic.txt content - 106 lines of instructions about:
    - Tone and style
    - Task management
    - Tool usage
    - Code editing guidelines
    - Security considerations]
```

### Message 2 (System)

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
    go.mod
    go.sum
    main.go
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
    tests/
      integration_test.go
    README.md
    Dockerfile
    .gitignore
</files>

Instructions from: /home/user/mygoproject/AGENTS.md
... [custom instructions if file exists]
```

### Messages 3+

User messages with file contents, conversation history, tool calls, etc.

---

## 6. Go-Specific Tooling

### Go Formatter

**File**: `packages/opencode/src/format/formatter.ts` (lines 14-21)

```typescript
export const gofmt: Info = {
  name: "gofmt",
  command: ["gofmt", "-w", "$FILE"],
  extensions: [".go"],
  async enabled() {
    return Bun.which("gofmt") !== null
  },
}
```

OpenCode uses `gofmt` (Go's standard formatter) when available.

### Tool Context for Go Code

**File**: `packages/opencode/src/session/prompt.ts` (lines 559-598)

When processing Go code, the model has access to:

| Tool | Usage for Go |
|------|--------------|
| **Read** | Read `.go` files |
| **Bash** | Run `go test`, `go build`, `go run`, `go fmt` |
| **Edit** | Modify Go source files |
| **LSP symbols** | Via gopls integration |
| **MCP servers** | Any connected MCP servers |

The model sees `go.mod`/`go.sum` contents when referenced in conversation.

---

## 7. Key Finding

### No Go-Specific Instructions

**Important**: The system prompts contain **no language-specific instructions for Go**. The model receives:

1. **Generic coding instructions** (same for JavaScript, Python, Rust, etc.)
2. **Project structure** (which shows `.go` files, `go.mod`, etc.)
3. **Custom AGENTS.md** (if provided by user)
4. **Go LSP diagnostics** (when errors are reported)

### How Go Best Practices Are Inferred

The model must infer Go conventions from:

1. **Project structure** - Standard Go layout (`cmd/`, `internal/`, `pkg/`)
2. **Existing `.go` files** - When read during conversation
3. **`go.mod` contents** - Module path, dependencies
4. **LSP diagnostics** - Type errors, unused imports from gopls
5. **User's custom instructions** - AGENTS.md can specify Go guidelines
6. **Model's training** - Knowledge of Go idioms, error handling patterns, etc.

### Recommended AGENTS.md for Go Projects

Users can add Go-specific instructions in `AGENTS.md`:

```markdown
## Go Development Guidelines

- Follow standard Go project layout (cmd/, internal/, pkg/)
- Use `go fmt` for formatting
- Handle errors explicitly, don't ignore them
- Use table-driven tests
- Prefer composition over inheritance
- Use interfaces for dependency injection
- Run `go vet` and `golangci-lint` before committing
```

---

## Summary Table

| Aspect | Implementation | Go-Specific Details |
|--------|----------------|---------------------|
| **Language Detection** | File extension `.go` | Detected via LSP extensions |
| **Project Root** | `go.mod`, `go.sum`, `go.work` | Searches up directory tree |
| **LSP Server** | gopls (auto-installed) | `go install golang.org/x/tools/gopls@latest` |
| **Formatter** | gofmt | Called when saving Go files |
| **System Prompt** | Model-agnostic | No Go-specific instructions |
| **Environment Context** | File tree + metadata | Includes entire project structure |
| **Custom Instructions** | AGENTS.md/CLAUDE.md | User-provided only |
| **Diagnostics** | gopls errors/warnings | Reported through LSP |

The design is **language-agnostic** - OpenCode treats Go projects the same as JavaScript, Python, or Rust projects, relying on LSP integration and the model's inherent knowledge of language conventions.
