# OpenCode Server Go Rewrite: Feasibility Evaluation & Implementation Plan

## Executive Summary

**Feasibility: HIGH** - The rewrite is technically feasible and strategically sound.

The OpenCode server can be rewritten in Go while maintaining full compatibility with the existing TUI client. The protocol (REST + SSE) is well-documented and standard. Existing Go code in the repository (go-memsh, OpenCode SDK) demonstrates the team's Go proficiency and provides reusable patterns.

**Estimated Effort: 8-12 weeks** for a production-ready implementation with a small team (2-3 developers).

---

## 1. Current Architecture Analysis

### Server Stack (TypeScript/Bun)
| Component | Technology | Lines of Code | Complexity |
|-----------|------------|---------------|------------|
| HTTP Server | Hono.js + Bun.serve | ~2,000 | Medium |
| LLM Integration | Vercel AI SDK | ~1,500 | High |
| Tool System | Custom + Zod | ~3,500 | High |
| Session Management | Custom | ~2,000 | Medium |
| Storage | File-based JSON | ~500 | Low |
| Permission System | Custom | ~400 | Medium |
| LSP Integration | Custom | ~600 | Medium |
| MCP Support | @modelcontextprotocol/sdk | ~300 | Medium |
| **Total** | | **~10,800** | |

### Protocol (TUI ↔ Server)
- **Transport**: HTTP REST + Server-Sent Events (SSE)
- **Format**: JSON
- **Endpoints**: 60+ REST endpoints
- **Streaming**: SSE for real-time events (`/event`, `/global/event`)
- **Authentication**: API key via headers

### Existing Go Assets
1. **go-memsh**: Complete shell interpreter with HTTP API + WebSocket JSON-RPC
2. **OpenCode SDK for Go**: Comprehensive client SDK (~89KB session.go)
3. Both demonstrate Go patterns for similar problems

---

## 2. Feasibility Assessment

### ✅ Strong Feasibility Factors

| Factor | Assessment |
|--------|------------|
| **Protocol Stability** | REST + SSE is standard; Go has excellent HTTP/SSE support |
| **Existing Go Code** | go-memsh and SDK provide patterns and reusable code |
| **LLM SDK Availability** | Go SDKs exist for all major providers (Anthropic, OpenAI, Google, etc.) |
| **Tool System** | Straightforward to port; Go has good process management |
| **Storage Layer** | Simple file-based JSON; trivial in Go |
| **Team Experience** | Codebase shows strong Go proficiency |

### ⚠️ Challenges to Address

| Challenge | Mitigation |
|-----------|------------|
| **Vercel AI SDK abstraction** | Build thin provider abstraction; each provider has native Go SDK |
| **Zod → Go validation** | Use go-playground/validator or custom validation |
| **TypeScript type inference** | Define explicit Go structs (more verbose but clearer) |
| **Hot module loading** | Use Go plugins or compile-time registration |
| **Tree-sitter bash parsing** | Use go-tree-sitter bindings or shell parser libraries |

### 🚫 Non-Issues

| Concern | Why It's Not a Problem |
|---------|------------------------|
| Protocol changes | None required; TUI client unchanged |
| Performance | Go typically faster than Bun for I/O-heavy workloads |
| Concurrency | Go's goroutines ideal for streaming + tool execution |
| Deployment | Single binary simplifies distribution |

---

## 3. Benefits of Go Rewrite

### Performance & Resource Efficiency
- **Lower memory footprint**: Go typically uses 3-5x less memory than Node.js/Bun
- **Faster startup**: Single binary, no runtime initialization
- **Better concurrency**: Native goroutines vs JavaScript event loop
- **Predictable latency**: No GC pauses like V8

### Operational Benefits
- **Single binary deployment**: No npm install, no node_modules
- **Cross-compilation**: Easy builds for all platforms
- **Static typing**: Catch errors at compile time
- **Smaller container images**: ~20MB vs ~200MB+ for Node.js

### Developer Experience
- **Simpler debugging**: Standard tooling, no transpilation
- **Better IDE support**: gopls is excellent
- **Consistent formatting**: gofmt eliminates style debates

---

## 4. Phased Implementation Plan

### Phase 1: Foundation (Weeks 1-2)

#### 1.1 Project Structure & Core Types
```
opencode-server/
├── cmd/
│   └── opencode-server/
│       └── main.go
├── internal/
│   ├── server/          # HTTP server + routes
│   ├── session/         # Session management
│   ├── message/         # Message types + storage
│   ├── provider/        # LLM provider abstraction
│   ├── tool/            # Tool system
│   ├── permission/      # Permission checking
│   ├── storage/         # File-based storage
│   ├── config/          # Configuration loading
│   └── event/           # Event bus + SSE
├── pkg/
│   └── types/           # Shared types (exported)
├── go.mod
└── go.sum
```

#### 1.2 Core Types (Port from TypeScript)
- [ ] Message types (User, Assistant, Parts)
- [ ] Session types
- [ ] Config schema
- [ ] Provider/Model types
- [ ] Tool definition types
- [ ] Permission types

