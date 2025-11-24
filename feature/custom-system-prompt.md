# Feature Plan: Custom System and Initial Prompt Templates Per Session

## Executive Summary

This document outlines the design and implementation plan for enabling custom system and initial instruction prompts on a per-session basis. This feature will allow users to create specialized agents (e.g., data analyst, Python expert, security auditor) by providing custom prompt templates when starting a session.

**Status:** Planning
**Priority:** Medium
**Complexity:** Medium
**Estimated Files to Modify:** 4-6

---

## Table of Contents

1. [Current Architecture](#current-architecture)
2. [Problem Statement](#problem-statement)
3. [Proposed Solution](#proposed-solution)
4. [Technical Design](#technical-design)
5. [Implementation Plan](#implementation-plan)
6. [API Changes](#api-changes)
7. [Backward Compatibility](#backward-compatibility)
8. [Testing Strategy](#testing-strategy)
9. [Future Enhancements](#future-enhancements)

---

## Current Architecture

### System Prompt Loading Mechanism

**Location:** `/packages/opencode/src/session/prompt.ts:621-641`

The `resolveSystemPrompt()` function assembles system prompts in the following **priority order**:

```typescript
async function resolveSystemPrompt(input: {
  system?: string              // 1. Per-request override (highest priority)
  agent: Agent.Info           // 2. Agent-specific prompt
  providerID: string
  modelID: string
}) {
  let system = SystemPrompt.header(providerID)        // Provider-specific header

  system.push(
    ...(() => {
      if (input.system) return [input.system]         // Step 1: Custom override
      if (input.agent.prompt) return [input.agent.prompt]  // Step 2: Agent prompt
      return SystemPrompt.provider(modelID)           // Step 3: Model-specific default
    })()
  )

  system.push(...(await SystemPrompt.environment()))  // Step 4: Environment context
  system.push(...(await SystemPrompt.custom()))       // Step 5: Custom instructions

  // Optimization: Combine into 2 messages for prompt caching
  const [first, ...rest] = system
  system = [first, rest.join("\n")]
  return system
}
```

### Prompt Template Files

**Location:** `/packages/opencode/src/session/prompt/*.txt`

| Template File | Model Target | Size | Purpose |
|--------------|--------------|------|---------|
| `anthropic.txt` | Claude | 8.2 KB | General coding assistant |
| `beast.txt` | GPT-4/o1/o3 | 11 KB | Autonomous problem-solving |
| `gemini.txt` | Gemini | 15 KB | Gemini-specific instructions |
| `codex.txt` | GPT-5 | 24 KB | Detailed workflows |
| `qwen.txt` | Other | 9.7 KB | Minimal prompt |
| `polaris.txt` | Polaris-alpha | 8.3 KB | Polaris-specific |

**Selection Logic:** `/packages/opencode/src/session/system.ts:27-34`

```typescript
export function provider(modelID: string) {
  if (modelID.includes("gpt-5")) return [PROMPT_CODEX]
  if (modelID.includes("gpt-") || modelID.includes("o1") || modelID.includes("o3")) return [PROMPT_BEAST]
  if (modelID.includes("gemini-")) return [PROMPT_GEMINI]
  if (modelID.includes("claude")) return [PROMPT_ANTHROPIC]
  if (modelID.includes("polaris-alpha")) return [PROMPT_POLARIS]
  return [PROMPT_ANTHROPIC_WITHOUT_TODO]  // Default
}
```

### Session Schema

**Location:** `/packages/opencode/src/session/index.ts:37-75`

```typescript
export const Info = z.object({
  id: Identifier.schema("session"),
  projectID: z.string(),
  directory: z.string(),
  parentID: Identifier.schema("session").optional(),
  summary: z.object({...}).optional(),
  share: z.object({...}).optional(),
  title: z.string(),
  version: z.string(),
  time: z.object({...}),
  revert: z.object({...}).optional(),
})
```

### Session Creation Flow

**API Endpoint:** `POST /session`
**Handler:** `/packages/opencode/src/server/server.ts:516-521`

```typescript
validator("json", Session.create.schema.optional()),
async (c) => {
  const body = c.req.valid("json") ?? {}
  const session = await Session.create(body)  // Currently accepts: {parentID?, title?}
  return c.json(session)
}
```

**Session.create Function:** `/packages/opencode/src/session/index.ts:122-135`

```typescript
export const create = fn(
  z.object({
    parentID: Identifier.schema("session").optional(),
    title: z.string().optional(),
  }).optional(),
  async (input) => {
    return createNext({
      parentID: input?.parentID,
      directory: Instance.directory,
      title: input?.title,
    })
  }
)
```

---

## Problem Statement

### Current Limitations

1. **No Persistent Session-Level Customization**
   - The `system` parameter in `PromptInput` must be passed on **every message request**
   - No way to set a custom prompt once during session creation and have it persist
   - Cumbersome for multi-turn conversations with specialized agents

2. **Agent Configs Are Global**
   - Agent configurations in `~/.opencode/agent/*.md` are project/user-wide
   - Cannot create ephemeral, one-off specialized sessions without modifying configs
   - No way to experiment with different prompts without file system changes

3. **Template Reusability**
   - Users cannot easily create and reference reusable prompt templates
   - No mechanism to version or share prompt templates across teams

### Use Cases

1. **Data Analyst Agent**
   ```bash
   # User wants to start a session with data analysis focus
   opencode --prompt templates/data-analyst.txt
   ```

2. **Security Auditor**
   ```bash
   # Security-focused session for code review
   opencode --prompt security-auditor
   ```

3. **Domain-Specific Agents**
   ```bash
   # Medical records processing (HIPAA-compliant)
   # Financial analysis (SOX-compliant)
   # Legal document review
   ```

4. **A/B Testing Prompts**
   - Test different prompt variations without editing config files
   - Compare agent behavior with different system prompts

---

## Proposed Solution

### Design Principles

1. **Persistent but Optional:** Custom prompts stored in session metadata, falling back to existing behavior
2. **File-Based Templates:** Support loading prompts from files for reusability
3. **Inline Prompts:** Support inline prompt strings for quick experiments
4. **Backward Compatible:** Zero breaking changes to existing API
5. **Composable:** Custom prompts work with existing environment/instruction system

### Solution Overview

Add **session-level custom prompt templates** that:
- Are specified once during session creation
- Persist in session metadata
- Take precedence between agent prompts and model-specific defaults
- Support both file paths and inline strings

### Priority Order (Updated)

```
1. Per-request `system` parameter (API override)
2. Agent-specific `agent.prompt` (from agent config)
3. ✨ NEW: Session-level `customPromptTemplate` (from session metadata)
4. Model-specific default (anthropic.txt, beast.txt, etc.)
5. Environment context (git status, file tree, etc.)
6. Custom instructions (AGENTS.md, CLAUDE.md, etc.)
```

---

## Technical Design

### 1. Schema Changes

#### Session.Info Schema Extension

**File:** `/packages/opencode/src/session/index.ts`

```typescript
export const Info = z.object({
  id: Identifier.schema("session"),
  projectID: z.string(),
  directory: z.string(),
  parentID: Identifier.schema("session").optional(),

  // ✨ NEW: Custom prompt template
  customPrompt: z.object({
    type: z.enum(["file", "inline"]),
    value: z.string(),  // File path or inline prompt text
    loadedAt: z.number().optional(),  // Timestamp for cache invalidation
  }).optional(),

  summary: z.object({...}).optional(),
  share: z.object({...}).optional(),
  title: z.string(),
  version: z.string(),
  time: z.object({...}),
  revert: z.object({...}).optional(),
})
```

#### Session.create Schema Extension

**File:** `/packages/opencode/src/session/index.ts`

```typescript
export const create = fn(
  z.object({
    parentID: Identifier.schema("session").optional(),
    title: z.string().optional(),

    // ✨ NEW: Custom prompt options
    customPrompt: z.union([
      z.string(),  // Shorthand: file path or inline text (auto-detect)
      z.object({
        type: z.enum(["file", "inline"]),
        value: z.string(),
      }),
    ]).optional(),
  }).optional(),
  async (input) => {
    // Implementation details below...
  }
)
```

### 2. Prompt Loading Logic

#### New Helper: `SystemPrompt.fromSession()`

**File:** `/packages/opencode/src/session/system.ts`

```typescript
export async function fromSession(sessionID: string): Promise<string | null> {
  const session = await Session.get(sessionID)
  if (!session.customPrompt) return null

  if (session.customPrompt.type === "inline") {
    return session.customPrompt.value
  }

  if (session.customPrompt.type === "file") {
    const filePath = resolveTemplatePath(session.customPrompt.value)

    // Cache check (optional optimization)
    const fileStats = await Bun.file(filePath).stat()
    if (session.customPrompt.loadedAt && fileStats.mtime.getTime() <= session.customPrompt.loadedAt) {
      // File hasn't changed, could use cached version
    }

    const content = await Bun.file(filePath).text()
    return content
  }

  return null
}

function resolveTemplatePath(value: string): string {
  // Priority order for file resolution:
  // 1. Absolute path: /path/to/template.txt
  // 2. Home directory: ~/templates/data-analyst.txt
  // 3. Project .opencode/prompts/: template.txt → .opencode/prompts/template.txt
  // 4. Global ~/.opencode/prompts/: template.txt → ~/.opencode/prompts/template.txt

  if (path.isAbsolute(value)) return value
  if (value.startsWith("~/")) return path.join(os.homedir(), value.slice(2))

  // Check project-level prompts
  const projectPrompt = path.join(Instance.directory, ".opencode", "prompts", value)
  if (Bun.file(projectPrompt).exists()) return projectPrompt

  // Check global prompts
  const globalPrompt = path.join(Global.Path.config, "prompts", value)
  if (Bun.file(globalPrompt).exists()) return globalPrompt

  // Fallback: treat as relative to cwd
  return path.resolve(Instance.directory, value)
}
```

#### Updated `resolveSystemPrompt()`

**File:** `/packages/opencode/src/session/prompt.ts`

```typescript
async function resolveSystemPrompt(input: {
  system?: string
  agent: Agent.Info
  providerID: string
  modelID: string
  sessionID: string  // ✨ NEW: Need session ID to load custom prompt
}) {
  let system = SystemPrompt.header(input.providerID)

  system.push(
    ...(() => {
      if (input.system) return [input.system]  // 1. Per-request override
      if (input.agent.prompt) return [input.agent.prompt]  // 2. Agent prompt

      // ✨ NEW: 3. Session-level custom prompt
      const sessionPrompt = await SystemPrompt.fromSession(input.sessionID)
      if (sessionPrompt) return [sessionPrompt]

      return SystemPrompt.provider(input.modelID)  // 4. Model default
    })()
  )

  system.push(...(await SystemPrompt.environment()))  // 5. Environment
  system.push(...(await SystemPrompt.custom()))       // 6. Custom instructions

  const [first, ...rest] = system
  system = [first, rest.join("\n")]
  return system
}
```

**Note:** Need to pass `sessionID` to `resolveSystemPrompt()` - already available in calling context at line 495.

### 3. Session Creation Logic

#### Updated `createNext()`

**File:** `/packages/opencode/src/session/index.ts`

```typescript
export async function createNext(input: {
  id?: string
  title?: string
  parentID?: string
  directory: string
  customPrompt?: {    // ✨ NEW
    type: "file" | "inline"
    value: string
  }
}) {
  const result: Info = {
    id: Identifier.descending("session", input.id),
    version: Installation.VERSION,
    projectID: Instance.project.id,
    directory: input.directory,
    parentID: input.parentID,
    title: input.title ?? createDefaultTitle(!!input.parentID),

    // ✨ NEW: Store custom prompt metadata
    customPrompt: input.customPrompt ? {
      type: input.customPrompt.type,
      value: input.customPrompt.value,
      loadedAt: Date.now(),
    } : undefined,

    time: {
      created: Date.now(),
      updated: Date.now(),
    },
  }

  await Storage.write(["session", Instance.project.id, result.id], result)
  // ... rest of existing logic
  return result
}
```

### 4. Auto-Detection Logic

**File:** `/packages/opencode/src/session/index.ts`

```typescript
function parseCustomPromptInput(input: string | { type: string; value: string }) {
  if (typeof input === "object") {
    return input as { type: "file" | "inline"; value: string }
  }

  // Auto-detect: if it looks like a file path, treat as file
  // Otherwise, treat as inline prompt

  const isFilePath =
    input.startsWith("/") ||           // Absolute path
    input.startsWith("~/") ||          // Home directory
    input.startsWith("./") ||          // Relative path
    input.startsWith("../") ||         // Parent directory
    input.endsWith(".txt") ||          // Common extension
    input.endsWith(".md") ||
    !input.includes("\n")              // Single line = likely a path

  return {
    type: isFilePath ? "file" as const : "inline" as const,
    value: input,
  }
}

export const create = fn(
  z.object({
    parentID: Identifier.schema("session").optional(),
    title: z.string().optional(),
    customPrompt: z.union([
      z.string(),
      z.object({
        type: z.enum(["file", "inline"]),
        value: z.string(),
      }),
    ]).optional(),
  }).optional(),
  async (input) => {
    const customPrompt = input?.customPrompt
      ? parseCustomPromptInput(input.customPrompt)
      : undefined

    return createNext({
      parentID: input?.parentID,
      directory: Instance.directory,
      title: input?.title,
      customPrompt,
    })
  }
)
```

---

## Implementation Plan

### Phase 1: Core Implementation (Priority: High)

#### Task 1.1: Extend Session Schema
**File:** `/packages/opencode/src/session/index.ts`

- [ ] Add `customPrompt` field to `Session.Info` schema (lines 37-71)
- [ ] Add `customPrompt` parameter to `Session.create` schema (lines 122-135)
- [ ] Add `customPrompt` parameter to `createNext()` function (lines 175-208)
- [ ] Implement `parseCustomPromptInput()` helper function
- [ ] Update session storage to persist custom prompt metadata

**Complexity:** Low
**Risk:** Low (additive change, backward compatible)

#### Task 1.2: Implement Prompt Loading
**File:** `/packages/opencode/src/session/system.ts`

- [ ] Add `fromSession()` function to load session-level prompts
- [ ] Implement `resolveTemplatePath()` helper for file resolution
- [ ] Add error handling for missing/invalid template files
- [ ] Add logging for prompt loading (debugging)

**Complexity:** Medium
**Risk:** Medium (file I/O, path resolution edge cases)

#### Task 1.3: Update Prompt Resolution
**File:** `/packages/opencode/src/session/prompt.ts`

- [ ] Pass `sessionID` to `resolveSystemPrompt()` function (line 621)
- [ ] Call `SystemPrompt.fromSession()` in priority order (line 629-633)
- [ ] Update all call sites of `resolveSystemPrompt()` to include sessionID
- [ ] Verify prompt caching still works correctly

**Complexity:** Low
**Risk:** Low (small change to existing function)

#### Task 1.4: API Validation
**File:** `/packages/opencode/src/server/server.ts`

- [ ] Verify OpenAPI schema includes new `customPrompt` field (line 516)
- [ ] Test API endpoint with new parameter
- [ ] Add validation for file path security (no directory traversal)

**Complexity:** Low
**Risk:** Medium (security validation important)

### Phase 2: CLI Integration (Priority: Medium)

#### Task 2.1: Add CLI Flag
**File:** `/packages/opencode/src/cli/cmd/*.ts` (TBD - find CLI entry point)

- [ ] Add `--prompt <template>` or `--system-prompt <template>` flag
- [ ] Add `--prompt-file <path>` flag (explicit file mode)
- [ ] Add `--prompt-inline <text>` flag (explicit inline mode)
- [ ] Update help text and documentation

**Complexity:** Low
**Risk:** Low

#### Task 2.2: Template Discovery Command
**File:** New file or existing CLI command

- [ ] Add command to list available prompt templates
  ```bash
  opencode prompts list
  # Output:
  # Project templates (.opencode/prompts/):
  #   - data-analyst.txt
  #   - security-auditor.txt
  #
  # Global templates (~/.opencode/prompts/):
  #   - python-expert.txt
  #   - frontend-specialist.txt
  ```

**Complexity:** Low
**Risk:** Low

### Phase 3: User Experience (Priority: Low)

#### Task 3.1: Template Management

- [ ] Add `opencode prompts create <name>` command
- [ ] Add `opencode prompts edit <name>` command
- [ ] Add `opencode prompts show <name>` command
- [ ] Add `opencode prompts delete <name>` command

**Complexity:** Medium
**Risk:** Low

#### Task 3.2: Session Inspection

- [ ] Add session info display showing which custom prompt is active
- [ ] Add to `GET /session/:id` response
- [ ] Show in TUI/CLI session details

**Complexity:** Low
**Risk:** Low

---

## API Changes

### REST API

#### `POST /session` (Session Creation)

**Before:**
```json
{
  "parentID": "session_abc123",
  "title": "My Session"
}
```

**After (Backward Compatible):**
```json
{
  "parentID": "session_abc123",
  "title": "Data Analysis Session",
  "customPrompt": "data-analyst.txt"
}
```

**Or with explicit type:**
```json
{
  "customPrompt": {
    "type": "file",
    "value": "/path/to/templates/data-analyst.txt"
  }
}
```

**Or inline:**
```json
{
  "customPrompt": {
    "type": "inline",
    "value": "You are a specialized data analyst. Focus on statistical analysis and visualization..."
  }
}
```

#### `GET /session/:id` (Session Details)

**Response includes new field:**
```json
{
  "id": "session_xyz789",
  "title": "Data Analysis Session",
  "customPrompt": {
    "type": "file",
    "value": "data-analyst.txt",
    "loadedAt": 1732464000000
  },
  ...
}
```

### CLI

```bash
# Start session with file-based template
opencode --prompt data-analyst.txt

# Explicit file mode
opencode --prompt-file ~/.opencode/prompts/security.txt

# Inline prompt (single-line)
opencode --prompt-inline "You are a Python expert focusing on type safety"

# List available templates
opencode prompts list

# Create new template
opencode prompts create data-analyst
# Opens editor with template from anthropic.txt

# Show active prompt for current session
opencode session info
```

---

## Backward Compatibility

### ✅ Zero Breaking Changes

1. **Schema:** `customPrompt` is optional field
2. **API:** Existing API calls work identically
3. **Behavior:** Sessions without custom prompts behave exactly as before
4. **Storage:** Existing sessions are valid (missing field = undefined)

### Migration

**Not required** - feature is fully additive.

Existing sessions will continue to work with:
- Agent prompts (if configured)
- Model-specific defaults
- Environment context
- Custom instructions

---

## Testing Strategy

### Unit Tests

**File:** `/packages/opencode/test/session/custom-prompt.test.ts` (new)

```typescript
import { describe, test, expect } from "bun:test"

describe("Custom Prompt Templates", () => {
  test("session creation with file-based prompt", async () => {
    const session = await Session.create({
      title: "Test Session",
      customPrompt: "test-prompt.txt",
    })
    expect(session.customPrompt?.type).toBe("file")
    expect(session.customPrompt?.value).toBe("test-prompt.txt")
  })

  test("session creation with inline prompt", async () => {
    const session = await Session.create({
      customPrompt: {
        type: "inline",
        value: "You are a test assistant",
      },
    })
    expect(session.customPrompt?.type).toBe("inline")
  })

  test("auto-detection: file path", () => {
    const result = parseCustomPromptInput("~/templates/analyst.txt")
    expect(result.type).toBe("file")
  })

  test("auto-detection: inline text", () => {
    const result = parseCustomPromptInput("You are an assistant\nwith multiple lines")
    expect(result.type).toBe("inline")
  })

  test("template resolution: project-level", async () => {
    // Create .opencode/prompts/test.txt
    const path = await resolveTemplatePath("test.txt")
    expect(path).toContain(".opencode/prompts/test.txt")
  })

  test("template resolution: global", async () => {
    const path = await resolveTemplatePath("global.txt")
    expect(path).toContain(".opencode/prompts/global.txt")
  })

  test("prompt loading from session", async () => {
    const session = await Session.create({
      customPrompt: { type: "inline", value: "Test prompt" },
    })
    const prompt = await SystemPrompt.fromSession(session.id)
    expect(prompt).toBe("Test prompt")
  })
})
```

### Integration Tests

```typescript
describe("End-to-End Custom Prompts", () => {
  test("custom prompt used in message flow", async () => {
    // 1. Create session with custom prompt
    const session = await Session.create({
      customPrompt: { type: "inline", value: "You are a math tutor" },
    })

    // 2. Send message
    const response = await SessionPrompt.prompt({
      sessionID: session.id,
      parts: [{ type: "text", text: "What is 2+2?" }],
    })

    // 3. Verify custom prompt was loaded (check logs or system messages)
    // ... implementation specific verification
  })

  test("prompt precedence: per-request overrides session", async () => {
    const session = await Session.create({
      customPrompt: { type: "inline", value: "Session prompt" },
    })

    const response = await SessionPrompt.prompt({
      sessionID: session.id,
      system: "Request prompt",  // Should override session prompt
      parts: [{ type: "text", text: "Test" }],
    })

    // Verify "Request prompt" was used, not "Session prompt"
  })
})
```

### Manual Testing Checklist

- [ ] Create session via API with file-based prompt
- [ ] Create session via CLI with `--prompt` flag
- [ ] Verify prompt loads from `.opencode/prompts/`
- [ ] Verify prompt loads from `~/.opencode/prompts/`
- [ ] Test absolute path prompts
- [ ] Test inline prompts
- [ ] Test auto-detection (file vs inline)
- [ ] Test missing file error handling
- [ ] Test invalid path security (directory traversal)
- [ ] Verify prompt precedence order
- [ ] Test session without custom prompt (backward compat)
- [ ] Test session export/import with custom prompts

---

## Security Considerations

### Path Traversal Prevention

```typescript
function resolveTemplatePath(value: string): string {
  const resolved = /* ... resolution logic ... */

  // Security: Ensure resolved path is within allowed directories
  const allowedDirs = [
    Instance.directory,           // Project directory
    Global.Path.config,           // ~/.opencode/
    os.homedir(),                 // User home (for ~/ paths)
  ]

  const normalizedPath = path.normalize(resolved)
  const isAllowed = allowedDirs.some(dir =>
    normalizedPath.startsWith(path.normalize(dir))
  )

  if (!isAllowed) {
    throw new Error(`Invalid template path: ${value} (outside allowed directories)`)
  }

  return normalizedPath
}
```

### File Size Limits

```typescript
export async function fromSession(sessionID: string): Promise<string | null> {
  // ... existing logic ...

  if (session.customPrompt.type === "file") {
    const file = Bun.file(filePath)
    const size = (await file.stat()).size

    // Limit: 100 KB for prompt templates
    if (size > 100 * 1024) {
      throw new Error(`Prompt template too large: ${size} bytes (max 100 KB)`)
    }

    return await file.text()
  }
}
```

---

## Example Prompt Templates

### Data Analyst Template

**File:** `.opencode/prompts/data-analyst.txt`

```
You are OpenCode configured as a specialized Data Analyst assistant.

# Core Expertise
- Statistical analysis and hypothesis testing
- Data cleaning and preprocessing
- Exploratory data analysis (EDA)
- Data visualization best practices
- Python data stack: pandas, numpy, scipy, matplotlib, seaborn

# Analysis Workflow
When analyzing data:
1. Understand the data structure and quality
2. Check for missing values, outliers, duplicates
3. Perform descriptive statistics
4. Create visualizations to identify patterns
5. Document findings with clear explanations

# Code Style
- Use type hints for pandas DataFrames
- Add docstrings to analysis functions
- Include comments explaining statistical choices
- Create reproducible analysis scripts

# Communication
- Explain statistical concepts in plain language
- Always validate assumptions before applying tests
- Suggest appropriate visualizations for each data type
- Flag potential data quality issues proactively

[Rest of base prompt from anthropic.txt will be appended]
```

### Security Auditor Template

**File:** `.opencode/prompts/security-auditor.txt`

```
You are OpenCode configured as a Security Auditor specializing in code security review.

# Security Focus Areas
- OWASP Top 10 vulnerabilities
- Input validation and sanitization
- Authentication and authorization flaws
- Cryptographic implementation issues
- Dependency vulnerabilities
- Information disclosure risks

# Review Methodology
When reviewing code:
1. Identify all external input points
2. Trace data flow through the application
3. Check for injection vulnerabilities (SQL, XSS, Command)
4. Verify authentication/authorization checks
5. Review cryptographic implementations
6. Check for sensitive data exposure

# Reporting
- Flag HIGH/MEDIUM/LOW severity issues
- Provide specific line numbers and code references
- Suggest concrete fixes with code examples
- Reference CVE/CWE identifiers when applicable

# Tools Preference
- Use grep/ripgrep for pattern-based security scans
- Recommend security linters (bandit, semgrep)
- Suggest dependency audit tools

[Base prompt continues...]
```

---

## Future Enhancements

### Phase 4: Advanced Features (Post-MVP)

1. **Prompt Variables/Interpolation**
   ```
   You are analyzing the ${PROJECT_NAME} codebase.
   Primary language: ${PRIMARY_LANGUAGE}
   ```

2. **Prompt Composition**
   ```json
   {
     "customPrompt": {
       "base": "data-analyst.txt",
       "extends": ["python-expert.txt", "visualization.txt"]
     }
   }
   ```

3. **Conditional Prompts**
   ```json
   {
     "customPrompt": {
       "file": "analyst.txt",
       "conditions": {
         "if_language": {
           "python": "python-analyst.txt",
           "javascript": "js-analyst.txt"
         }
       }
     }
   }
   ```

4. **Prompt Templates Registry**
   - Community-shared templates
   - Template versioning
   - Template marketplace

5. **Dynamic Prompt Updates**
   - Allow updating session prompt mid-conversation
   - API: `PATCH /session/:id/prompt`

6. **Prompt Analytics**
   - Track which prompts lead to better outcomes
   - A/B testing framework
   - Usage statistics

---

## Migration Guide

### For Users

**Before (using agent configs):**
```yaml
# ~/.opencode/agent/data-analyst.md
---
description: "Data analyst agent"
model: "anthropic/claude-sonnet-4"
---

You are a data analyst...
```

**After (using session templates):**
```bash
# Create reusable template
mkdir -p ~/.opencode/prompts
cat > ~/.opencode/prompts/data-analyst.txt << 'EOF'
You are a data analyst...
EOF

# Start session with template
opencode --prompt data-analyst.txt
```

**Benefits:**
- Templates are lighter-weight than agents
- Can use different templates with same agent
- Easier to experiment without modifying configs

### For API Users

**Before:**
```javascript
// Had to pass system prompt on EVERY request
await fetch('http://localhost:3456/session/xxx/message', {
  method: 'POST',
  body: JSON.stringify({
    system: "You are a data analyst...",  // Repeated every time
    parts: [{ type: "text", text: "Analyze this data" }]
  })
})
```

**After:**
```javascript
// Set once during session creation
const session = await fetch('http://localhost:3456/session', {
  method: 'POST',
  body: JSON.stringify({
    title: "Data Analysis",
    customPrompt: "data-analyst.txt"
  })
}).then(r => r.json())

// Subsequent requests automatically use custom prompt
await fetch(`http://localhost:3456/session/${session.id}/message`, {
  method: 'POST',
  body: JSON.stringify({
    parts: [{ type: "text", text: "Analyze this data" }]
  })
})
```

---

## Success Metrics

1. **Adoption:**
   - % of sessions using custom prompts
   - Number of templates created per user
   - Template reuse frequency

2. **Performance:**
   - Prompt loading latency (target: <50ms)
   - Cache hit rate for file-based templates
   - No regression in message response time

3. **User Satisfaction:**
   - User feedback on ease of use
   - Number of issues related to custom prompts
   - Documentation clarity ratings

---

## Open Questions

1. **Template Format:**
   - Should we support JSON/YAML metadata in template files (like agent configs)?
   - Should templates support frontmatter for metadata?

   ```markdown
   ---
   name: Data Analyst
   version: 1.0
   author: user@example.com
   ---

   You are a data analyst...
   ```

2. **Template Validation:**
   - Should we validate template contents before accepting?
   - Warn on very long prompts that might hit token limits?

3. **Template Inheritance:**
   - Should session child inherit parent's custom prompt?
   - Or always use default unless explicitly set?

4. **Prompt Visibility:**
   - Should users be able to view the final assembled system prompt?
   - API endpoint: `GET /session/:id/prompt/resolved`

---

## Related Documentation

- [Agent Configuration](https://opencode.ai/docs/agents)
- [System Prompt Architecture](#current-architecture)
- [Session Management API](https://opencode.ai/docs/api/sessions)
- [Custom Instructions](https://opencode.ai/docs/custom-instructions)

---

## Appendix: File Modification Summary

| File Path | Lines Modified | Changes |
|-----------|---------------|---------|
| `/packages/opencode/src/session/index.ts` | ~50 | Add schema fields, update create() |
| `/packages/opencode/src/session/system.ts` | ~80 | Add fromSession(), resolveTemplatePath() |
| `/packages/opencode/src/session/prompt.ts` | ~10 | Update resolveSystemPrompt() |
| `/packages/opencode/src/server/server.ts` | ~5 | Verify schema validation |
| **Total Estimated LOC** | **~145** | Core implementation |

---

**Document Version:** 1.0
**Last Updated:** 2024-11-24
**Author:** OpenCode Analysis
**Status:** Ready for Review
