# MySQL Storage Design

## Overview

This document describes an alternative storage design using MySQL optimized for high-scale deployments. The design avoids stored procedures, foreign keys, and triggers for maximum portability and performance, using efficient `BIGINT` primary keys instead of UUIDs.

## Design Principles

### Why These Constraints?

| Constraint | Reason |
|------------|--------|
| No Foreign Keys | Eliminates FK checks on writes, enables easier sharding |
| No Stored Procedures | Application-level logic, better portability |
| No Triggers | Predictable performance, easier debugging |
| BIGINT Keys | 8 bytes vs 16 bytes (UUID), better index performance |

### Trade-offs

**Advantages**:
- 50% smaller primary key storage
- Faster index lookups (sequential vs random)
- No FK constraint overhead on inserts
- Easier horizontal sharding
- Better cache locality

**Considerations**:
- Application must enforce referential integrity
- Need distributed ID generation strategy
- Orphan cleanup requires background jobs

## ID Generation

### Snowflake ID Structure

Use Twitter Snowflake-style IDs for distributed, time-ordered, unique identifiers:

```
┌─────────────────────────────────────────────────────────────────┐
│ 63 bits total (signed BIGINT)                                   │
├─────────────────────┬──────────────┬────────────┬───────────────┤
│ Timestamp (41 bits) │ Worker (10)  │ Seq (12)   │ Sign (1)      │
│ ~69 years           │ 1024 workers │ 4096/ms    │ Always 0      │
└─────────────────────┴──────────────┴────────────┴───────────────┘
```

### ID Generator Implementation

```typescript
class SnowflakeGenerator {
  private readonly epoch = 1704067200000n // 2024-01-01 00:00:00 UTC
  private readonly workerIdBits = 10n
  private readonly sequenceBits = 12n

  private readonly maxWorkerId = (1n << this.workerIdBits) - 1n
  private readonly maxSequence = (1n << this.sequenceBits) - 1n

  private readonly workerIdShift = this.sequenceBits
  private readonly timestampShift = this.sequenceBits + this.workerIdBits

  private workerId: bigint
  private sequence = 0n
  private lastTimestamp = -1n

  constructor(workerId: number) {
    if (workerId < 0 || BigInt(workerId) > this.maxWorkerId) {
      throw new Error(`Worker ID must be between 0 and ${this.maxWorkerId}`)
    }
    this.workerId = BigInt(workerId)
  }

  nextId(): bigint {
    let timestamp = BigInt(Date.now()) - this.epoch

    if (timestamp === this.lastTimestamp) {
      this.sequence = (this.sequence + 1n) & this.maxSequence
      if (this.sequence === 0n) {
        // Wait for next millisecond
        while (timestamp <= this.lastTimestamp) {
          timestamp = BigInt(Date.now()) - this.epoch
        }
      }
    } else {
      this.sequence = 0n
    }

    this.lastTimestamp = timestamp

    return (
      (timestamp << this.timestampShift) |
      (this.workerId << this.workerIdShift) |
      this.sequence
    )
  }

  // Extract timestamp from ID
  static getTimestamp(id: bigint): Date {
    const epoch = 1704067200000n
    const timestamp = (id >> 22n) + epoch
    return new Date(Number(timestamp))
  }
}

// Usage
const idGen = new SnowflakeGenerator(parseInt(process.env.WORKER_ID || "1"))
const sessionId = idGen.nextId() // 7159429562834944001n
```

### Worker ID Assignment

```typescript
// Assign worker IDs via environment or coordination service
interface WorkerIdConfig {
  // Static assignment via environment
  static: {
    workerId: number
  }
  // Dynamic assignment via Redis
  redis: {
    key: "workers:ids"
    ttl: 60 // seconds, heartbeat interval
  }
  // Kubernetes pod ordinal
  kubernetes: {
    statefulSetName: string
    // Pod name: opencode-api-3 → workerId: 3
  }
}

// Redis-based dynamic assignment
async function acquireWorkerId(redis: Redis): Promise<number> {
  for (let id = 0; id < 1024; id++) {
    const key = `worker:${id}`
    const acquired = await redis.set(key, process.pid, {
      nx: true,
      ex: 60,
    })
    if (acquired) {
      // Start heartbeat
      setInterval(() => redis.expire(key, 60), 30000)
      return id
    }
  }
  throw new Error("No available worker IDs")
}
```

