# Go-OpenCode Architecture

This document describes the package structure and dependencies of the Go OpenCode implementation.

## Package Dependency Graph

```
                                    ┌─────────────┐
                                    │   server    │ ◄── HTTP API Layer
                                    └──────┬──────┘
                                           │
          ┌────────────────────────────────┼────────────────────────────────┐
          │                                │                                │
          ▼                                ▼                                ▼
    ┌───────────┐                   ┌───────────┐                    ┌───────────┐
    │  session  │                   │    mcp    │                    │ executor  │
    └─────┬─────┘                   └─────┬─────┘                    └─────┬─────┘
          │                               │                                │
          │         ┌─────────────────────┴────────────────────────────────┤
          │         │                                                      │
          ▼         ▼                                                      ▼
    ┌───────────────────┐                                          ┌───────────┐
    │       tool        │ ◄──────────────────────────────────────  │  provider │
    └─────────┬─────────┘                                          └───────────┘
              │
    ┌─────────┴─────────┬────────────────┐
    │                   │                │
    ▼                   ▼                ▼
┌─────────┐       ┌───────────┐    ┌───────────┐
│  agent  │       │  storage  │    │   event   │
└────┬────┘       └───────────┘    └───────────┘
     │
     ▼
┌────────────┐
│ permission │
└────────────┘
```

## Package Descriptions

### Command Layer (`cmd/`)

| Package | Description |
|---------|-------------|
| `cmd/opencode` | Main CLI application entry point. Provides `run` and `serve` commands. |
| `cmd/calculator-mcp` | Example MCP server implementation for testing. |

### Public API (`pkg/`)

| Package | Description |
|---------|-------------|
| `pkg/types` | Shared type definitions used across the codebase (Session, Message, Part, etc.). |
| `pkg/mcpserver` | MCP Server implementation utilities for building MCP servers. |

### Internal Packages (`internal/`)

#### Foundation Layer (No internal dependencies)

| Package | Description |
|---------|-------------|
| `event` | Type-safe pub/sub event system. Enables decoupled communication between components for session events, message updates, and tool execution notifications. |
| `logging` | Structured logging using zerolog. Provides consistent logging across all packages. |
| `storage` | File-based JSON storage matching TypeScript implementation. Provides persistent storage for sessions, messages, and parts. |
| `formatter` | Code formatting integration. Supports automatic code formatting via external tools. |
| `lsp` | Language Server Protocol client. Provides code intelligence and symbol search capabilities. |
| `sharing` | Session sharing and collaboration features. |
| `command` | Flexible command execution system. Supports templated commands with variable substitution from configuration or markdown files. |

#### Configuration Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `config` | Configuration loading, merging, and path management. Handles hierarchical loading from multiple sources (global, project, environment) with TypeScript compatibility. | `logging` |

#### Permission & Safety Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `permission` | Comprehensive permission control system. Manages user consent for file editing, bash commands, web fetching, and external directory access. | `event` |
| `agent` | Multi-agent configuration and management. Implements flexible agent system with different operation modes (primary/subagent) and tool access controls. | `permission` |

#### Tool & Execution Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `tool` | Tool registry and execution framework. Manages tool registration, lookup, and execution. Includes built-in tools: Read, Write, Edit, Bash, Glob, Grep, List, WebFetch, TodoRead, TodoWrite, Task. | `agent`, `event`, `permission`, `storage` |
| `mcp` | Model Context Protocol (MCP) client. Connects to external MCP servers and exposes their tools, resources, and prompts to the LLM. | `tool` |
| `clienttool` | Registry for client-side tools. Enables external clients to register and execute tools via HTTP API. | `event` |
| `executor` | Task execution implementations. Runs subagent tasks by creating child sessions and managing their lifecycle. | `agent`, `event`, `permission`, `provider`, `session`, `storage`, `tool` |

#### Provider Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `provider` | LLM provider abstraction layer using Eino framework. Supports Anthropic (Claude), OpenAI (GPT), and Volcengine ARK models. Handles streaming, tool calls, and message formatting. | (none) |

#### Session Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `session` | Core agentic loop and session management. Manages conversations, message processing, tool execution, and session state. Implements the main LLM interaction loop with tool calling. | `event`, `logging`, `permission`, `provider`, `storage`, `tool` |

#### Integration Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `vcs` | Version control system (Git) integration. Provides repository status, diff tracking, and file change monitoring. | `event` |

#### API Layer

| Package | Description | Dependencies |
|---------|-------------|--------------|
| `server` | HTTP server implementation for OpenCode API. Provides RESTful endpoints for sessions, messages, files, config, events (SSE), MCP management, and client tools. | `clienttool`, `command`, `event`, `formatter`, `logging`, `lsp`, `mcp`, `provider`, `session`, `storage`, `tool`, `vcs` |

## Dependency Statistics

### Most Depended Upon (Foundation Packages)

1. **event** - 6 dependents (core pub/sub infrastructure)
2. **tool** - 5 dependents (tool execution framework)
3. **storage** - 4 dependents (persistence layer)
4. **provider** - 4 dependents (LLM abstraction)
5. **permission** - 4 dependents (safety controls)

### Hub Packages (High Fan-out)

1. **server** - imports 12 internal packages (API orchestration)
2. **executor** - imports 7 internal packages (subagent coordination)
3. **session** - imports 6 internal packages (agentic loop)

## Key Design Principles

1. **Layered Architecture**: Foundation packages (event, storage, logging) support higher-level abstractions (session, server).

2. **No Circular Dependencies**: The import cycle between `tool` and `session` was resolved by extracting `executor` as a separate package.

3. **Event-Driven Communication**: Components communicate via the `event` package for loose coupling.

4. **Permission-First Design**: All potentially dangerous operations (file writes, bash commands) go through the `permission` package.

5. **Provider Abstraction**: LLM providers are abstracted behind a common interface, enabling easy addition of new providers.

## External Dependencies

Key external packages used:

| Package | Purpose |
|---------|---------|
| `github.com/cloudwego/eino` | LLM framework for provider abstraction |
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/rs/zerolog` | Structured logging |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/mark3labs/mcp-go` | MCP protocol implementation |
| `github.com/JohannesKaufmann/html-to-markdown` | HTML to Markdown conversion (WebFetch tool) |
| `github.com/PuerkitoBio/goquery` | HTML parsing and text extraction (WebFetch tool) |
| `github.com/sergi/go-diff` | Diff computation for file changes |
