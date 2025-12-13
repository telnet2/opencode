# GitHub Packages Opportunities

This document identifies custom implementations in go-opencode that could be replaced with well-established GitHub packages.

## High Priority Replacements

| Current Implementation | File Location | Recommended Package |
|------------------------|---------------|---------------------|
| ~~Custom Event Bus/Pub-Sub~~ | ~~`internal/event/bus.go:1-182`~~ | **DONE**: [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) |
| File-Based Storage with Locking | `internal/storage/storage.go`, `lock.go` | [etcd-io/bbolt](https://github.com/etcd-io/bbolt) or [dgraph-io/badger](https://github.com/dgraph-io/badger) |
| Custom Permission System | `internal/permission/checker.go:1-214` | [casbin/casbin](https://github.com/casbin/casbin) |
| ~~No Structured Logging~~ | ~~Throughout codebase~~ | **DONE**: Implemented with [rs/zerolog](https://github.com/rs/zerolog) |

### Details

#### 1. ~~Custom Event Bus/Pub-Sub~~ (COMPLETED)

**Previous Implementation:**
- Hand-rolled pub/sub event system with subscriber registration, ID tracking
- Type-specific and global event subscriptions
- Concurrent publishing (async) and sync variants
- Manual subscription management with unsubscribe functions

**Implementation (COMPLETED):**
- Integrated [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) as the pub/sub infrastructure
- Uses watermill's `gochannel.GoChannel` for in-memory pub/sub
- Maintains original API compatibility (`Subscribe`, `SubscribeAll`, `Publish`, `PublishSync`)
- Exposes watermill's `GoChannel` via `PubSub()` for advanced use cases (middleware, routing, distributed backends)
- Preserves type information through direct subscriber callbacks

#### 2. File-Based Storage with Locking (`internal/storage/`)

**Current Implementation:**
- Custom file-based JSON storage layer with in-memory file lock management
- Path-based storage abstraction
- Atomic writes using temp files
- Directory-based scanning
- Manual flock implementation

**Why Replace:**
- ACID compliance
- Transaction support
- Better concurrency handling
- Atomic operations at database level (not file level)
- MVCC support (Badger)
- Built-in indexing and querying
- Better performance for concurrent access

#### 3. Custom Permission System (`internal/permission/checker.go:1-214`)

**Current Implementation:**
- Hand-rolled permission system with session-based approval tracking
- Pattern-based approval
- Event-driven permission requests
- Response channel matching

**Why Replace (with Casbin):**
- Policy-based instead of code-based
- Supports RBAC, ABAC, ACL
- Easy to audit and modify permissions
- Extensible with custom functions

#### 4. ~~Structured Logging~~ (COMPLETED)

**Previous Implementation:**
- No structured logging library detected
- Uses standard `fmt` and `log` packages
- Missing proper log levels, structured fields

**Implementation (COMPLETED):**
- Added `internal/logging` package using [rs/zerolog](https://github.com/rs/zerolog)
- Structured JSON logging with proper log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- Pretty-print console output mode for development
- Global logger with convenient helper functions
- Integrated with CLI via `--print-logs` and `--log-level` flags

---

## Medium Priority Replacements

| Current Implementation | File Location | Recommended Package |
|------------------------|---------------|---------------------|
| ~~Custom Levenshtein Distance~~ | ~~`internal/tool/edit.go:232-281`~~ | **DONE**: [agnivade/levenshtein](https://github.com/agnivade/levenshtein) |
| Manual JSON-RPC (LSP) | `internal/lsp/client.go:200-343` | [sourcegraph/jsonrpc2](https://github.com/sourcegraph/jsonrpc2) |
| Manual JSON-RPC (MCP) | `internal/mcp/transport.go:1-334` | [sourcegraph/jsonrpc2](https://github.com/sourcegraph/jsonrpc2) |
| Custom Config with Interpolation | `internal/config/config.go:1-364` | [spf13/viper](https://github.com/spf13/viper) |

### Details

#### 1. Custom Levenshtein Distance (`internal/tool/edit.go:232-281`)

**Current Implementation:**
- Hand-rolled Levenshtein distance algorithm for fuzzy string matching
- Used to find best match when exact string replacement fails
- Full matrix-based implementation with optimization for long strings

**Why Replace:**
- Battle-tested, optimized implementations
- Better performance for large strings
- Well-maintained with community support
- Handles edge cases more robustly

#### 2. Manual JSON-RPC (LSP) (`internal/lsp/client.go:200-343`)

**Current Implementation:**
- Manual JSON-RPC 2.0 protocol implementation including:
  - Message header parsing (Content-Length)
  - Request/response matching via ID
  - Pending request tracking with channels
  - Message serialization/deserialization

**Why Replace:**
- Handles protocol details correctly
- Better error handling and edge cases
- Familiar to LSP community
- Used by official language servers
- Handles connection lifecycle better

#### 3. Manual JSON-RPC (MCP) (`internal/mcp/transport.go:1-334`)

**Current Implementation:**
- Manual JSON-RPC protocol over HTTP and stdio with:
  - Newline-delimited JSON parsing for stdio
  - HTTP POST-based JSON-RPC
  - Manual ID tracking and pending request management

**Why Replace:**
- Standardized implementation
- Tested with various MCP servers
- Better connection management
- Cleaner error handling

#### 4. Custom Config with Interpolation (`internal/config/config.go:1-364`)

**Current Implementation:**
- Custom configuration loader with multiple source priority handling
- JSON/JSONC parsing (uses `tidwall/jsonc`)
- Custom interpolation (`{env:VAR}`, `{file:path}`)
- Regex-based placeholder replacement
- Manual config merging and normalization

**Why Replace:**
- Built-in support for multiple formats (YAML, TOML, JSON, etc.)
- Automatic environment variable binding
- Nested configuration support
- Config watching/reloading support
- Better validation and defaults

---

## Low Priority Replacements

| Current Implementation | File Location | Recommended Package |
|------------------------|---------------|---------------------|
| ~~Manual Exponential Backoff~~ | ~~`internal/session/loop.go:164-199`~~ | **DONE**: [cenkalti/backoff](https://github.com/cenkalti/backoff) |
| Custom SSE Implementation | `internal/server/sse.go:1-178` | [r3labs/sse](https://github.com/r3labs/sse) |
| Manual Process Management | `internal/tool/bash.go:162-260` | [creack/pty](https://github.com/creack/pty), [oklog/run](https://github.com/oklog/run) |

### Details

#### 1. Manual Exponential Backoff (`internal/session/loop.go:164-199`)

**Current Implementation:**
```go
// Lines 175-178: Manual exponential backoff
retries++
delay := RetryBaseDelay * time.Duration(1<<retries)
time.Sleep(delay)
```

**Why Replace:**
- Jitter support to prevent thundering herd
- Maximum delay caps
- Circuit breaker patterns
- Better error classification
- Context awareness

#### 2. Custom SSE Implementation (`internal/server/sse.go:1-178`)

**Current Implementation:**
- Manual Server-Sent Events implementation with custom SSE writer
- Heartbeat mechanism
- Event filtering per session

**Why Replace:**
- Automatic heartbeat management
- Connection tracking
- Better error handling
- Browser compatibility handling

#### 3. Manual Process Management (`internal/tool/bash.go:162-260`)

**Current Implementation:**
- Manual process group management
- Signal handling (SIGTERM -> SIGKILL)
- Custom process killing logic with syscall.Flock

**Why Replace:**
- Better terminal emulation support
- Proper signal handling
- Process pooling and resource management
- Cross-platform compatibility

---

## Already Using Best Practices

These implementations are already using appropriate packages:

| Implementation | Package Used | Status |
|----------------|--------------|--------|
| ULID generation | `oklog/ulid/v2` | Excellent choice |
| Bash parsing | `mvdan.cc/sh/v3` | Appropriate and well-maintained |
| HTTP framework | `go-chi/chi/v5` | Excellent choice |
| Glob patterns | `doublestar/v4` | Already in go.mod |

---

## Summary

### Priority Matrix

1. **High Priority** - Significant improvements in reliability, maintainability, and features:
   - ~~Structured logging~~ - **DONE** (zerolog)
   - ~~Event bus~~ - **DONE** (watermill)
   - Storage layer (ACID compliance, transactions)
   - Permissions (policy-based, auditable)

2. **Medium Priority** - Code quality and standardization:
   - JSON-RPC implementations (LSP/MCP)
   - ~~Levenshtein distance~~ - **DONE** (agnivade/levenshtein)
   - Configuration management

3. **Low Priority** - Nice to have improvements:
   - ~~Exponential backoff~~ - **DONE** (cenkalti/backoff)
   - SSE implementation
   - Process management

### Quick Wins

1. ~~**Integrate logrus**~~ - **DONE**: Implemented with zerolog instead
2. **Use doublestar/v4 more** - Already in go.mod but underutilized
3. ~~**Replace Levenshtein**~~ - **DONE**: Replaced with agnivade/levenshtein
