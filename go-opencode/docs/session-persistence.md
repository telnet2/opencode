# Session Persistence and Management

## Overview

OpenCode uses a file-based persistence system that enables conversation continuity across server restarts. Sessions, messages, and their parts are stored as JSON files in a hierarchical directory structure, allowing the TUI client to reconnect to any previous session seamlessly.

This document describes how session management works in Go OpenCode, covering the data structures, storage mechanisms, and how persistence enables conversation continuity.

## Key Concepts

### Session

A **Session** represents a single conversation thread with the LLM. Each session:
- Belongs to a specific **Project** (identified by the git repository)
- Contains multiple **Messages** (user prompts and assistant responses)
- Tracks code changes made during the conversation
- Can be forked to create parallel conversation branches

### Project

A **Project** represents a git repository or working directory. Projects:
- Are identified by the git repository's initial commit SHA (ensuring consistency with TypeScript OpenCode)
- Group related sessions together
- Enable session isolation between different codebases

### Message

A **Message** represents either a user prompt or an assistant response. Messages:
- Contain multiple **Parts** (text, tool calls, files, etc.)
- Track token usage and costs
- Link to parent messages (for assistant responses)

### Part

A **Part** is a component of a message. Parts include:
- **TextPart**: Plain text content
- **ToolPart**: Tool invocations and their results
- **FilePart**: File attachments
- **ReasoningPart**: Extended thinking/reasoning content
- **StepStartPart/StepFinishPart**: Inference step markers
- **CompactionPart**: Conversation summarization markers
- And more...

## Data Structures

### Session Structure

```go
type Session struct {
    ID           string         `json:"id"`           // ULID identifier
    ProjectID    string         `json:"projectID"`    // Git commit SHA (first 16 chars)
    Directory    string         `json:"directory"`    // Working directory path
    ParentID     *string        `json:"parentID"`     // For forked sessions
    Title        string         `json:"title"`        // Session title
    Version      string         `json:"version"`      // Schema version
    Summary      SessionSummary `json:"summary"`      // Code change statistics
    Share        *SessionShare  `json:"share"`        // Sharing information
    Time         SessionTime    `json:"time"`         // Timestamps
    Revert       *SessionRevert `json:"revert"`       // Revert state
    CustomPrompt *CustomPrompt  `json:"customPrompt"` // Custom system prompt
}
```

### Message Structure

```go
type Message struct {
    ID         string       `json:"id"`         // ULID identifier
    SessionID  string       `json:"sessionID"`  // Parent session
    Role       string       `json:"role"`       // "user" | "assistant"
    Time       MessageTime  `json:"time"`       // Timestamps

    // User-specific fields
    Agent      string       `json:"agent"`      // Agent name
    Model      *ModelRef    `json:"model"`      // Model reference

    // Assistant-specific fields
    ParentID   string       `json:"parentID"`   // User message that prompted this
    ProviderID string       `json:"providerID"` // LLM provider
    ModelID    string       `json:"modelID"`    // Model used
    Cost       float64      `json:"cost"`       // API cost
    Tokens     *TokenUsage  `json:"tokens"`     // Token statistics
}
```

## Storage Architecture

### Directory Structure

All data is stored under the XDG data directory:

```
~/.local/share/opencode/storage/
├── session/
│   └── {projectID}/
│       ├── {sessionID}.json      # Session metadata
│       └── ...
├── message/
│   └── {sessionID}/
│       ├── {messageID}.json      # Message metadata
│       └── ...
├── part/
│   └── {messageID}/
│       ├── {partID}.json         # Message parts
│       └── ...
└── project/
    └── {projectID}.json          # Project metadata
```

### Storage Operations

The storage system provides five core operations:

| Operation | Method | Description |
|-----------|--------|-------------|
| Create/Update | `Put()` | Writes JSON to file with atomic rename |
| Read | `Get()` | Reads and unmarshals JSON file |
| Delete | `Delete()` | Removes file from storage |
| List | `List()` | Lists items in a directory |
| Scan | `Scan()` | Iterates over all items in a directory |