## MySQL Schema

### Core Tables

```sql
-- Organizations (tenants)
CREATE TABLE organizations (
    id BIGINT NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    settings JSON NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    UNIQUE KEY uk_slug (slug),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Users
CREATE TABLE users (
    id BIGINT NOT NULL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    avatar_url VARCHAR(500),
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    preferences JSON NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    last_active_at TIMESTAMP(3) NULL,

    UNIQUE KEY uk_org_email (org_id, email),
    KEY idx_org_id (org_id),
    KEY idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Workspaces
CREATE TABLE workspaces (
    id BIGINT NOT NULL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    git_config JSON,
    settings JSON NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    KEY idx_org_id (org_id),
    KEY idx_org_name (org_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Projects
CREATE TABLE projects (
    id BIGINT NOT NULL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(1000) NOT NULL,
    git_commit VARCHAR(40),
    settings JSON NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    KEY idx_workspace_id (workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Sessions
CREATE TABLE sessions (
    id BIGINT NOT NULL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    parent_id BIGINT NULL,
    title VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    model_provider VARCHAR(100) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    summary JSON,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    expires_at TIMESTAMP(3) NULL,

    KEY idx_project_id (project_id),
    KEY idx_user_id (user_id),
    KEY idx_user_created (user_id, created_at DESC),
    KEY idx_status (status),
    KEY idx_parent_id (parent_id),
    KEY idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Messages
CREATE TABLE messages (
    id BIGINT NOT NULL PRIMARY KEY,
    session_id BIGINT NOT NULL,
    role VARCHAR(50) NOT NULL,
    metadata JSON NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    completed_at TIMESTAMP(3) NULL,

    KEY idx_session_id (session_id),
    KEY idx_session_created (session_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Message Parts
CREATE TABLE message_parts (
    id BIGINT NOT NULL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    content JSON NOT NULL,
    sort_order INT NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_message_id (message_id),
    KEY idx_message_order (message_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Session Diffs
CREATE TABLE session_diffs (
    id BIGINT NOT NULL PRIMARY KEY,
    session_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    diff_content MEDIUMTEXT NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_session_id (session_id),
    KEY idx_message_id (message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Authentication Tables

```sql
-- API Keys
CREATE TABLE api_keys (
    id BIGINT NOT NULL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    prefix VARCHAR(20) NOT NULL,
    hash VARCHAR(255) NOT NULL,
    scopes JSON NOT NULL,
    rate_limit JSON,
    expires_at TIMESTAMP(3) NULL,
    last_used_at TIMESTAMP(3) NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_org_id (org_id),
    KEY idx_user_id (user_id),
    KEY idx_prefix (prefix)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Refresh Tokens
CREATE TABLE refresh_tokens (
    id BIGINT NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    family_id BIGINT NOT NULL,
    hash VARCHAR(255) NOT NULL,
    used TINYINT(1) NOT NULL DEFAULT 0,
    revoked TINYINT(1) NOT NULL DEFAULT 0,
    expires_at TIMESTAMP(3) NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_user_id (user_id),
    KEY idx_family_id (family_id),
    KEY idx_hash (hash),
    KEY idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- User Sessions (login sessions)
CREATE TABLE user_sessions (
    id BIGINT NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_family BIGINT NOT NULL,
    device VARCHAR(255),
    ip VARCHAR(45),
    location VARCHAR(255),
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    last_active_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at TIMESTAMP(3) NOT NULL,

    KEY idx_user_id (user_id),
    KEY idx_token_family (token_family),
    KEY idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- OAuth Connections
CREATE TABLE oauth_connections (
    id BIGINT NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT,
    expires_at TIMESTAMP(3) NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    UNIQUE KEY uk_provider_user (provider, provider_user_id),
    KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Provider & Usage Tables

```sql
-- User Provider Keys (BYOK)
CREATE TABLE user_provider_keys (
    id BIGINT NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    encrypted_key TEXT NOT NULL,
    key_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    last_used_at TIMESTAMP(3) NULL,

    UNIQUE KEY uk_user_provider (user_id, provider_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Organization Provider Config
CREATE TABLE org_provider_configs (
    id BIGINT NOT NULL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    encrypted_key TEXT NOT NULL,
    key_id VARCHAR(255) NOT NULL,
    rate_limit JSON,
    allow_user_override TINYINT(1) NOT NULL DEFAULT 1,
    usage_tracking TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    UNIQUE KEY uk_org_provider (org_id, provider_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Usage Records
CREATE TABLE usage_records (
    id BIGINT NOT NULL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    session_id BIGINT NULL,
    provider_id VARCHAR(100) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    tokens_input INT NOT NULL,
    tokens_output INT NOT NULL,
    tokens_cache_read INT NOT NULL DEFAULT 0,
    tokens_cache_write INT NOT NULL DEFAULT 0,
    cost_cents INT NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_org_created (org_id, created_at),
    KEY idx_user_created (user_id, created_at),
    KEY idx_session_id (session_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Audit Logs
CREATE TABLE audit_logs (
    id BIGINT NOT NULL PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id BIGINT NULL,
    metadata JSON NOT NULL,
    ip VARCHAR(45),
    user_agent TEXT,
    status VARCHAR(50) NOT NULL,
    error_code VARCHAR(100),
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    KEY idx_org_created (org_id, created_at),
    KEY idx_user_created (user_id, created_at),
    KEY idx_action (action),
    KEY idx_resource (resource, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## Application-Level Referential Integrity

### Validation on Insert/Update

```typescript
// Validate parent exists before insert
class SessionRepository {
  async create(input: CreateSessionInput): Promise<Session> {
    // Validate project exists
    const project = await this.db.query<Project>`
      SELECT id, workspace_id FROM projects WHERE id = ${input.projectId}
    `.first()

    if (!project) {
      throw new NotFoundError("Project not found", "PROJECT_NOT_FOUND")
    }

    // Validate workspace belongs to org (for tenant isolation)
    const workspace = await this.db.query<Workspace>`
      SELECT id FROM workspaces
      WHERE id = ${project.workspaceId} AND org_id = ${this.ctx.orgId}
    `.first()

    if (!workspace) {
      throw new ForbiddenError("Access denied", "WORKSPACE_ACCESS_DENIED")
    }

    // Insert session
    const id = this.idGen.nextId()
    await this.db.query`
      INSERT INTO sessions (id, project_id, user_id, title, model_provider, model_id)
      VALUES (${id}, ${input.projectId}, ${this.ctx.userId}, ${input.title},
              ${input.modelProvider}, ${input.modelId})
    `

    return this.findById(id)
  }
}
```

### Cascading Deletes

```typescript
// Manual cascade delete (no FK constraints)
class SessionRepository {
  async delete(id: bigint): Promise<void> {
    // Verify ownership
    const session = await this.findById(id)
    if (!session) {
      throw new NotFoundError("Session not found")
    }

    // Delete in order: parts → messages → diffs → session
    // Use transaction for atomicity
    await this.db.transaction(async (tx) => {
      // Get all message IDs for this session
      const messageIds = await tx.query<{ id: bigint }>`
        SELECT id FROM messages WHERE session_id = ${id}
      `.all()

      if (messageIds.length > 0) {
        const ids = messageIds.map(m => m.id)

        // Delete parts for all messages
        await tx.query`
          DELETE FROM message_parts WHERE message_id IN (${ids})
        `

        // Delete messages
        await tx.query`
          DELETE FROM messages WHERE session_id = ${id}
        `
      }

      // Delete diffs
      await tx.query`
        DELETE FROM session_diffs WHERE session_id = ${id}
      `

      // Delete session
      await tx.query`
        DELETE FROM sessions WHERE id = ${id}
      `
    })
  }
}
```

### Orphan Cleanup Job

```typescript
// Background job to clean orphaned records
class OrphanCleanupJob {
  async run(): Promise<CleanupResult> {
    const result: CleanupResult = {
      messageParts: 0,
      messages: 0,
      diffs: 0,
      sessions: 0,
    }

    // Find and delete orphaned message_parts
    const orphanedParts = await this.db.query`
      DELETE mp FROM message_parts mp
      LEFT JOIN messages m ON mp.message_id = m.id
      WHERE m.id IS NULL
    `
    result.messageParts = orphanedParts.affectedRows

    // Find and delete orphaned messages
    const orphanedMessages = await this.db.query`
      DELETE m FROM messages m
      LEFT JOIN sessions s ON m.session_id = s.id
      WHERE s.id IS NULL
    `
    result.messages = orphanedMessages.affectedRows

    // Find and delete orphaned session_diffs
    const orphanedDiffs = await this.db.query`
      DELETE sd FROM session_diffs sd
      LEFT JOIN sessions s ON sd.session_id = s.id
      WHERE s.id IS NULL
    `
    result.diffs = orphanedDiffs.affectedRows

    // Find and delete orphaned sessions (no project)
    const orphanedSessions = await this.db.query`
      DELETE s FROM sessions s
      LEFT JOIN projects p ON s.project_id = p.id
      WHERE p.id IS NULL
    `
    result.sessions = orphanedSessions.affectedRows

    return result
  }
}

// Schedule: run every hour
schedule.every("1 hour", () => orphanCleanupJob.run())
```

## Query Patterns

### Efficient Pagination with BIGINT

```typescript
// Cursor-based pagination (efficient with BIGINT)
async function listSessions(
  userId: bigint,
  cursor?: bigint,
  limit: number = 50
): Promise<PaginatedResult<Session>> {
  // Snowflake IDs are time-ordered, so we can use them directly
  const sessions = await db.query<Session>`
    SELECT * FROM sessions
    WHERE user_id = ${userId}
      ${cursor ? sql`AND id < ${cursor}` : sql``}
    ORDER BY id DESC
    LIMIT ${limit + 1}
  `.all()

  const hasMore = sessions.length > limit
  if (hasMore) sessions.pop()

  return {
    data: sessions,
    pagination: {
      cursor: hasMore ? sessions[sessions.length - 1].id.toString() : undefined,
      hasMore,
    },
  }
}
```

### Batch Loading with IN Clause

```typescript
// Efficient batch loading
async function getMessagesWithParts(sessionId: bigint): Promise<MessageWithParts[]> {
  // Load messages
  const messages = await db.query<Message>`
    SELECT * FROM messages
    WHERE session_id = ${sessionId}
    ORDER BY created_at ASC
  `.all()

  if (messages.length === 0) return []

  // Batch load all parts
  const messageIds = messages.map(m => m.id)
  const parts = await db.query<MessagePart>`
    SELECT * FROM message_parts
    WHERE message_id IN (${messageIds})
    ORDER BY message_id, sort_order
  `.all()

  // Group parts by message
  const partsByMessage = new Map<bigint, MessagePart[]>()
  for (const part of parts) {
    const list = partsByMessage.get(part.message_id) || []
    list.push(part)
    partsByMessage.set(part.message_id, list)
  }

  // Combine
  return messages.map(msg => ({
    ...msg,
    parts: partsByMessage.get(msg.id) || [],
  }))
}
```

### Multi-Tenant Queries

```typescript
// All queries scoped to organization
class TenantScopedRepository<T> {
  constructor(
    protected db: Database,
    protected ctx: TenantContext
  ) {}

  // Helper to add org scope through joins
  protected async withOrgScope(
    table: string,
    id: bigint
  ): Promise<boolean> {
    // Different paths to org based on table
    const scopeQueries: Record<string, string> = {
      sessions: `
        SELECT 1 FROM sessions s
        JOIN projects p ON s.project_id = p.id
        JOIN workspaces w ON p.workspace_id = w.id
        WHERE s.id = ? AND w.org_id = ?
      `,
      messages: `
        SELECT 1 FROM messages m
        JOIN sessions s ON m.session_id = s.id
        JOIN projects p ON s.project_id = p.id
        JOIN workspaces w ON p.workspace_id = w.id
        WHERE m.id = ? AND w.org_id = ?
      `,
      projects: `
        SELECT 1 FROM projects p
        JOIN workspaces w ON p.workspace_id = w.id
        WHERE p.id = ? AND w.org_id = ?
      `,
      workspaces: `
        SELECT 1 FROM workspaces WHERE id = ? AND org_id = ?
      `,
    }

    const query = scopeQueries[table]
    if (!query) {
      throw new Error(`Unknown table: ${table}`)
    }

    const result = await this.db.execute(query, [id, this.ctx.orgId])
    return result.length > 0
  }
}
```

## Index Optimization

### Covering Indexes

```sql
-- Covering index for common query patterns
-- Sessions by user with status filter
CREATE INDEX idx_sessions_user_status_created
ON sessions (user_id, status, created_at DESC, id, title, model_provider, model_id);

-- Messages with metadata for listing
CREATE INDEX idx_messages_session_created
ON messages (session_id, created_at, id, role);
```

### JSON Indexing

```sql
-- Virtual columns for JSON fields (MySQL 5.7+)
ALTER TABLE sessions
ADD COLUMN summary_files INT
GENERATED ALWAYS AS (JSON_EXTRACT(summary, '$.files')) VIRTUAL;

CREATE INDEX idx_sessions_summary_files ON sessions (summary_files);

-- Or use JSON_VALUE in MySQL 8.0+
CREATE INDEX idx_sessions_plan
ON organizations ((CAST(JSON_VALUE(settings, '$.plan') AS CHAR(50))));
```

### Composite Index Strategy

```sql
-- Order matters: equality → range → sort
-- Good: WHERE user_id = ? AND status = ? ORDER BY created_at DESC
CREATE INDEX idx_sessions_user_status_created
ON sessions (user_id, status, created_at DESC);

-- For time-range queries with org scope
CREATE INDEX idx_usage_org_created
ON usage_records (org_id, created_at);

-- For prefix searches on API keys
CREATE INDEX idx_api_keys_prefix
ON api_keys (prefix(8));
```

## Connection Management

### Connection Pool Configuration

```typescript
import mysql from "mysql2/promise"

const pool = mysql.createPool({
  host: process.env.MYSQL_HOST,
  port: parseInt(process.env.MYSQL_PORT || "3306"),
  user: process.env.MYSQL_USER,
  password: process.env.MYSQL_PASSWORD,
  database: process.env.MYSQL_DATABASE,

  // Pool settings
  connectionLimit: 20,
  queueLimit: 0,
  waitForConnections: true,

  // Timeouts
  connectTimeout: 10000,
  acquireTimeout: 10000,

  // Keep-alive
  enableKeepAlive: true,
  keepAliveInitialDelay: 30000,

  // Character set
  charset: "utf8mb4",

  // Timezone
  timezone: "+00:00",

  // Named placeholders
  namedPlaceholders: true,
})

// Health check
async function checkHealth(): Promise<boolean> {
  try {
    const conn = await pool.getConnection()
    await conn.ping()
    conn.release()
    return true
  } catch {
    return false
  }
}
```

### Read/Write Splitting

```typescript
interface DatabaseConfig {
  writer: mysql.PoolOptions
  readers: mysql.PoolOptions[]
}

class ReadWritePool {
  private writer: mysql.Pool
  private readers: mysql.Pool[]
  private readerIndex = 0

  constructor(config: DatabaseConfig) {
    this.writer = mysql.createPool(config.writer)
    this.readers = config.readers.map(r => mysql.createPool(r))
  }

  // Get writer for INSERT/UPDATE/DELETE
  getWriter(): mysql.Pool {
    return this.writer
  }

  // Round-robin reader selection
  getReader(): mysql.Pool {
    if (this.readers.length === 0) {
      return this.writer
    }
    const reader = this.readers[this.readerIndex]
    this.readerIndex = (this.readerIndex + 1) % this.readers.length
    return reader
  }

  // Smart routing based on query
  async query<T>(sql: string, params?: unknown[]): Promise<T[]> {
    const isWrite = /^\s*(INSERT|UPDATE|DELETE|REPLACE)/i.test(sql)
    const pool = isWrite ? this.getWriter() : this.getReader()
    const [rows] = await pool.execute(sql, params)
    return rows as T[]
  }
}
```

## Sharding Strategy

### Shard Key Selection

```typescript
// Shard by organization for tenant isolation
interface ShardConfig {
  shardKey: "org_id"
  shardCount: 16
  shardMap: Map<number, DatabaseConfig> // shard_id → connection
}

function getShardId(orgId: bigint, shardCount: number): number {
  // Consistent hashing
  return Number(orgId % BigInt(shardCount))
}

class ShardedDatabase {
  private shards: Map<number, ReadWritePool>

  constructor(config: ShardConfig) {
    this.shards = new Map()
    for (const [shardId, dbConfig] of config.shardMap) {
      this.shards.set(shardId, new ReadWritePool(dbConfig))
    }
  }

  getPool(orgId: bigint): ReadWritePool {
    const shardId = getShardId(orgId, this.shards.size)
    const pool = this.shards.get(shardId)
    if (!pool) {
      throw new Error(`Shard ${shardId} not configured`)
    }
    return pool
  }

  // Cross-shard query (fan-out)
  async queryAll<T>(sql: string, params?: unknown[]): Promise<T[]> {
    const results = await Promise.all(
      Array.from(this.shards.values()).map(pool =>
        pool.query<T>(sql, params)
      )
    )
    return results.flat()
  }
}
```

### Schema Per Shard

```sql
-- Each shard has identical schema
-- Shard 0: opencode_shard_0
-- Shard 1: opencode_shard_1
-- ...

-- Global tables (not sharded) in separate database
-- opencode_global: organizations, users, api_keys
```

## Migration from UUID

### Migration Script

```typescript
// Add bigint columns alongside UUID
async function migrationStep1(): Promise<void> {
  await db.query`
    ALTER TABLE sessions
    ADD COLUMN id_new BIGINT NULL AFTER id,
    ADD COLUMN project_id_new BIGINT NULL AFTER project_id,
    ADD COLUMN user_id_new BIGINT NULL AFTER user_id
  `
}

// Populate bigint columns
async function migrationStep2(): Promise<void> {
  // Generate mapping: UUID → BIGINT
  const idGen = new SnowflakeGenerator(0)

  // Process in batches
  let cursor: string | null = null
  while (true) {
    const sessions = await db.query<Session>`
      SELECT id, project_id, user_id FROM sessions
      WHERE id_new IS NULL
      ${cursor ? sql`AND id > ${cursor}` : sql``}
      ORDER BY id
      LIMIT 1000
    `.all()

    if (sessions.length === 0) break

    for (const session of sessions) {
      const newId = idGen.nextId()
      await db.query`
        UPDATE sessions SET id_new = ${newId}
        WHERE id = ${session.id}
      `
    }

    cursor = sessions[sessions.length - 1].id
  }
}

// Swap columns
async function migrationStep3(): Promise<void> {
  await db.query`
    ALTER TABLE sessions
    DROP COLUMN id,
    CHANGE COLUMN id_new id BIGINT NOT NULL,
    ADD PRIMARY KEY (id)
  `
}
```

## Performance Considerations

### Batch Inserts

```typescript
// Bulk insert for message parts
async function insertParts(parts: MessagePart[]): Promise<void> {
  if (parts.length === 0) return

  const values = parts.map(p => [
    p.id,
    p.message_id,
    p.type,
    JSON.stringify(p.content),
    p.sort_order,
  ])

  await db.query`
    INSERT INTO message_parts (id, message_id, type, content, sort_order)
    VALUES ${values}
  `
}
```

### Query Optimization Tips

```sql
-- Use STRAIGHT_JOIN to force join order when optimizer chooses poorly
SELECT STRAIGHT_JOIN s.*
FROM sessions s
JOIN projects p ON s.project_id = p.id
JOIN workspaces w ON p.workspace_id = w.id
WHERE w.org_id = ?;

-- Use index hints if needed
SELECT * FROM sessions USE INDEX (idx_user_status_created)
WHERE user_id = ? AND status = 'active'
ORDER BY created_at DESC;

-- Avoid SELECT * in production
SELECT id, title, status, created_at FROM sessions WHERE user_id = ?;
```

### Monitoring Queries

```sql
-- Find slow queries
SELECT * FROM performance_schema.events_statements_summary_by_digest
ORDER BY SUM_TIMER_WAIT DESC
LIMIT 10;

-- Check index usage
SELECT * FROM sys.schema_unused_indexes;

-- Table sizes
SELECT
    table_name,
    ROUND(data_length / 1024 / 1024, 2) AS data_mb,
    ROUND(index_length / 1024 / 1024, 2) AS index_mb
FROM information_schema.tables
WHERE table_schema = 'opencode'
ORDER BY data_length DESC;
```
