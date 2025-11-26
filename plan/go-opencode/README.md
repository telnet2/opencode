# Go OpenCode Server Implementation Plan

## Overview

This directory contains the detailed implementation plan for rewriting the OpenCode server in Go. The plan preserves full compatibility with the existing TUI client by maintaining the same REST + SSE protocol.

## Documents

| Document | Description |
|----------|-------------|
| [01-foundation.md](./01-foundation.md) | Core types, storage, event bus |
| [02-http-server.md](./02-http-server.md) | HTTP server, routing, middleware |
| [03-llm-providers.md](./03-llm-providers.md) | LLM provider abstraction and implementations |
| [04-tool-system.md](./04-tool-system.md) | Tool framework and implementations |
| [05-permission-security.md](./05-permission-security.md) | Permission system and bash parsing |
| [06-session-processing.md](./06-session-processing.md) | Agentic loop and message processing |
| [07-advanced-features.md](./07-advanced-features.md) | LSP, MCP, agents |
| [test-plan.md](./test-plan.md) | Comprehensive test strategy |
| [technical-specs.md](./technical-specs.md) | Technical specifications |

## Project Structure

```
go-opencode/
├── cmd/
│   └── opencode-server/
│       └── main.go                 # Entry point
├── internal/
│   ├── server/                     # HTTP server + routes
│   │   ├── server.go
│   │   ├── routes.go
│   │   ├── middleware.go
│   │   └── sse.go
│   ├── session/                    # Session management
│   │   ├── session.go
│   │   ├── processor.go
│   │   └── prompt.go
│   ├── message/                    # Message types + storage
│   │   ├── message.go
│   │   ├── parts.go
│   │   └── convert.go
│   ├── provider/                   # LLM provider abstraction
│   │   ├── provider.go
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   ├── google.go
│   │   └── transform.go
│   ├── tool/                       # Tool system
│   │   ├── tool.go
│   │   ├── registry.go
│   │   ├── bash.go
│   │   ├── read.go
│   │   ├── write.go
│   │   ├── edit.go
│   │   ├── glob.go
│   │   └── grep.go
│   ├── permission/                 # Permission checking
│   │   ├── permission.go
│   │   └── bash_parser.go
│   ├── storage/                    # File-based storage
│   │   ├── storage.go
│   │   └── lock.go
│   ├── config/                     # Configuration loading
│   │   ├── config.go
│   │   └── paths.go
│   ├── event/                      # Event bus + SSE
│   │   ├── bus.go
│   │   └── types.go
│   ├── lsp/                        # LSP integration
│   │   └── client.go
│   └── mcp/                        # MCP support
│       └── client.go
├── pkg/
│   └── types/                      # Exported types
│       ├── session.go
│       ├── message.go
│       └── config.go
├── test/
│   ├── fixture/                    # Test utilities
│   ├── integration/                # Integration tests
│   └── unit/                       # Unit tests
├── go.mod
├── go.sum
└── Makefile
```

## Timeline Summary

| Phase | Duration | Focus | Status |
|-------|----------|-------|--------|
| 1. Foundation | Weeks 1-2 | Core types, storage, event bus | ✅ COMPLETE |
| 2. HTTP Server | Weeks 3-4 | REST endpoints, SSE streaming | ✅ COMPLETE |
| 3. LLM Integration | Weeks 5-6 | Provider abstraction, streaming | ✅ COMPLETE |
| 4. Tool System | Weeks 7-8 | Core tools implementation | ✅ COMPLETE |
| 5. Security | Week 9 | Permission system, bash parsing (mvdan/sh) | 🔲 Pending |
| 6. Processing Loop | Week 10 | Agentic loop, message handling | 🔲 Pending |
| 7. Polish | Weeks 11-12 | LSP, MCP, testing, documentation | 🔲 Pending |

**Implementation Progress:** 104 tests passing across all Phase 1-4 components (as of 2025-11-26)

## Key Dependencies

```go
// go.mod
module github.com/opencode-ai/opencode-server

go 1.22

require (
    // HTTP
    github.com/go-chi/chi/v5 v5.0.12
    github.com/go-chi/cors v1.2.1

    // LLM Providers
    github.com/anthropics/anthropic-sdk-go v0.2.0
    github.com/openai/openai-go v0.1.0
    google.golang.org/genai v0.1.0

    // Shell Parsing (from go-memsh)
    mvdan.cc/sh/v3 v3.12.0

    // Utilities
    github.com/oklog/ulid/v2 v2.1.0
    github.com/fsnotify/fsnotify v1.7.0
    github.com/go-playground/validator/v10 v10.18.0

    // Testing
    github.com/stretchr/testify v1.9.0
)
```

## Success Criteria

1. **Protocol Compatibility**: TUI client works without modification
2. **Feature Parity**: All 60+ endpoints implemented
3. **Test Coverage**: >80% coverage on critical paths
4. **Performance**: Lower memory, faster startup than TypeScript version
5. **Documentation**: OpenAPI spec, migration guide

## Getting Started

```bash
# Build
make build

# Run tests
make test

# Run server
./bin/opencode-server --port 8080

# Run with TUI client (verification)
opencode --server http://localhost:8080
```