#### 1.3 Storage Layer
- [ ] File-based JSON storage (matching existing format)
- [ ] Session CRUD operations
- [ ] Message CRUD operations
- [ ] Part management
- [ ] File locking for concurrent access

#### 1.4 Event System
- [ ] In-memory event bus
- [ ] SSE streaming implementation
- [ ] Event types matching TypeScript

**Deliverable**: Core types, storage layer, event bus - all unit tested

---

### Phase 2: HTTP Server & Basic Endpoints (Weeks 3-4)

#### 2.1 HTTP Server Setup
- [ ] Chi or Gin router setup
- [ ] Middleware (CORS, logging, error handling)
- [ ] OpenAPI documentation (go-swagger or oapi-codegen)

#### 2.2 Session Endpoints
```go
// Must match existing API exactly
GET    /session              // List sessions
POST   /session              // Create session
GET    /session/:id          // Get session
PATCH  /session/:id          // Update session
DELETE /session/:id          // Delete session
POST   /session/:id/abort    // Abort session
POST   /session/:id/fork     // Fork session
POST   /session/:id/revert   // Revert message
```

#### 2.3 File Endpoints
```go
GET    /file                 // List directory
GET    /file/content         // Read file
GET    /file/status          // Git status
GET    /find                 // Grep search
GET    /find/file            // File search
```

#### 2.4 Config Endpoints
```go
GET    /config               // Get config
GET    /config/providers     // List providers
GET    /provider             // List available providers
GET    /path                 // Get paths
```

#### 2.5 Event Streaming
```go
GET    /event                // Session SSE stream
GET    /global/event         // Global SSE stream
```

**Deliverable**: Working HTTP server with session/file/config endpoints

---

### Phase 3: LLM Provider Integration (Weeks 5-6)

#### 3.1 Provider Abstraction Layer
```go
type Provider interface {
    ID() string
    Name() string
    Models() []Model
    CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionStream, error)
}

type CompletionStream interface {
    Next() (StreamEvent, error)
    Close() error
}
```

#### 3.2 Provider Implementations
Priority order based on usage:
- [ ] **Anthropic** (anthropic-go SDK)
- [ ] **OpenAI** (openai-go SDK)
- [ ] **Google Gemini** (google.golang.org/genai)
- [ ] **OpenRouter** (OpenAI-compatible)
- [ ] **Azure OpenAI** (azure-sdk-for-go)
- [ ] **Amazon Bedrock** (aws-sdk-go-v2)

#### 3.3 Streaming Implementation
- [ ] Delta text streaming
- [ ] Tool call streaming
- [ ] Reasoning/thinking streaming
- [ ] Token counting + cost calculation
- [ ] Error handling + retries

#### 3.4 Provider-Specific Transformations
- [ ] Message format normalization per provider
- [ ] Cache control headers (Anthropic)
- [ ] Temperature defaults per model
- [ ] Provider options mapping

**Deliverable**: Working LLM completions with streaming for top 3 providers

---

### Phase 4: Tool System (Weeks 7-8)

#### 4.1 Tool Framework
```go
type Tool interface {
    ID() string
    Description() string
    Parameters() json.RawMessage  // JSON Schema
    Execute(ctx context.Context, args json.RawMessage, toolCtx ToolContext) (*ToolResult, error)
}

type ToolContext struct {
    SessionID string
    MessageID string
    Agent     string
    Abort     context.Context
    Metadata  func(title string, meta map[string]any)
}
```

#### 4.2 Core Tool Implementations
Priority order:
- [ ] **read** - File reading with line numbers
- [ ] **write** - File creation/overwriting
- [ ] **edit** - String replacement with fuzzy matching
- [ ] **bash** - Shell command execution
- [ ] **glob** - File pattern matching (via ripgrep)
- [ ] **grep** - Content search (via ripgrep)
- [ ] **list** - Directory listing
- [ ] **webfetch** - HTTP fetching
- [ ] **todowrite/todoread** - Task management

#### 4.3 Edit Tool - Fuzzy Matching
- [ ] Exact string matching
- [ ] Levenshtein distance fallback
- [ ] Block anchor strategy
- [ ] Line normalization (CRLF/LF)

#### 4.4 Bash Tool - Process Management
- [ ] Shell detection (bash/zsh)
- [ ] Process group management
- [ ] Timeout handling
- [ ] Output streaming + truncation
- [ ] Graceful termination (SIGTERM → SIGKILL)

#### 4.5 Tool Registration
- [ ] Built-in tool registry
- [ ] Dynamic tool loading (Go plugins or config)
- [ ] Per-agent tool filtering

**Deliverable**: All core tools working with proper validation

---

### Phase 5: Permission & Security (Week 9)

#### 5.1 Permission System
```go
type PermissionChecker interface {
    Check(ctx context.Context, req PermissionRequest) (PermissionResult, error)
    Approve(sessionID string, permType string, always bool)
    Reject(sessionID string, permType string)
}
```

#### 5.2 Permission Types
- [ ] Edit permission (file modifications)
- [ ] Bash permission (command execution)
- [ ] WebFetch permission (external requests)
- [ ] External directory permission
- [ ] Doom loop detection

