# LSP (Language Server Protocol) Utilization Analysis

This document provides a comprehensive analysis of how OpenCode utilizes LSP (Language Server Protocol) for enhanced code intelligence.

## Table of Contents

1. [Overview](#1-overview)
2. [LSP Core Integration](#2-lsp-core-integration)
3. [LSP Client Implementation](#3-lsp-client-implementation)
4. [Supported Language Servers](#4-supported-language-servers)
5. [LSP Data Usage](#5-lsp-data-usage)
6. [LSP Tools](#6-lsp-tools)
7. [LSP Lifecycle Management](#7-lsp-lifecycle-management)
8. [Event System](#8-event-system)
9. [Dependencies](#9-dependencies)

---

## 1. Overview

OpenCode has a comprehensive LSP integration that provides:

- **25+ language servers** with automatic binary downloads
- **Diagnostics** automatically injected into edit tool context
- **Hover information** available for type inspection
- **Symbol search** for workspace and document symbols
- **On-demand server spawning** per file extension
- **Configurable per server** with custom commands and initialization options

---

## 2. LSP Core Integration

### Main API Location

**File**: `packages/opencode/src/lsp/index.ts` (lines 1-370)

The LSP namespace provides the primary interface for all LSP functionality.

### Key Features Exposed

| Function | Lines | Description |
|----------|-------|-------------|
| `LSP.init()` | 125-127 | Initializes LSP state via `Instance.state()` |
| `LSP.diagnostics()` | 256-266 | Aggregates diagnostics from all language servers |
| `LSP.hover()` | 268-280 | Sends `textDocument/hover` requests |
| `LSP.workspaceSymbol()` | 322-332 | Searches for symbols across workspace |
| `LSP.documentSymbol()` | 334-346 | Gets symbols within a specific file |
| `LSP.touchFile()` | 242-254 | Opens/updates files and optionally waits for diagnostics |
| `LSP.Diagnostic.pretty()` | 354-369 | Formats diagnostics with severity levels |

### Workspace Symbol Filtering

**Lines 322-332**: Workspace symbols are filtered to specific kinds:
- Classes
- Functions
- Methods
- Interfaces
- Variables
- Constants
- Structs
- Enums

---

## 3. LSP Client Implementation

### Transport Mechanism

**File**: `packages/opencode/src/lsp/client.ts` (lines 1-215)

Uses **stdio-based** transport (lines 41-44):

```typescript
createMessageConnection(
  new StreamMessageReader(input.server.process.stdout),
  new StreamMessageWriter(input.server.process.stdin)
)
```

Uses `vscode-jsonrpc` for JSON-RPC communication with spawned language server processes.

### LSP Client Initialization

**Lines 76-116**: Sends LSP `initialize` request with:

- Root URI and workspace folders
- Process ID
- Capabilities:
  - Window: `workDoneProgress: true`
  - Workspace: `configuration: true`
  - TextDocument: `didOpen`, `didChange`, `publishDiagnostics`
- 5-second timeout (line 107)
- Followed by `initialized` notification (line 118)

### Notification Handling

**Diagnostics Publishing** (lines 47-56):
- Listens for `textDocument/publishDiagnostics` notifications
- Tracks diagnostics by file path
- Publishes `LSPClient.Event.Diagnostics` event

**Window/Workspace Requests** (lines 57-72):
- `window/workDoneProgress/create` → returns null
- `workspace/configuration` → returns initialization options
- `client/registerCapability` and `unregisterCapability` → empty handlers
- `workspace/workspaceFolders` → returns workspace folder info

### File Management

**Lines 138-176**:

```typescript
notify.open(file: string, text: string)
```

- Opens or updates files with language ID mapping via `LANGUAGE_EXTENSIONS`
- Tracks file versions to distinguish between `didOpen` and `didChange`
- Clears cached diagnostics on first open (line 165)

### Diagnostics Waiting

**Lines 181-201**:

```typescript
waitForDiagnostics(file: string)
```

- 3-second timeout
- Subscribes to `LSPClient.Event.Diagnostics` bus events

### Lifecycle

**Lines 202-208**:

```typescript
shutdown()
```

- Calls `connection.end()`
- Calls `connection.dispose()`
- Calls `process.kill()`

---

## 4. Supported Language Servers

**File**: `packages/opencode/src/lsp/server.ts` (lines 1-1168)

OpenCode supports 19 built-in language servers:

| Language Server | ID | Extensions | Root Finder | Auto-Install |
|---|---|---|---|---|
| **Deno** | `deno` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs` | `deno.json`/`deno.jsonc` | No |
| **TypeScript** | `typescript` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`, `.mts`, `.cts` | Lockfiles (excludes deno) | Yes (npm) |
| **Vue** | `vue` | `.vue` | Lockfiles | Yes (npm) |
| **ESLint** | `eslint` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mts`, `.cts`, `.vue` | Lockfiles | Yes (VS Code server) |
| **Go (gopls)** | `gopls` | `.go` | `go.work`, `go.mod`/`go.sum` | Yes (`go install`) |
| **Ruby** | `ruby-lsp` | `.rb`, `.rake`, `.gemspec`, `.ru` | `Gemfile` | Yes (`gem install`) |
| **Python (Pyright)** | `pyright` | `.py`, `.pyi` | `pyproject.toml`, `requirements.txt`, etc. | Yes (npm) |
| **Elixir** | `elixir-ls` | `.ex`, `.exs` | `mix.exs`, `mix.lock` | Yes (GitHub) |
| **Zig** | `zls` | `.zig`, `.zon` | `build.zig` | Yes (GitHub) |
| **C#** | `csharp` | `.cs` | `.sln`, `.csproj`, `global.json` | Yes (`dotnet tool`) |
| **Swift** | `sourcekit-lsp` | `.swift`, `.objc`, `.objcpp` | `Package.swift`, xcodeproj | No |
| **Rust** | `rust` | `.rs` | `Cargo.toml`/`Cargo.lock` | No |
| **Clang (C++)** | `clangd` | `.c`, `.cpp`, `.cc`, `.h`, `.hpp` | `compile_commands.json`, `CMakeLists.txt` | Yes (GitHub) |
| **Svelte** | `svelte` | `.svelte` | Lockfiles | Yes (npm) |
| **Astro** | `astro` | `.astro` | Lockfiles | Yes (npm) |
| **Java (JDTLS)** | `jdtls` | `.java` | `pom.xml`, `build.gradle` | Yes (Eclipse) |
| **YAML** | `yaml-ls` | `.yaml`, `.yml` | Lockfiles | Yes (npm) |
| **Lua** | `lua-ls` | `.lua` | `.luarc.json`, `.luacheckrc` | Yes (GitHub) |
| **PHP** | `php intelephense` | `.php` | `composer.json` | Yes (npm) |

### Root Finding Strategy

**Lines 23-45**: `NearestRoot()` function:

- Searches up directory tree for specific markers
- Supports `excludePatterns` to skip certain paths
- Falls back to instance directory if no markers found

### Auto-Download Capability

Respects `Flag.OPENCODE_DISABLE_LSP_DOWNLOAD` to disable automatic downloads.

---

## 5. LSP Data Usage

### In Edit Tool

**File**: `packages/opencode/src/tool/edit.ts` (lines 139-150)

After file edits, diagnostics are automatically fetched and displayed:

```typescript
await LSP.touchFile(filePath, true)  // Wait for diagnostics
const diagnostics = await LSP.diagnostics()
// Filter for errors (severity 1) and display to model
issues.filter((item) => item.severity === 1).map(LSP.Diagnostic.pretty)
```

This provides immediate feedback on syntax errors and type issues after edits.

### In Prompt Generation

**File**: `packages/opencode/src/session/prompt.ts` (lines 862-880)

When file ranges from workspace symbol searches are incomplete:

```typescript
const symbols = await LSP.documentSymbol(filePathURI)
// Matches symbol line numbers to refine start/end positions
// Uses range data to calculate file offset and limit for Read tool
```

### Symbol Source Tracking

**File**: `packages/opencode/src/session/message-v2.ts` (lines 99-118)

Symbols are tracked with source metadata:

```typescript
SymbolSource = z.object({
  path: z.string(),           // File path
  range: LSP.Range,           // Start/end line, character
  name: z.string(),           // Symbol name
  kind: z.number(),           // LSP symbol kind
})
```

---

## 6. LSP Tools

### Diagnostics Tool

**File**: `packages/opencode/src/tool/lsp-diagnostics.ts` (lines 1-26)

- **Tool ID**: `lsp_diagnostics`
- **Parameters**: `path` (string)
- **Execution**: Touches file, waits for diagnostics, returns formatted errors

### Hover Tool

**File**: `packages/opencode/src/tool/lsp-hover.ts` (lines 1-31)

- **Tool ID**: `lsp_hover`
- **Parameters**: `file`, `line`, `character` (numbers)
- **Execution**: Touches file, sends hover request, returns JSON response

**Note**: Both tools are marked "do not use" - not currently exposed to models directly.

### Debug Commands

**File**: `packages/opencode/src/cli/cmd/debug/lsp.ts` (lines 1-47)

Available CLI commands for debugging:

```bash
opencode debug lsp diagnostics <file>
opencode debug lsp symbols <query>
opencode debug lsp document-symbols <uri>
```

---

## 7. LSP Lifecycle Management

### Initialization

**File**: `packages/opencode/src/project/bootstrap.ts` (line 21)

```typescript
await LSP.init()  // Called during instance bootstrap
```

### Per-File Activation

**Lines 156-240** in `lsp/index.ts`:

The `getClients(file)` function:

1. Determines which servers handle a file by extension
2. Spawns servers on-demand based on file extension match
3. Caches spawned clients to avoid duplication
4. Tracks "broken" servers to avoid repeated spawn attempts
5. Uses inflight promises to deduplicate simultaneous spawn requests

### Configuration Loading

LSP configuration is loaded from:

- Config files (`opencode.jsonc`/`opencode.json`)
- Environment variable `OPENCODE_CONFIG`
- `Flag.OPENCODE_CONFIG_CONTENT` for inline config

Configuration can disable LSP globally or per-server:

```json
{
  "lsp": {
    "typescript": {
      "disabled": true,
      "command": ["custom-ts-lsp"],
      "env": { ... },
      "extensions": [".ts", ".tsx"],
      "initialization": { ... }
    }
  }
}
```

### Shutdown

**Lines 120-122** in `lsp/index.ts`:

- Triggered during instance cleanup
- Calls `client.shutdown()` on all active clients
- Closes connections and kills processes

---

## 8. Event System

### LSP Events

**File**: `packages/opencode/src/lsp/index.ts` (lines 14-16)

```typescript
Event.Updated: Bus.event("lsp.updated", {})
// Fired when new clients connect
```

### Client Events

**File**: `packages/opencode/src/lsp/client.ts` (lines 27-35)

```typescript
Event.Diagnostics: Bus.event("lsp.client.diagnostics", {
  serverID: string,
  path: string
})
```

---

## 9. Dependencies

**File**: `packages/opencode/package.json`

```json
{
  "devDependencies": {
    "vscode-languageserver-types": "3.17.5"  // LSP type definitions
  },
  "dependencies": {
    "vscode-jsonrpc": "8.2.1"  // JSON-RPC transport
  }
}
```

---

## Language-to-Extension Mapping

**File**: `packages/opencode/src/lsp/language.ts` (lines 1-106)

Maps 100+ file extensions to LSP language IDs. Used by:

- LSP client to determine language ID when opening files
- Session UI for syntax highlighting

---

## Summary

OpenCode's LSP integration is **comprehensive and modern**:

| Feature | Implementation |
|---------|----------------|
| **Transport** | Stdio-based communication |
| **Servers** | 25+ language servers with auto-download |
| **Spawning** | On-demand per file extension |
| **Diagnostics** | Automatically injected into edit tool context |
| **Symbols** | Hover and symbol information for code navigation |
| **Configuration** | Per server with custom commands and init options |
| **Error Handling** | Timeouts, broken server tracking |
| **Events** | Bus-based event system for diagnostics updates |
