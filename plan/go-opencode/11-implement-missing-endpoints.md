# Implement Missing SDK Endpoints - Feature Parity Plan

## Overview

This document tracks the implementation of missing endpoints to achieve full feature parity between the Go server and the TypeScript/Python/Go SDKs. The goal is to expose all Go server capabilities through the SDKs.

## Implementation Phases

| Phase | Focus | Status |
|-------|-------|--------|
| Phase 1 | Initial SDK alignment | ✅ COMPLETE |
| Phase 2 | Core SDK endpoints (project, message, TUI control) | ✅ COMPLETE |
| Phase 3 | Feature parity (MCP, commands, formatters, sharing) | ✅ COMPLETE |
| Phase 4 | SDK expansion (expose Go server advanced features) | ✅ COMPLETE |

---

## Phase 1: Initial SDK Alignment (COMPLETE)

**Completed:** 2025-11-27

Initial alignment of Go server types with SDK contract:
- Aligned response types with SDK expectations
- Fixed boolean return types for write operations
- Ensured consistent JSON field naming

---

## Phase 2: Core SDK Endpoints (COMPLETE)

**Completed:** 2025-11-27

**Commit:** `8920bfd` - feat(api): implement missing SDK endpoints for Phase 2 compatibility

### Endpoints Implemented:
| Method | Path | Description |
|--------|------|-------------|
| GET | `/project` | List all projects |
| GET | `/project/current` | Get current project for directory |
| GET | `/session/{id}/message/{messageID}` | Get single message by ID |
| GET | `/tui/control/next` | Get next TUI control request |
| POST | `/tui/control/response` | Submit TUI control response |

### Types Added:
- `Project` struct with ID, Worktree, VCS, Time fields
- `ProjectTime` struct with Created/Initialized timestamps
- `TUIControlRequest` struct for control queue

---

## Phase 3: Feature Parity (COMPLETE)

**Completed:** 2025-11-27

**Commit:** `244bcfb` - feat(api): implement Phase 3 feature parity - MCP, commands, formatters, sharing

### MCP (Model Context Protocol):
| Method | Path | Description |
|--------|------|-------------|
| GET | `/mcp` | List MCP servers |
| POST | `/mcp` | Add MCP server |
| DELETE | `/mcp/{name}` | Remove MCP server |
| GET | `/mcp/tools` | List MCP server tools |
| POST | `/mcp/tool/{name}` | Execute MCP tool |
| GET | `/mcp/resources` | List MCP resources |
| GET | `/mcp/resource` | Read MCP resource content |

### Custom Commands:
| Method | Path | Description |
|--------|------|-------------|
| GET | `/command` | List available commands |
| GET | `/command/{name}` | Get command details |
| POST | `/command/{name}` | Execute command |

### Formatter:
| Method | Path | Description |
|--------|------|-------------|
| GET | `/formatter` | List formatters |
| POST | `/formatter/format` | Format file |

### Session Sharing:
| Method | Path | Description |
|--------|------|-------------|
| POST | `/session/{id}/share` | Create share link |
| DELETE | `/session/{id}/share` | Remove share |

### Components Added:
- `internal/command/executor.go` - Command execution with templates
- `internal/formatter/manager.go` - Formatter management
- `internal/sharing/token.go` - Token-based sharing

---

## Phase 4: SDK Expansion (COMPLETE)

**Completed:** 2025-11-27

This phase exposes advanced Go server features to the SDKs by updating the stainless configuration.

### New SDK Resources

#### 1. Client Tools Management (4 endpoints)

External tool registration and execution system for client-side tool integration.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/client-tools/register` | Register external client tools |
| POST | `/client-tools/execute` | Execute registered client tools |
| POST | `/client-tools/result` | Submit results from client tools |
| DELETE | `/client-tools/unregister` | Unregister client tools |

**Use Case:** Allows external applications to register custom tools that can be invoked by the AI assistant.

#### 2. MCP Extended Methods (5 endpoints)

Extended MCP management capabilities beyond basic server listing.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mcp/tools` | List all tools from MCP servers |
| GET | `/mcp/resources` | List all resources from MCP servers |
| GET | `/mcp/resource` | Read specific MCP resource content |
| POST | `/mcp/tool/{name}` | Execute a specific MCP tool |
| DELETE | `/mcp/{name}` | Remove an MCP server |