#### 5.3 Bash Command Analysis
- [ ] Command parsing (go-shellwords or custom)
- [ ] Dangerous command detection
- [ ] Path extraction + validation
- [ ] Wildcard pattern matching

#### 5.4 Directory Isolation
- [ ] Working directory scoping
- [ ] External path detection
- [ ] `.env` file blocking

**Deliverable**: Full permission system with bash analysis

---

### Phase 6: Session Processing Loop (Week 10)

#### 6.1 Message Processing
- [ ] User message handling
- [ ] Assistant message creation
- [ ] Part management (text, reasoning, tool)
- [ ] Message history loading

#### 6.2 Agentic Loop
```go
func (s *SessionProcessor) Loop(ctx context.Context, sessionID string) error {
    for {
        // 1. Load message history
        // 2. Build system prompt
        // 3. Resolve available tools
        // 4. Call LLM with streaming
        // 5. Process stream events
        // 6. Execute tool calls
        // 7. Continue if tool calls, stop if done
    }
}
```

#### 6.3 Stream Event Processing
- [ ] text-delta → TextPart updates
- [ ] reasoning-delta → ReasoningPart updates
- [ ] tool-call-start/delta/end → ToolPart state machine
- [ ] finish → Cost calculation + token tracking

#### 6.4 Message Endpoint
```go
POST   /session/:id/message  // Stream completion
```

**Deliverable**: Full agentic loop working end-to-end

---

### Phase 7: Advanced Features (Weeks 11-12)

#### 7.1 LSP Integration
- [ ] LSP client implementation
- [ ] TypeScript server support
- [ ] Diagnostics on file save
- [ ] Hover information

#### 7.2 MCP Support
- [ ] MCP client (HTTP/SSE transport)
- [ ] MCP tool registration
- [ ] Remote tool execution

#### 7.3 Agent System
- [ ] Agent definitions
- [ ] Agent-specific permissions
- [ ] Subagent spawning (Task tool)
- [ ] Agent context inheritance

#### 7.4 Additional Endpoints
- [ ] OAuth flow endpoints
- [ ] TUI command endpoints
- [ ] Client tool registration
- [ ] Instance management

#### 7.5 Testing & Documentation
- [ ] Integration tests with TUI client
- [ ] API documentation
- [ ] Migration guide
- [ ] Performance benchmarks

**Deliverable**: Feature-complete server ready for production

---

## 5. Risk Assessment

### High Risk
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Provider SDK differences | Medium | High | Build abstraction early; test all providers |
| Edit tool fuzzy matching accuracy | Medium | High | Port algorithm carefully; comprehensive tests |

### Medium Risk
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| LSP integration complexity | Medium | Medium | Start with TypeScript only; expand later |
| Permission edge cases | Low | Medium | Port all test cases from TypeScript |

### Low Risk
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Protocol incompatibility | Low | High | Test with TUI client continuously |
| Performance regression | Very Low | Medium | Benchmark critical paths |

---

## 6. Resource Requirements

### Team
- **2-3 Go developers** with experience in:
  - HTTP services
  - Streaming/SSE
  - Process management
  - LLM APIs

### Infrastructure
- CI/CD pipeline for multi-platform builds
- Integration test environment with TUI client
- Access to all LLM provider APIs for testing

### Timeline
| Milestone | Week | Deliverable |
|-----------|------|-------------|
| Foundation | 2 | Core types, storage, events |
| HTTP Server | 4 | Basic endpoints working |
| LLM Integration | 6 | Streaming completions |
| Tool System | 8 | Core tools implemented |
| Security | 9 | Permission system complete |
| Processing Loop | 10 | End-to-end flow working |
| Polish | 12 | Production-ready release |

---

## 7. Recommendation

**Proceed with the rewrite.** The benefits outweigh the costs:

1. **Strategic alignment**: Go fits the infrastructure/CLI tooling domain
2. **Operational simplicity**: Single binary deployment
3. **Performance gains**: Lower memory, faster startup
4. **Team capability**: Existing Go code demonstrates proficiency
5. **Protocol stability**: No client changes required

### Suggested Approach
1. **Parallel development**: Keep TypeScript server running while building Go
2. **Incremental migration**: Route traffic gradually to Go server
3. **Feature flags**: Allow switching between implementations
4. **Comprehensive testing**: Integration tests with TUI client at every phase

---

## 8. Appendix: Key File Mappings

| TypeScript Source | Go Target | Priority |
|-------------------|-----------|----------|
| `server/server.ts` | `internal/server/` | P0 |
| `session/index.ts` | `internal/session/` | P0 |
| `session/message-v2.ts` | `internal/message/` | P0 |
| `provider/provider.ts` | `internal/provider/` | P0 |
| `tool/*.ts` | `internal/tool/` | P0 |
| `storage/storage.ts` | `internal/storage/` | P0 |
| `permission/index.ts` | `internal/permission/` | P1 |
| `lsp/` | `internal/lsp/` | P2 |
| `mcp/` | `internal/mcp/` | P2 |

---

*Document generated: 2025-11-26*
*Author: Claude (Opus 4)*