### Atomic Writes

All write operations use atomic file operations to prevent data corruption:

```go
// 1. Write to temporary file
tmpPath := filePath + ".tmp"
os.WriteFile(tmpPath, data, 0644)

// 2. Atomic rename (cannot be interrupted)
os.Rename(tmpPath, filePath)
```

### File Locking

The storage system uses file-level locking to prevent concurrent write conflicts:

```go
lock := s.getLock(filePath)
lock.Lock()
defer lock.Unlock()
// ... perform write operation
```

## Project ID Generation

### Git-Based Project IDs

Project IDs are derived from the git repository's initial commit SHA, ensuring:
- **Consistency**: Same repository always gets the same project ID
- **Compatibility**: Sessions created by TypeScript OpenCode are visible to Go OpenCode
- **Stability**: Project ID doesn't change even if directory path changes

```go
// Get the first (root) commit SHA
cmd := exec.Command("git", "rev-list", "--max-parents=0", "--all")
roots := strings.Split(output, "\n")
sort.Strings(roots)
projectID := roots[0]  // Use first root commit
```

### Caching

The project ID is cached in `.git/opencode` for fast subsequent lookups:

```
.git/
└── opencode           # Contains the project ID
```

### Non-Git Directories

For directories not in a git repository, sessions are stored under the `"global"` project ID.

## Session Lifecycle

### 1. Session Creation

```
User requests new session
         │
         ▼
┌─────────────────────┐
│ Detect project ID   │ ◄── git rev-list --max-parents=0 --all
│ (from git SHA)      │
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Check for migration │ ◄── Migrate from hash-based IDs
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Generate session ID │ ◄── ULID (time-sortable)
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Write to storage    │ ◄── session/{projectID}/{sessionID}.json
└─────────────────────┘
```

### 2. Message Processing

```
User sends message
         │
         ▼
┌─────────────────────┐
│ Create user message │ ◄── message/{sessionID}/{messageID}.json
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Process with LLM    │
│ (streaming)         │
└─────────┬───────────┘
         │
         ▼
┌─────────────────────────────────┐
│ Create assistant message        │
│ + parts (text, tools, etc.)     │ ◄── part/{messageID}/{partID}.json
└─────────┬───────────────────────┘
         │
         ▼
┌─────────────────────┐
│ Update session      │ ◄── Update timestamps, summary
└─────────────────────┘
```

### 3. Session Recovery (On Server Restart)

```
Server starts
         │
         ▼
┌─────────────────────┐
│ Initialize storage  │ ◄── Point to ~/.local/share/opencode/storage
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Client requests     │
│ session list        │
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Scan storage for    │ ◄── session/{projectID}/*.json
│ sessions            │
└─────────┬───────────┘
         │
         ▼
┌─────────────────────┐
│ Return to client    │ ◄── Sessions available immediately
└─────────────────────┘
```

## Migration System

### Hash-Based to Git-Based Migration

When Go OpenCode starts, it automatically migrates sessions from the old hash-based project ID format to the new git-based format:

```go
func migrateFromHashBasedID(directory, newProjectID string) {
    oldProjectID := SHA256(directory)[:16]  // Old format

    // Find sessions under old project ID
    oldSessions := storage.List(["session", oldProjectID])

    for _, session := range oldSessions {
        if session.Directory == directory {
            // Move to new location
            session.ProjectID = newProjectID
            storage.Put(["session", newProjectID, session.ID], session)
            storage.Delete(["session", oldProjectID, session.ID])
        }
    }
}
```

### Global Project Migration

Sessions created before git detection was implemented are migrated from `"global"` to project-specific storage:

```go
func migrateFromGlobal(directory, newProjectID string) {
    globalSessions := storage.List(["session", "global"])

    for _, session := range globalSessions {
        if session.Directory == directory {
            session.ProjectID = newProjectID
            storage.Put(["session", newProjectID, session.ID], session)
            storage.Delete(["session", "global", session.ID])
        }
    }
}
```

