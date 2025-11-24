# LSP Selection Mechanism Analysis

This document provides a comprehensive analysis of how OpenCode decides which LSP (Language Server Protocol) server to use for different files and projects.

## Table of Contents

1. [Overview](#1-overview)
2. [Configuration Schema](#2-configuration-schema)
3. [Server Selection Algorithm](#3-server-selection-algorithm)
4. [Default LSP Servers](#4-default-lsp-servers)
5. [Root Detection Methods](#5-root-detection-methods)
6. [Custom LSP Configuration](#6-custom-lsp-configuration)
7. [Per-Project Settings](#7-per-project-settings)
8. [Auto-Discovery and Installation](#8-auto-discovery-and-installation)
9. [Multiple Servers Per File](#9-multiple-servers-per-file)
10. [Server Lifecycle](#10-server-lifecycle)

---

## 1. Overview

OpenCode's LSP selection is based on:

1. **File extension** - Primary matching criteria
2. **Project root detection** - Finding the appropriate workspace
3. **Configuration** - User-defined server settings
4. **Availability** - Whether the server binary exists

---

## 2. Configuration Schema

**File**: `packages/opencode/src/config/config.ts` (lines 565-600)

```typescript
lsp: z
  .union([
    z.literal(false),  // Disable all LSPs
    z.record(
      z.string(),      // Server ID (e.g., "gopls", "typescript")
      z.union([
        z.object({
          disabled: z.literal(true),  // Disable specific server
        }),
        z.object({
          command: z.array(z.string()),                    // Command to spawn
          extensions: z.array(z.string()).optional(),      // File extensions
          disabled: z.boolean().optional(),
          env: z.record(z.string(), z.string()).optional(), // Environment vars
          initialization: z.record(z.string(), z.any()).optional(), // Init options
        }),
      ]),
    ),
  ])
  .optional()
```

### Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `lsp: false` | boolean | Disables all LSP servers globally |
| `lsp.<id>.disabled` | boolean | Disables a specific LSP server |
| `lsp.<id>.command` | string[] | Custom command to run the LSP |
| `lsp.<id>.extensions` | string[] | File extensions to match |
| `lsp.<id>.env` | object | Environment variables for the LSP process |
| `lsp.<id>.initialization` | object | Initialization options passed to LSP |

---

## 3. Server Selection Algorithm

**File**: `packages/opencode/src/lsp/index.ts` (lines 156-240)

The `getClients(file)` function implements the selection:

```typescript
async function getClients(file: string) {
  const s = state()
  const extension = path.parse(file).ext || file  // Step 1: Get extension
  const result: LSPClient[] = []

  for (const [name, server] of Object.entries(s.servers)) {  // Step 2: Iterate servers
    // Step 3: Extension filtering
    if (server.extensions.length && !server.extensions.includes(extension)) continue

    // Step 4: Root detection
    const root = await server.root(file)
    if (!root) continue

    // Step 5: Skip broken servers
    if (s.broken.has(root + server.id)) continue

    // Step 6: Check cache
    const key = root + server.id
    const existing = s.clients[key]
    if (existing) {
      result.push(existing)
      continue
    }

    // Step 7: Check inflight spawns
    const inflight = s.spawning.get(key)
    if (inflight) {
      const client = await inflight
      if (client) result.push(client)
      continue
    }

    // Step 8: Spawn new server
    const promise = (async () => {
      const handle = await server.spawn(root)
      if (!handle) {
        s.broken.add(key)
        return undefined
      }
      const client = await LSPClient.create({...})
      s.clients[key] = client
      return client
    })()

    s.spawning.set(key, promise)
    const client = await promise
    if (client) result.push(client)
  }

  return result
}
```

### Selection Flow Summary

```
1. Extract file extension (.ts, .go, .py, etc.)
2. Load server configuration from Config
3. For each LSP server:
   a. Check if file extension matches server.extensions
   b. Call server.root(file) to determine project root
   c. Skip if root not found or server previously failed
   d. Check cache for existing client at (root, serverID)
   e. If cached, return cached client
   f. If spawning in progress, wait for completion
   g. Otherwise spawn new server process
4. Return array of all applicable LSP clients
```

---

## 4. Default LSP Servers

**File**: `packages/opencode/src/lsp/server.ts` (lines 13-1168)

OpenCode ships with 19 built-in LSP server definitions:

| Server | ID | Extensions | Root Detection |
|--------|-----|------------|----------------|
| **Deno** | `deno` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs` | `deno.json`, `deno.jsonc` |
| **TypeScript** | `typescript` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`, `.mts`, `.cts` | Lockfiles (excludes deno) |
| **Vue** | `vue` | `.vue` | Lockfiles |
| **ESLint** | `eslint` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mts`, `.cts`, `.vue` | Lockfiles |
| **Go** | `gopls` | `.go` | `go.work`, `go.mod`, `go.sum` |
| **Ruby** | `ruby-lsp` | `.rb`, `.rake`, `.gemspec`, `.ru` | `Gemfile` |
| **Python** | `pyright` | `.py`, `.pyi` | `pyproject.toml`, `requirements.txt`, etc. |
| **Elixir** | `elixir-ls` | `.ex`, `.exs` | `mix.exs`, `mix.lock` |
| **Zig** | `zls` | `.zig`, `.zon` | `build.zig` |
| **C#** | `csharp` | `.cs` | `.sln`, `.csproj`, `global.json` |
| **Swift** | `sourcekit-lsp` | `.swift`, `.objc`, `.objcpp` | `Package.swift`, xcodeproj |
| **Rust** | `rust` | `.rs` | `Cargo.toml`, `Cargo.lock` |
| **C/C++** | `clangd` | `.c`, `.cpp`, `.cc`, `.h`, `.hpp` | `compile_commands.json`, `CMakeLists.txt` |
| **Svelte** | `svelte` | `.svelte` | Lockfiles |
| **Astro** | `astro` | `.astro` | Lockfiles |
| **Java** | `jdtls` | `.java` | `pom.xml`, `build.gradle` |
| **YAML** | `yaml-ls` | `.yaml`, `.yml` | Lockfiles |
| **Lua** | `lua-ls` | `.lua` | `.luarc.json`, `.luacheckrc` |
| **PHP** | `php intelephense` | `.php` | `composer.json` |

---

## 5. Root Detection Methods

### NearestRoot Pattern

**File**: `packages/opencode/src/lsp/server.ts` (lines 23-45)

```typescript
function NearestRoot(includePatterns: string[], excludePatterns?: string[]) {
  return async (file: string) => {
    let dir = path.dirname(file)
    while (true) {
      // Check exclude patterns first
      if (excludePatterns) {
        for (const pattern of excludePatterns) {
          if (await Bun.file(path.join(dir, pattern)).exists()) {
            return undefined
          }
        }
      }
      // Check include patterns
      for (const pattern of includePatterns) {
        if (await Bun.file(path.join(dir, pattern)).exists()) {
          return dir
        }
      }
      // Walk up directory tree
      const parent = path.dirname(dir)
      if (parent === dir) break
      dir = parent
    }
    return Instance.directory  // Fallback
  }
}
```

### Language-Specific Root Detection

**TypeScript** (lines 85-88):
- Looks for: `package-lock.json`, `bun.lockb`, `yarn.lock`
- Excludes: `deno.json`, `deno.jsonc`

**Go** (lines 213-216):
- Prefers: `go.work`
- Falls back to: `go.mod`, `go.sum`

**Rust** (lines 586-614):
- Finds workspace root by searching for `[workspace]` in `Cargo.toml`

---

## 6. Custom LSP Configuration

Custom LSP servers can be configured in `opencode.jsonc`:

```jsonc
{
  "lsp": {
    "my-custom-server": {
      "command": ["node", "/path/to/server.js"],
      "extensions": [".custom", ".myext"],
      "env": {
        "CUSTOM_VAR": "value"
      },
      "initialization": {
        "customOption": "value"
      }
    }
  }
}
```

### Configuration Rules

1. **Built-in override**: If server ID matches a built-in (e.g., "typescript"), it replaces that server
2. **Custom requirement**: Custom servers must specify the `extensions` array
3. **Validation**: Config validates that custom servers have extensions

### Example: Override TypeScript LSP

```jsonc
{
  "lsp": {
    "typescript": {
      "command": ["custom-ts-server", "--stdio"],
      "extensions": [".ts", ".tsx"],
      "initialization": {
        "customOption": true
      }
    }
  }
}
```

---

## 7. Per-Project Settings

### Configuration Hierarchy

**File**: `packages/opencode/src/config/config.ts` (lines 24-94)

Priority order (later overrides earlier):

1. Global config: `~/.opencode/config.json`
2. Worktree config: `.opencode/opencode.jsonc`
3. Project config: `<project>/opencode.jsonc`
4. Environment variable: `OPENCODE_CONFIG`
5. Flag override: `OPENCODE_CONFIG_CONTENT`
6. Directory configs: All `.opencode` directories up the tree

### Merge Strategy

All configs are **deep-merged**. Example:

```
~/.opencode/config.json:              # Global defaults
  lsp:
    typescript: { command: [...] }

<workspace>/.opencode/opencode.jsonc: # Workspace override
  lsp:
    typescript:
      disabled: true

<project>/opencode.jsonc:             # Project override
  lsp:
    typescript:
      command: ["custom-ts-server"]   # Re-enables with custom command
```

---

## 8. Auto-Discovery and Installation

### Automatic Binary Download

OpenCode auto-downloads LSP servers on-demand unless disabled:

**Environment Variable**: `OPENCODE_DISABLE_LSP_DOWNLOAD`

### Auto-Installation Methods

| Server | Installation Method |
|--------|---------------------|
| **Gopls** | `go install golang.org/x/tools/gopls@latest` |
| **Pyright** | Downloads npm package to `$OPENCODE_BIN/node_modules/pyright` |
| **Clangd** | Downloads platform-specific binary from GitHub |
| **Zls** | Downloads and extracts platform-specific binary from GitHub |
| **ElixirLS** | Downloads from GitHub, compiles with `mix` |
| **JDTLS** | Downloads from Eclipse |
| **Ruby-LSP** | `gem install ruby-lsp` |
| **C#** | `dotnet tool install csharp-ls` |
| **Vue/Svelte/Astro** | Downloads from npm |

### Binary Discovery

Servers check multiple locations:

```typescript
let bin = Bun.which("gopls", {
  PATH: process.env["PATH"] + ":" + Global.Path.bin,
})
```

- System PATH
- OpenCode's bin directory (`~/.opencode/bin`)

---

## 9. Multiple Servers Per File

OpenCode can run **multiple LSP servers for the same file** simultaneously.

### Examples

| File Type | Active Servers |
|-----------|----------------|
| `.ts` file | `typescript`, `eslint` |
| `.vue` file | `vue`, `typescript` |
| `.tsx` file | `typescript`, `eslint` |

Each server provides different capabilities:
- TypeScript: Type checking, completions, hover
- ESLint: Linting, code style

### Selection Result

The `getClients()` function returns an **array** of all applicable clients:

```typescript
const clients = await getClients("/project/src/app.ts")
// Returns: [typescriptClient, eslintClient]
```

---

## 10. Server Lifecycle

### Initialization

**File**: `packages/opencode/src/lsp/client.ts` (lines 76-106)

```typescript
await connection.sendRequest("initialize", {
  rootUri: "file://" + root,
  initializationOptions: {
    ...input.server.initialization,
  },
  capabilities: {
    window: { workDoneProgress: true },
    workspace: { configuration: true },
    textDocument: {
      synchronization: {
        didOpen: true,
        didChange: true,
      },
      publishDiagnostics: {},
    },
  },
})
await connection.sendNotification("initialized")
```

### Disabling Servers

**Global disable**:
```jsonc
{
  "lsp": false
}
```

**Individual disable**:
```jsonc
{
  "lsp": {
    "typescript": { "disabled": true },
    "eslint": { "disabled": true }
  }
}
```

### Broken Server Tracking

**Lines 165-172, 207** in `lsp/index.ts`:

Failed servers are tracked to avoid repeated spawn attempts:

```typescript
if (!handle) {
  s.broken.add(key)  // Key: "{root}{serverId}"
  return undefined
}

// Later, skip broken servers:
if (s.broken.has(root + server.id)) continue
```

### Shutdown

**Lines 120-122** in `lsp/index.ts`:

```typescript
async function shutdown() {
  for (const client of Object.values(s.clients)) {
    await client.shutdown()
  }
}
```

---

## Summary: Decision Tree

| Condition | Action |
|-----------|--------|
| `lsp: false` globally | No LSP servers run |
| Server `disabled: true` | Server removed from available servers |
| File extension not in `extensions` | Server skipped for that file |
| `server.root(file)` returns `undefined` | Server skipped (no project root) |
| Server in `s.broken` set | Server skipped (previously failed) |
| Existing client at (root, serverId) | Cached client reused |
| Server spawn inflight | Wait for spawn to complete |
| Otherwise | New server process spawned |

---

## Key Files and Line Numbers

| File | Lines | Purpose |
|------|-------|---------|
| `src/lsp/server.ts` | 13-1168 | LSP server definitions |
| `src/lsp/index.ts` | 156-240 | Server selection logic |
| `src/config/config.ts` | 565-600 | Configuration schema |
| `src/lsp/language.ts` | 1-106 | Language extensions mapping |
| `src/lsp/client.ts` | 1-216 | LSP client implementation |
| `src/config/config.ts` | 24-94 | Config loading and merging |
