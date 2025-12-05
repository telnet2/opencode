# MCP Server Connection Analysis

This document provides a comprehensive analysis of how MCP (Model Context Protocol) server connections are supported in the OpenCode codebase, starting from `packages/opencode`.

## Table of Contents

1. [Entry Points](#1-entry-points)
2. [Configuration](#2-configuration)
3. [Connection Lifecycle](#3-connection-lifecycle)
4. [Protocol Handling](#4-protocol-handling)
5. [Tool Registration](#5-tool-registration)
6. [Error Handling](#6-error-handling)
7. [Key Files Summary](#7-key-files-summary)
8. [Dependencies](#8-dependencies)
9. [Data Flow Diagram](#9-data-flow-diagram)
10. [Security Considerations](#10-security-considerations)

---

## 1. Entry Points

### CLI Command Handler

**File**: `packages/opencode/src/cli/cmd/mcp.ts` (lines 1-81)

The MCP command is registered as a CLI subcommand in the main application at `packages/opencode/src/index.ts` (line 17).

**Key Handler**: `McpAddCommand` allows users to interactively add MCP servers:
- Prompts for server name
- Selects between "local" (run local command) or "remote" (connect to URL)
- For remote: validates URL and attempts connection test
- For local: captures command string

---

## 2. Configuration

### Schema Definition

**File**: `packages/opencode/src/config/config.ts` (lines 294-337)

OpenCode supports two configuration types using a Zod discriminated union:

### A. Local MCP Server (`McpLocal`)

```typescript
{
  type: "local",
  command: string[],              // Required - Command and arguments to execute
  environment: Record<string, string>,  // Optional - Environment variables
  enabled: boolean,               // Optional - Enable/disable on startup
  timeout: number                 // Optional, default: 5000ms - Tool fetching timeout
}
```

### B. Remote MCP Server (`McpRemote`)

```typescript
{
  type: "remote",
  url: string,                    // Required - URL endpoint of MCP server
  headers: Record<string, string>, // Optional - HTTP headers (for auth, etc.)
  enabled: boolean,               // Optional - Enable/disable on startup
  timeout: number                 // Optional, default: 5000ms - Tool fetching timeout
}
```

### Configuration Storage Locations

- **Global config**: `~/.opencode/opencode.json` or `opencode.jsonc`
- **Project config**: `opencode.json`/`opencode.jsonc` or `.opencode/opencode.json`
- **Config field**: `mcp: Record<string, Mcp>` (line 550 in config.ts)

### Example Configuration

```jsonc
{
  "mcp": {
    "filesystem": {
      "type": "local",
      "command": ["opencode", "x", "@modelcontextprotocol/server-filesystem"],
      "timeout": 5000
    },
    "remote-api": {
      "type": "remote",
      "url": "https://example.com/mcp",
      "headers": { "Authorization": "Bearer token" },
      "timeout": 10000
    }
  }
}
```

---

## 3. Connection Lifecycle

### Lifecycle Management

**File**: `packages/opencode/src/mcp/index.ts` (lines 56-91)

The MCP module uses `Instance.state()` (a key-value state management system) to manage the lifecycle:

### Initialization Phase (lines 56-79)

```typescript
1. On first call: Load config via Config.get()
2. Extract mcp config object (cfg.mcp ?? {})
3. For each MCP server config:
   - Call create(key, mcp) function
   - If successful: store client in clients{} and status in status{}
4. Return state object: { status, clients }
```

### Maintenance Phase

- Clients remain in memory and are reused across requests
- Status tracked per server (connected/disabled/failed)
- Tools cached within client instances

### Cleanup/Disposal Phase (lines 80-90)

- Registered cleanup function called on `Instance.dispose()`
- Closes all active clients via `client.close()`
- Errors logged but don't block disposal
- Prevents hanging subprocess connections (especially important for Docker containers)

### Connection State Schema

**File**: `packages/opencode/src/mcp/index.ts` (lines 25-53)

```typescript
Status = discriminatedUnion("status", [
  { status: "connected" },
  { status: "disabled" },
  { status: "failed", error: string }
])
```

---

## 4. Protocol Handling

### Transport Layer Implementations

**File**: `packages/opencode/src/mcp/index.ts` (lines 129-210)

### For Remote Servers (lines 129-175)

Two transport implementations tried in sequence:

1. **StreamableHTTPClientTransport** (lines 131-137)
   - URL-based connection
   - Headers passed via `requestInit`
   - Attempts bidirectional streaming over HTTP

2. **SSEClientTransport** (lines 139-145)
   - Server-Sent Events fallback
   - Same header support
   - Used if StreamableHTTP fails

**Error Handling**: If both transports fail, the last error is captured and returned as failed status.

### For Local Servers (lines 178-210)

**StdioClientTransport** (lines 182-191):
- Spawns subprocess with specified command
- stderr: "ignore" - suppresses subprocess errors
- Environment variables merged with `process.env`
- Special handling: sets `BUN_BE_BUN=1` for "opencode" command
- Custom environment variables from config applied

### MCP Client Wrapper

**File**: `packages/opencode/src/mcp/index.ts` (line 2)

Uses `experimental_createMCPClient` from `@ai-sdk/mcp` library to wrap transport layers. This provides:
- Protocol message marshalling/unmarshalling
- Tool discovery and invocation
- Resource management

---

## 5. Tool Registration

### Tool Discovery and Registration

**File**: `packages/opencode/src/mcp/index.ts` (lines 264-288)

**MCP.tools()** function:
1. Gets all MCP clients from state
2. For each client, calls `client.tools()` to fetch available tools
3. **Tool Naming Convention**: Sanitizes tool names by replacing non-alphanumeric characters:
   - Format: `{sanitized_client_name}_{sanitized_tool_name}`
   - Example: "filesystem_read_file", "remote_api_get_user"
4. Returns `Record<string, Tool>` for AI SDK consumption

### Tool Name Sanitization

**Lines 282-284** in `mcp/index.ts`:

```typescript
const sanitizedClientName = clientName.replace(/[^a-zA-Z0-9_-]/g, "_")
const sanitizedToolName = toolName.replace(/[^a-zA-Z0-9_-]/g, "_")
result[sanitizedClientName + "_" + sanitizedToolName] = tool
```

### Integration into Session Processing

**File**: `packages/opencode/src/session/prompt.ts` (lines 727-789)

In the `resolveTools()` function:

1. **Retrieval** (line 727): `for (const [key, item] of Object.entries(await MCP.tools()))`
2. **Filtering** (line 728): Applied against enabledTools using Wildcard matching
3. **Tool Wrapping** (lines 731-787):
   - Wraps original execute function
   - Triggers plugin hooks: `tool.execute.before` and `tool.execute.after`
   - Handles result content processing:
     - Text content extracted to output string
     - Image content converted to FilePart attachments with base64 encoding
   - Sets up tool output formatter (lines 782-787)
4. **Registration** (line 788): Added to tools dictionary with original AI SDK Tool interface

### Server API Endpoints

**File**: `packages/opencode/src/server/server.ts` (lines 1577-1625)

#### GET /mcp (Status Endpoint, lines 1577-1595)
- Returns status of all configured MCP servers
- Response: `Record<string, Status>`

#### POST /mcp (Add Endpoint, lines 1597-1625)
- Dynamically add new MCP server at runtime
- Request body:
  ```json
  {
    "name": "server-name",
    "config": { "type": "local" | "remote", ... }
  }
  ```
- Response: Updated status record

---

## 6. Error Handling

### Error Types and Handling

#### A. Failed Status Tracking (lines 120-240)

- Each server gets separate status tracking
- On failure: `{ status: "failed", error: "error message" }`
- Errors captured for:
  - Connection failures (both transport types)
  - Tool fetching timeouts
  - Subprocess startup failures
  - Unknown errors

#### B. Timeout Handling (line 226)

**File**: `packages/opencode/src/mcp/index.ts`

Uses `withTimeout()` utility (from `packages/opencode/src/util/timeout.ts`):

```typescript
const result = await withTimeout(mcpClient.tools(), mcp.timeout ?? 5000)
```

- Default: 5000ms timeout
- If exceeded: Operation timed out error
- Caught and status set to failed with error message

#### C. Client Closure on Tool Fetch Failure (lines 231-246)

- If tool fetching fails after successful connection
- Client is immediately closed
- Status marked as failed: "Failed to get tools"
- Prevents hanging connections

#### D. Plugin Hook Exception Handling (lines 732-752)

- Before/after hooks wrapped in plugin trigger
- Any plugin hook errors don't break tool execution
- Errors logged per server

#### E. Error Formatting for CLI

**File**: `packages/opencode/src/cli/error.ts` (lines 8-9)

`MCP.Failed` error detected and formatted as:
> "MCP server "{name}" failed. Note, opencode does not support MCP authentication yet."

#### F. Disposal Error Handling (lines 82-87)

- Individual `client.close()` errors logged but don't prevent other clients from closing
- Graceful degradation

---

## 7. Key Files Summary

| File Path | Lines | Role |
|-----------|-------|------|
| `src/mcp/index.ts` | Full | Core MCP module - client creation, connection lifecycle, tool fetching |
| `src/config/config.ts` | 294-550 | MCP schema definitions (McpLocal, McpRemote, Mcp union type) and config loading |
| `src/cli/cmd/mcp.ts` | Full | CLI interface for adding MCP servers interactively |
| `src/session/prompt.ts` | 727-789 | Tool registration in AI SDK, wrapping MCP tools with plugin hooks |
| `src/server/server.ts` | 1577-1625 | HTTP API endpoints for MCP status and dynamic registration |
| `src/acp/agent.ts` | 480-518 | ACP integration: configures MCP servers during session init |
| `src/project/instance.ts` | Full | Instance state management - handles MCP client lifecycle per project |
| `src/project/state.ts` | Full | Underlying state storage and disposal mechanism |
| `src/util/timeout.ts` | Full | Timeout wrapper for tool fetching operations |
| `src/cli/error.ts` | 8-9 | Error formatting for MCP failures |

---

## 8. Dependencies

**MCP-Related NPM Packages** (from package.json):
- `@ai-sdk/mcp@0.0.8` - AI SDK provider for MCP integration
- `@modelcontextprotocol/sdk@1.15.1` - Official MCP SDK with transport implementations

---

## 9. Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    opencode.json/opencode.jsonc                 │
│                    (mcp config section)                          │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ▼
        ┌────────────────┐
        │  Config.get()  │
        └────────┬───────┘
                 │
                 ▼
    ┌────────────────────────────────┐
    │  MCP.state() initialization    │
    │  (Instance.state)              │
    └───────┬────────────┬───────────┘
            │            │
    ┌───────▼──┐  ┌──────▼──────┐
    │ Remote   │  │ Local       │
    │ Servers  │  │ Servers     │
    └─┬────┬───┘  └──┬────┬─────┘
      │    │         │    │
   ┌──▼┐ ┌─▼──┐  ┌───▼┐ ┌──▼───┐
   │HTP│ │SSE │  │Cmd │ │Env   │
   │   │ │    │  │Std │ │Vars  │
   └──┬┘ └─┬──┘  └───┬┘ └──┬───┘
      │    │         │    │
      └────┴─┬──┬────┴────┘
             │  │
             ▼  ▼
    ┌─────────────────────────────────┐
    │ experimental_createMCPClient    │
    │ (@ai-sdk/mcp)                  │
    └────────┬──────────────┬─────────┘
             │              │
             ▼              ▼
      ┌────────────┐  ┌─────────────┐
      │ Connected  │  │ Failed      │
      │ Clients    │  │ Status      │
      └────┬───────┘  └─────────────┘
           │
           ▼
    ┌─────────────────────────────────┐
    │  MCP.tools()                    │
    │  - Fetch tools per client       │
    │  - Sanitize names              │
    │  - Timeout enforcement         │
    └────────┬──────────────┬─────────┘
             │              │
             ▼              ▼
      ┌────────────┐  ┌────────────┐
      │ AI SDK     │  │ Wrapped    │
      │ Tool{}     │  │ Execute    │
      └────┬───────┘  └─────────────┘
           │
           ▼
    ┌─────────────────────────────────┐
    │ resolveTools() in prompt.ts      │
    │ - Plugin hook wrapping          │
    │ - Tool filtering                │
    │ - Result processing             │
    └────────┬──────────────┬─────────┘
             │              │
             ▼              ▼
      ┌───────────────┐  ┌──────────────┐
      │ Available     │  │ Agent Model  │
      │ Tools{}       │  │ Tool Calls   │
      └───────────────┘  └──────────────┘
```

---

## 10. Security Considerations

1. **No Built-in MCP Authentication**: Error message explicitly states this (`cli/error.ts` line 9)

2. **Custom Header Support**: Remote servers can pass headers for custom auth, but handled at transport layer

3. **Permission System**: Agent-level permissions (edit, bash, webfetch) respected; MCP tools inherit these

4. **Subprocess Isolation**: Local servers run in subprocess with configurable environment variables

5. **Timeout Protection**: Default 5-second timeout prevents hanging connections from blocking the system

---

## Complete Configuration Flow

```
1. User/Config → opencode.json mcp field
                ↓
2. Config Loading → Config.get() merges all config sources
                ↓
3. Instance Initialization → Instance.state() called
                ↓
4. MCP Client Creation → For each config entry:
                        a) Validate config schema
                        b) Select transport based on type
                        c) Create MCP client via @ai-sdk/mcp
                        d) Fetch tools with timeout
                        e) Store status and client reference
                ↓
5. Tool Resolution → SessionPrompt.resolveTools():
                    a) Call MCP.tools()
                    b) Wrap each tool with plugin hooks
                    c) Add to session's available tools
                ↓
6. Tool Execution → AI model calls tool
                   → Original execute wrapped function called
                   → Result formatted and returned
                ↓
7. Cleanup → Instance.dispose():
            a) Close all MCP clients
            b) Terminate subprocesses
            c) Log any errors
```