## Session Features

### Session Forking

Sessions can be forked to create parallel conversation branches:

```go
func Fork(sessionID, messageID string) *Session {
    // Create new session with parent reference
    newSession := Create(directory, title + " (fork)")
    newSession.ParentID = &sessionID

    // Copy messages up to fork point
    for _, msg := range GetMessages(sessionID) {
        CopyMessage(msg, newSession.ID)
        if msg.ID == messageID {
            break
        }
    }

    return newSession
}
```

### Session Compaction

When conversations become too long, sessions support compaction to summarize older messages:

```go
type CompactionPart struct {
    ID        string `json:"id"`
    Type      string `json:"type"`      // "compaction"
    Auto      bool   `json:"auto"`      // Automatic vs manual trigger
}
```

### Session Sharing

Sessions can be shared via URL:

```go
type SessionShare struct {
    URL string `json:"url"`  // Public share URL
}
```

### Session Revert

Sessions can be reverted to a previous state:

```go
type SessionRevert struct {
    MessageID string  `json:"messageID"`  // Target message
    PartID    *string `json:"partID"`     // Optional part
    Snapshot  *string `json:"snapshot"`   // Git snapshot
}
```

## API Endpoints

### Session Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/session` | GET | List all sessions for current project |
| `/session` | POST | Create new session |
| `/session/{id}` | GET | Get session details |
| `/session/{id}` | PATCH | Update session |
| `/session/{id}` | DELETE | Delete session |
| `/session/{id}/children` | GET | Get forked sessions |
| `/session/{id}/message` | GET | Get session messages |
| `/session/{id}/message` | POST | Send message (streaming) |

### Project Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/project` | GET | List all projects |
| `/project/current` | GET | Get current project |

## Compatibility

### TypeScript OpenCode Compatibility

Go OpenCode maintains full compatibility with TypeScript OpenCode sessions:

1. **Same Storage Location**: Both use XDG data directories
2. **Same Project IDs**: Both use git commit SHA
3. **Same JSON Schema**: Session, message, and part structures match
4. **Automatic Migration**: Legacy sessions are migrated on first access

### TUI Client Compatibility

The persistence system enables the TUI client to:

1. **List Previous Sessions**: Immediately available after server restart
2. **Resume Conversations**: Continue from any previous message
3. **Switch Sessions**: Move between different conversations
4. **View History**: Access complete message and tool call history

## Performance Considerations

### Lazy Loading

Sessions are loaded on-demand, not at startup:
- Server starts immediately regardless of session count
- Only requested sessions are read from disk
- Memory usage scales with active sessions

### File-Based vs Database

The file-based approach provides:
- **Simplicity**: No database setup or management
- **Debuggability**: Human-readable JSON files
- **Portability**: Easy backup and migration
- **Durability**: Files survive process crashes

Trade-offs:
- **Query Performance**: No indexing (full directory scan)
- **Concurrency**: File-level locking (not row-level)

## Troubleshooting

### Sessions Not Appearing

1. **Check Project ID**: Verify `.git/opencode` contains the correct SHA
2. **Check Directory**: Ensure sessions were created in the same directory
3. **Check Migration**: Look for sessions under old hash-based project IDs

### Session Data Corruption

1. **Check File Permissions**: Ensure write access to storage directory
2. **Check Disk Space**: Atomic writes need space for temp files
3. **Check JSON Validity**: Parse session files manually

### Migration Issues

1. **Manual Migration**: Move session files between project directories
2. **Reset Project ID**: Delete `.git/opencode` to regenerate

## References

- **Storage Implementation**: `go-opencode/internal/storage/storage.go`
- **Session Service**: `go-opencode/internal/session/service.go`
- **Project Package**: `go-opencode/internal/project/project.go`
- **Type Definitions**: `go-opencode/pkg/types/`
- **TypeScript Implementation**: `packages/opencode/src/session/`