#### 3. Command Extended Methods (2 endpoints)

Command retrieval and execution capabilities.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/command/{name}` | Get specific command details |
| POST | `/command/{name}` | Execute a specific command |

#### 4. Formatter Extended Method (1 endpoint)

File formatting capabilities.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/formatter/format` | Format a file with specified formatter |

#### 5. Documentation Endpoint (1 endpoint)

API documentation access.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/doc` | OpenAPI documentation |

### Implementation Details

#### stainless.yml Updates

Added to `packages/sdk/stainless/stainless.yml`:

```yaml
resources:
  client_tools:
    name: ClientTools
    methods:
      register: post /client-tools/register
      execute: post /client-tools/execute
      result: post /client-tools/result
      unregister: delete /client-tools/unregister

  mcp:
    # Extended methods
    methods:
      list: get /mcp
      add: post /mcp
      remove: delete /mcp/{name}
      list_tools: get /mcp/tools
      list_resources: get /mcp/resources
      read_resource: get /mcp/resource
      execute_tool: post /mcp/tool/{name}

  command:
    # Extended methods
    methods:
      list: get /command
      get: get /command/{name}
      execute: post /command/{name}

  formatter:
    # Extended methods
    methods:
      list: get /formatter
      format: post /formatter/format
```

#### Test Coverage

Added tests in `go-opencode/internal/server/citest/phase4_test.go`:
- Client tools registration and execution
- MCP tool and resource operations
- Command retrieval and execution
- File formatting

---

## Endpoint Summary

### Total Endpoints by Phase

| Phase | New Endpoints | Cumulative |
|-------|---------------|------------|
| Phase 1 | 0 (alignment) | 64 |
| Phase 2 | 5 | 69 |
| Phase 3 | 14 | 77* |
| Phase 4 | 0 (SDK exposure) | 77 |

*Some endpoints overlap between SDK and Go server implementations

### Final SDK Coverage

After Phase 4, the SDKs expose **77 endpoints** covering:

- **Session Management:** Create, list, update, delete, fork, abort sessions
- **Message Handling:** Send, receive, stream messages
- **Tool System:** Register, execute, manage tools
- **File Operations:** Read, write, edit, search files
- **Project Management:** List projects, get current project
- **Configuration:** Get/update config, providers, auth
- **MCP Integration:** Full MCP server and tool management
- **Commands:** Custom command execution
- **Formatters:** File formatting
- **TUI Control:** Bidirectional TUI communication
- **Client Tools:** External tool registration and execution
- **Sharing:** Session sharing with tokens
- **Events:** SSE event streaming

---

## Files Modified

### Phase 2
- `go-opencode/internal/server/handlers_message.go`
- `go-opencode/internal/server/handlers_project.go`
- `go-opencode/internal/server/handlers_tui.go`
- `go-opencode/internal/server/routes.go`
- `go-opencode/internal/session/service.go`
- `go-opencode/pkg/types/session.go`

### Phase 3
- `go-opencode/cmd/opencode-server/main.go`
- `go-opencode/internal/command/executor.go` (new)
- `go-opencode/internal/formatter/manager.go` (new)
- `go-opencode/internal/sharing/token.go` (new)
- `go-opencode/internal/server/handlers_config.go`
- `go-opencode/internal/server/routes.go`
- `go-opencode/internal/server/server.go`

### Phase 4
- `packages/sdk/stainless/stainless.yml`
- `go-opencode/internal/server/citest/phase4_test.go` (new)

---

## Verification

### Running Tests

```bash
# Run all endpoint tests
cd go-opencode
go test ./internal/server/citest/... -v

# Run specific phase tests
go test ./internal/server/citest/... -v -run "Phase4"
```

### SDK Regeneration

```bash
# After updating stainless.yml
cd packages/sdk/stainless
npm run generate

# Verify generated SDKs
cd ../typescript && npm run build
cd ../python && pip install -e . && python -c "import opencode_sdk"
cd ../go && go build ./...
```

---

## Success Criteria

- [x] All 64 original SDK endpoints implemented in Go server
- [x] Phase 2: Project and TUI control endpoints working
- [x] Phase 3: MCP, commands, formatters, sharing working
- [x] Phase 4: SDK updated to expose all Go server features
- [x] All tests passing
- [x] Documentation updated
