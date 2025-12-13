# LLM Assistant Support Feasibility Report

## Executive Summary

This report evaluates the feasibility of running an AI assistant (from `packages/opencode`) over the memory file system provided by `go-memsh` using a client-server architecture. The analysis indicates this integration is **feasible** with moderate development effort, leveraging the existing JSON-RPC API in go-memsh and implementing a file system provider abstraction in opencode.

**Verdict: Feasible with Medium Complexity**

| Aspect | Assessment |
|--------|------------|
| Technical Feasibility | High |
| Architecture Compatibility | Good |
| Implementation Effort | Medium (2-4 weeks) |
| Risk Level | Low-Medium |

---

## 1. System Overview

### 1.1 go-memsh Capabilities

The go-memsh project provides:

- **In-Memory File System**: Built on [afero.MemMapFs](https://github.com/spf13/afero), providing full POSIX-like file operations
- **Shell Interpreter**: 40+ built-in commands including file operations, text processing, HTTP, and JSON handling
- **Client-Server Architecture**: REST API + JSON-RPC 2.0 over WebSocket
- **Session Isolation**: Each session has isolated filesystem, environment, and working directory

**Key APIs:**
```
POST /api/v1/session/create    - Create isolated session
POST /api/v1/session/list      - List active sessions
POST /api/v1/session/remove    - Remove session
WS   /api/v1/session/repl      - JSON-RPC command execution
```

### 1.2 OpenCode Assistant Architecture

The opencode project provides:

- **Tool-Based Architecture**: Pluggable tools for file operations (read, write, edit, glob, grep, bash)
- **Permission System**: Three-level model (allow/deny/ask) for controlled access
- **Instance Isolation**: Per-project state management via `Instance.provide()`
- **File Tracking**: FileTime system prevents concurrent modification conflicts
- **Snapshot System**: Git-based tracking for undo/restore capabilities

**File System Operations Used:**
| Tool | Operations | Current Implementation |
|------|------------|----------------------|
| read | read, stat, exists | Bun.file() API |
| write | write, mkdir | Bun.write() API |
| edit | read, write, stat | Bun.file() + Bun.write() |
| glob | file pattern search | ripgrep binary |
| grep | content search | ripgrep binary |
| bash | shell execution | child_process.spawn() |
| list | directory tree | ripgrep + fs.readdir() |

---

## 2. Integration Architecture Options

### Option A: Protocol Bridge (Recommended)

Create a TypeScript adapter that translates opencode tool calls to go-memsh JSON-RPC calls.

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   OpenCode      │     │   MemFS Adapter  │     │   go-memsh      │
│   Assistant     │────▶│   (TypeScript)   │────▶│   API Server    │
│                 │     │                  │     │                 │
│  Tool Calls:    │     │  Translates to   │     │  Executes on    │
│  - read         │     │  JSON-RPC:       │     │  MemMapFs       │
│  - write        │     │  shell.execute   │     │                 │
│  - edit         │     │                  │     │                 │
│  - glob         │     │  WebSocket       │     │                 │
│  - grep         │     │  Connection      │     │                 │
│  - bash         │     │                  │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

**Advantages:**
- Minimal changes to existing opencode tools
- Leverages existing go-memsh API
- Clear separation of concerns
- Both projects can evolve independently

**Disadvantages:**
- Network overhead (WebSocket latency)
- Requires adapter maintenance
- Two processes to manage

### Option B: Embedded Go Runtime

Embed go-memsh as a library and call directly via FFI or WASM.

```
┌─────────────────────────────────────────┐
│            OpenCode Process             │
│  ┌───────────────┐  ┌────────────────┐  │
│  │   Assistant   │  │  go-memsh      │  │
│  │   Tools       │──│  (WASM/FFI)    │  │
│  │               │  │                │  │
│  │  TypeScript   │  │  Go Runtime    │  │
│  └───────────────┘  └────────────────┘  │
└─────────────────────────────────────────┘
```

**Advantages:**
- No network overhead
- Single process deployment
- Tighter integration

**Disadvantages:**
- Complex FFI/WASM setup
- Memory management challenges
- Harder to debug
- Go WASM limitations

### Option C: Dual-Mode Provider

Abstract file system operations behind an interface, supporting both local and remote modes.

```typescript
interface FileSystemProvider {
  read(path: string): Promise<string>
  write(path: string, content: string): Promise<void>
  exists(path: string): Promise<boolean>
  stat(path: string): Promise<FileStat>
  glob(pattern: string): Promise<string[]>
  grep(pattern: string, path: string): Promise<SearchResult[]>
  exec(command: string): Promise<ExecResult>
}

class LocalFSProvider implements FileSystemProvider { /* Bun APIs */ }
class MemFSProvider implements FileSystemProvider { /* go-memsh JSON-RPC */ }
```

**Advantages:**
- Clean abstraction
- Easy testing with mock providers
- Future-proof for other backends

**Disadvantages:**
- Requires refactoring all file tools
- Higher upfront development cost

---

## 3. Tool Mapping Analysis

### 3.1 Direct Mappings (Easy)

| OpenCode Tool | go-memsh Command | Complexity |
|--------------|------------------|------------|
| read | `cat <file>` | Low |
| write | `echo "content" > <file>` or custom | Low |
| bash | Direct script execution | Low |
| list (ls) | `ls -la <path>` | Low |

### 3.2 Composite Mappings (Medium)

| OpenCode Tool | go-memsh Implementation | Complexity |
|--------------|------------------------|------------|
| glob | `find <path> -name "<pattern>"` | Medium |
| grep | `grep "<pattern>" <path>` | Medium |
| edit | read + replace + write | Medium |

### 3.3 Requires Enhancement (Higher Effort)

| OpenCode Tool | Required Enhancement | Complexity |
|--------------|---------------------|------------|
| stat (mtime) | Add `stat` builtin to go-memsh | Medium |
| mkdir -p | Already supported | Low |
| ripgrep features | Implement subset in go-memsh | High |

---

## 4. Implementation Plan

### Phase 1: Core Adapter (Week 1)

1. **Create MemFS Provider Interface**
   ```typescript
   // packages/opencode/src/provider/memfs.ts
   export interface MemFSProvider {
     sessionId: string
     connect(): Promise<void>
     execute(command: string, args?: string[]): Promise<ExecuteResult>
     disconnect(): Promise<void>
   }
   ```

2. **Implement WebSocket Client**
   - Connect to go-memsh API server
   - Handle JSON-RPC 2.0 protocol
   - Manage session lifecycle

3. **Basic Tool Adapters**
   - read → `cat`
   - write → `echo > file` or shell heredoc
   - list → `ls -la`

### Phase 2: Search Operations (Week 2)

1. **Glob Adapter**
   - Translate glob patterns to `find` commands
   - Handle result parsing and sorting by mtime

2. **Grep Adapter**
   - Map grep options to go-memsh grep
   - Consider adding extended grep options to go-memsh

3. **Add Missing Commands to go-memsh**
   - `stat` command for file metadata
   - Enhanced `find` with more filters

### Phase 3: Edit and Advanced Features (Week 3)

1. **Edit Tool Adapter**
   - Implement read-modify-write cycle
   - Port replacement strategies to work with remote content
   - Handle diff generation

2. **Permission Integration**
   - Extend permission system for remote operations
   - Session-based permission tracking

3. **Error Handling**
   - Map go-memsh errors to opencode error types
   - Handle connection failures gracefully

### Phase 4: Testing and Optimization (Week 4)

1. **Integration Tests**
   - Tool functionality tests
   - Session management tests
   - Error scenario tests

2. **Performance Optimization**
   - Connection pooling
   - Command batching for related operations
   - Caching strategies

---

## 5. Required go-memsh Enhancements

### 5.1 New Commands Needed

```go
// stat - Return file metadata
// Usage: stat <file>
// Output: JSON with size, mtime, mode, isDir

// readfile - Return file contents without cat formatting
// Usage: readfile <file>
// Output: Raw file content

// writefile - Write content from stdin to file
// Usage: writefile <file> <<< "content"
// Better handling for large content
```

### 5.2 API Enhancements

```go
// Extended shell.execute response
type ExecuteResult struct {
    Output   []string `json:"output"`
    Cwd      string   `json:"cwd"`
    Error    string   `json:"error"`
    ExitCode int      `json:"exit_code"`  // NEW
    Files    []string `json:"files"`      // NEW: modified files
}

// New method: shell.readFile
// Direct file read without shell parsing overhead

// New method: shell.writeFile
// Direct file write for large content
```

### 5.3 Search Enhancements

```go
// Enhanced find command
// find <path> -name "*.ts" -type f -mtime +1d

// Enhanced grep command
// grep -r -n -l "<pattern>" <path> --include="*.ts"
```

---

## 6. Risk Assessment

### 6.1 Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| WebSocket reliability | Low | Medium | Reconnection logic, heartbeat |
| Large file handling | Medium | Medium | Streaming API, chunked transfer |
| Search performance | Medium | Low | Index caching, limit results |
| Edit conflicts | Low | High | FileTime tracking over API |

### 6.2 Integration Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| API version mismatch | Medium | Medium | Version negotiation, compatibility layer |
| Feature gap in go-memsh | Medium | Medium | Prioritize core features first |
| Permission model mismatch | Low | Low | Adapter handles permission translation |

---

## 7. Alternative Approaches

### 7.1 Pure TypeScript MemFS

Instead of go-memsh, implement memory filesystem directly in TypeScript:

```typescript
import { Volume, createFsFromVolume } from 'memfs'

const vol = new Volume()
const fs = createFsFromVolume(vol)
```

**Pros:** Single language, no network overhead
**Cons:** No shell scripting, must reimplement commands

### 7.2 Docker/Container Integration

Run assistant inside a container with tmpfs:

```bash
docker run --tmpfs /workspace opencode-assistant
```

**Pros:** Full isolation, real shell
**Cons:** Heavy, startup overhead, not truly in-memory

### 7.3 Browser-Based (WebContainer)

Use WebContainer API for browser-based memory filesystem:

```typescript
import { WebContainer } from '@webcontainer/api'
const container = await WebContainer.boot()
```

**Pros:** Runs in browser, WASM-based Node.js
**Cons:** Browser-only, limited to Node.js runtime

---

## 8. Recommended Approach

### Recommendation: Option A (Protocol Bridge) with Option C Preparation

1. **Start with Protocol Bridge** - Quickest path to working integration
2. **Design with Provider Interface** - Enable future flexibility
3. **Enhance go-memsh incrementally** - Add features as needed

### Implementation Priority

```
High Priority (Must Have):
├── read/write file operations
├── directory listing
├── basic shell execution
└── session management

Medium Priority (Should Have):
├── glob file search
├── grep content search
├── edit tool support
└── error handling

Low Priority (Nice to Have):
├── snapshot/restore
├── LSP integration over memfs
├── advanced search features
└── performance optimizations
```

---

## 9. Code Examples

### 9.1 MemFS Client (TypeScript)

```typescript
// packages/opencode/src/provider/memfs-client.ts
import WebSocket from 'ws'

export class MemFSClient {
  private ws: WebSocket | null = null
  private requestId = 0
  private pending = new Map<number, { resolve: Function, reject: Function }>()

  constructor(
    private serverUrl: string,
    public sessionId: string
  ) {}

  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(`${this.serverUrl}/api/v1/session/repl`)
      this.ws.on('open', () => resolve())
      this.ws.on('error', reject)
      this.ws.on('message', (data) => this.handleMessage(data.toString()))
    })
  }

  async execute(command: string, args: string[] = []): Promise<ExecuteResult> {
    const id = ++this.requestId
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      this.ws?.send(JSON.stringify({
        jsonrpc: '2.0',
        method: 'shell.execute',
        params: { session_id: this.sessionId, command, args },
        id
      }))
    })
  }

  private handleMessage(data: string) {
    const response = JSON.parse(data)
    const pending = this.pending.get(response.id)
    if (pending) {
      this.pending.delete(response.id)
      if (response.error) {
        pending.reject(new Error(response.error.message))
      } else {
        pending.resolve(response.result)
      }
    }
  }
}
```

### 9.2 Read Tool Adapter

```typescript
// packages/opencode/src/tool/read-memfs.ts
import { MemFSClient } from '../provider/memfs-client'

export async function readFile(
  client: MemFSClient,
  path: string
): Promise<{ content: string; exists: boolean }> {
  // Check if file exists
  const testResult = await client.execute('test', ['-f', path])
  if (testResult.error) {
    return { content: '', exists: false }
  }

  // Read file content
  const result = await client.execute('cat', [path])
  if (result.error) {
    throw new Error(`Failed to read ${path}: ${result.error}`)
  }

  return {
    content: result.output.join('\n'),
    exists: true
  }
}
```

### 9.3 Session Manager Integration

```typescript
// packages/opencode/src/session/memfs-session.ts
export class MemFSSession {
  private client: MemFSClient | null = null

  async initialize(serverUrl: string): Promise<void> {
    // Create session via REST API
    const response = await fetch(`${serverUrl}/api/v1/session/create`, {
      method: 'POST'
    })
    const { session } = await response.json()

    // Connect WebSocket
    this.client = new MemFSClient(serverUrl, session.id)
    await this.client.connect()
  }

  getClient(): MemFSClient {
    if (!this.client) throw new Error('Session not initialized')
    return this.client
  }
}
```

---

## 10. Conclusion

Running the opencode AI assistant over go-memsh's memory file system is **technically feasible** and offers compelling benefits:

1. **Sandboxed Execution**: Complete isolation from host filesystem
2. **Reproducible Environments**: Each session starts fresh
3. **Browser Compatibility**: Potential for web-based assistant
4. **Scalability**: Stateless server design enables horizontal scaling

The recommended approach is to implement a **Protocol Bridge** adapter that translates opencode tool calls to go-memsh JSON-RPC commands. This approach:

- Minimizes changes to existing codebases
- Leverages proven APIs on both sides
- Allows incremental development and testing
- Maintains clear architectural boundaries

With an estimated **2-4 weeks** of development effort, a functional prototype can demonstrate the integration, with additional time for production hardening and feature completeness.

---

## Appendix A: Command Compatibility Matrix

| OpenCode Operation | go-memsh Command | Status | Notes |
|-------------------|------------------|--------|-------|
| Read file | `cat` | Ready | |
| Write file | `echo >` | Ready | Large files need enhancement |
| Create directory | `mkdir -p` | Ready | |
| Delete file | `rm` | Ready | |
| Delete directory | `rm -r` | Ready | |
| Copy file | `cp` | Ready | |
| Move file | `mv` | Ready | |
| List directory | `ls` | Ready | |
| Find files | `find` | Ready | Basic patterns |
| Search content | `grep` | Ready | Basic patterns |
| File exists | `test -f` | Ready | |
| Dir exists | `test -d` | Ready | |
| File stat | - | Needed | Add `stat` command |
| Execute script | `sh` | Ready | |
| Environment vars | `export`, `env` | Ready | |

## Appendix B: References

- go-memsh API Documentation: `go-memsh/API.md`
- go-memsh Design Document: `go-memsh/DESIGN.md`
- OpenCode Tool System: `packages/opencode/src/tool/`
- afero Library: https://github.com/spf13/afero
- mvdan/sh Parser: https://github.com/mvdan/sh
