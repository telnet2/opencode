# Storage & Data Persistence

## Overview

The server-side deployment replaces the file-based storage system with a distributed storage architecture optimized for multi-tenancy, scalability, and reliability.

## Storage Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
     ┌────────▼────────┐ ┌────▼────┐ ┌────────▼────────┐
     │   PostgreSQL    │ │  Redis  │ │  Object Store   │
     │  (Primary DB)   │ │ (Cache) │ │  (Files/Blobs)  │
     └─────────────────┘ └─────────┘ └─────────────────┘
              │
              ▼
     ┌─────────────────┐
     │    Replicas     │
     │  (Read scaling) │
     └─────────────────┘
```

## PostgreSQL Schema

### Core Tables

```sql
-- Organizations (tenants)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    avatar_url VARCHAR(500),
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    preferences JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ,
    UNIQUE(org_id, email)
);

-- Workspaces
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    git_config JSONB,
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(1000) NOT NULL,
    git_commit VARCHAR(40),
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sessions
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    parent_id UUID REFERENCES sessions(id),
    title VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    model_provider VARCHAR(100) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    summary JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- Messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Message Parts (text, tools, files, etc.)
CREATE TABLE message_parts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    content JSONB NOT NULL,
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Session Diffs (code changes)
CREATE TABLE session_diffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id),
    file_path VARCHAR(1000) NOT NULL,
    diff_content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Authentication Tables

```sql
-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    prefix VARCHAR(20) NOT NULL,
    hash VARCHAR(255) NOT NULL,
    scopes VARCHAR(50)[] NOT NULL DEFAULT '{}',
    rate_limit JSONB,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refresh Tokens
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL,
    hash VARCHAR(255) NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User Sessions (login sessions)
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_family UUID NOT NULL,
    device VARCHAR(255),
    ip INET,
    location VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- OAuth Connections
CREATE TABLE oauth_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
```

### Provider & Usage Tables

```sql
-- User Provider Keys (BYOK)
CREATE TABLE user_provider_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id VARCHAR(100) NOT NULL,
    encrypted_key TEXT NOT NULL,
    key_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    UNIQUE(user_id, provider_id)
);

-- Organization Provider Config
CREATE TABLE org_provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider_id VARCHAR(100) NOT NULL,
    encrypted_key TEXT NOT NULL,
    key_id VARCHAR(255) NOT NULL,
    rate_limit JSONB,
    allow_user_override BOOLEAN NOT NULL DEFAULT TRUE,
    usage_tracking BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, provider_id)
);

-- Usage Tracking
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    session_id UUID REFERENCES sessions(id),
    provider_id VARCHAR(100) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    tokens_input INTEGER NOT NULL,
    tokens_output INTEGER NOT NULL,
    tokens_cache_read INTEGER DEFAULT 0,
    tokens_cache_write INTEGER DEFAULT 0,
    cost_cents INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit Logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL,
    user_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}',
    ip INET,
    user_agent TEXT,
    status VARCHAR(50) NOT NULL,
    error_code VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Indexes

```sql
-- Performance indexes
CREATE INDEX idx_sessions_project_id ON sessions(project_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_message_parts_message_id ON message_parts(message_id);
CREATE INDEX idx_session_diffs_session_id ON session_diffs(session_id);

-- Multi-tenant indexes
CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_workspaces_org_id ON workspaces(org_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(prefix);

-- Usage and audit indexes
CREATE INDEX idx_usage_records_org_id_created ON usage_records(org_id, created_at DESC);
CREATE INDEX idx_usage_records_user_id_created ON usage_records(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_org_id_created ON audit_logs(org_id, created_at DESC);
CREATE INDEX idx_audit_logs_user_id_created ON audit_logs(user_id, created_at DESC);

-- Full-text search
CREATE INDEX idx_sessions_title_fts ON sessions USING gin(to_tsvector('english', title));
```

## Redis Data Structures

### Caching Strategy

```typescript
interface CacheConfig {
  // Session metadata cache
  session: {
    key: (id: string) => `session:${id}`,
    ttl: 3600,         // 1 hour
  },
  // User preferences cache
  user: {
    key: (id: string) => `user:${id}`,
    ttl: 1800,         // 30 minutes
  },
  // Provider config cache
  provider: {
    key: (orgId: string, providerId: string) => `provider:${orgId}:${providerId}`,
    ttl: 300,          // 5 minutes
  },
  // Rate limit counters
  rateLimit: {
    key: (id: string, window: string) => `rl:${id}:${window}`,
    ttl: 60,           // 1 minute
  },
}
```

### Real-time Data

```typescript
// Active session tracking
interface ActiveSession {
  key: `active:session:${sessionId}`,
  value: {
    userId: string
    status: "idle" | "processing" | "streaming"
    lastActivity: number
    currentMessageId?: string
  },
  ttl: 3600
}

// SSE connection tracking
interface SSEConnection {
  key: `sse:user:${userId}`,
  value: Set<connectionId>,
  ttl: 86400
}

// Pub/Sub channels
const channels = {
  session: (id: string) => `events:session:${id}`,
  user: (id: string) => `events:user:${id}`,
  workspace: (id: string) => `events:workspace:${id}`,
}
```

### Job Queue

```typescript
// Background job queues using Redis Streams
interface JobQueue {
  // Session compaction jobs
  compaction: {
    stream: "jobs:compaction",
    group: "compaction-workers",
  },
  // Usage aggregation
  usage: {
    stream: "jobs:usage",
    group: "usage-workers",
  },
  // Cleanup expired sessions
  cleanup: {
    stream: "jobs:cleanup",
    group: "cleanup-workers",
  },
}
```

## Object Storage

### File Organization

```
bucket/
├── workspaces/
│   └── {workspaceId}/
│       └── {projectId}/
│           ├── files/           # Project files
│           │   └── {hash}
│           └── snapshots/       # Git snapshots
│               └── {snapshotId}
├── sessions/
│   └── {sessionId}/
│       ├── attachments/         # User uploads
│       │   └── {attachmentId}
│       └── artifacts/           # Generated files
│           └── {artifactId}
├── exports/
│   └── {exportId}/              # Session exports
│       └── export.zip
└── avatars/
    └── {userId}/
        └── avatar.{ext}
```

### Storage Operations

```typescript
interface ObjectStorage {
  // Upload file
  upload(key: string, content: Buffer | Stream, options?: UploadOptions): Promise<string>

  // Download file
  download(key: string): Promise<Buffer>

  // Get signed URL for client-side download
  getSignedUrl(key: string, expiresIn: number): Promise<string>

  // Delete file
  delete(key: string): Promise<void>

  // List files by prefix
  list(prefix: string): Promise<StorageObject[]>
}

interface UploadOptions {
  contentType?: string
  metadata?: Record<string, string>
  acl?: "private" | "public-read"
}
```

### Content-Addressable Storage

```typescript
// Store files by content hash for deduplication
async function storeFile(
  workspaceId: string,
  projectId: string,
  content: Buffer
): Promise<string> {
  const hash = crypto.createHash("sha256").update(content).digest("hex")
  const key = `workspaces/${workspaceId}/${projectId}/files/${hash}`

  // Check if already exists
  const exists = await storage.exists(key)
  if (!exists) {
    await storage.upload(key, content)
  }

  return hash
}
```

## Data Access Layer

### Repository Pattern

```typescript
// Base repository with tenant scoping
abstract class BaseRepository<T> {
  constructor(
    protected db: Database,
    protected ctx: TenantContext
  ) {}

  protected get orgId() {
    return this.ctx.orgId
  }

  protected get userId() {
    return this.ctx.userId
  }
}

// Session repository
class SessionRepository extends BaseRepository<Session> {
  async findById(id: string): Promise<Session | null> {
    return this.db.query<Session>`
      SELECT s.*
      FROM sessions s
      JOIN projects p ON s.project_id = p.id
      JOIN workspaces w ON p.workspace_id = w.id
      WHERE s.id = ${id}
        AND w.org_id = ${this.orgId}
    `.first()
  }

  async create(input: CreateSessionInput): Promise<Session> {
    return this.db.query<Session>`
      INSERT INTO sessions (
        project_id, user_id, title, model_provider, model_id
      ) VALUES (
        ${input.projectId},
        ${this.userId},
        ${input.title},
        ${input.modelProvider},
        ${input.modelId}
      )
      RETURNING *
    `.first()
  }

  async listByUser(options: ListOptions): Promise<Session[]> {
    return this.db.query<Session>`
      SELECT s.*
      FROM sessions s
      JOIN projects p ON s.project_id = p.id
      JOIN workspaces w ON p.workspace_id = w.id
      WHERE w.org_id = ${this.orgId}
        AND s.user_id = ${this.userId}
      ORDER BY s.created_at DESC
      LIMIT ${options.limit}
      OFFSET ${options.offset}
    `.all()
  }
}
```

### Caching Layer

```typescript
// Cache-aside pattern
class CachedSessionRepository {
  constructor(
    private repo: SessionRepository,
    private cache: Redis,
    private ctx: TenantContext
  ) {}

  async findById(id: string): Promise<Session | null> {
    const cacheKey = `session:${id}`

    // Try cache first
    const cached = await this.cache.get<Session>(cacheKey)
    if (cached) return cached

    // Fetch from database
    const session = await this.repo.findById(id)
    if (session) {
      await this.cache.set(cacheKey, session, { ex: 3600 })
    }

    return session
  }

  async update(id: string, input: UpdateSessionInput): Promise<Session> {
    const session = await this.repo.update(id, input)

    // Invalidate cache
    await this.cache.del(`session:${id}`)

    // Publish update event
    await this.cache.publish(`events:session:${id}`, {
      type: "session.updated",
      session,
    })

    return session
  }
}
```

## Migration Strategy

### From File-Based to Database

```typescript
// Migration script for existing data
async function migrateFromFiles(
  sourceDir: string,
  targetDb: Database
): Promise<MigrationResult> {
  const result: MigrationResult = {
    sessions: 0,
    messages: 0,
    parts: 0,
    errors: [],
  }

  // Read existing sessions
  const sessionFiles = await glob(`${sourceDir}/session/**/*.json`)

  for (const file of sessionFiles) {
    try {
      const data = JSON.parse(await fs.readFile(file, "utf-8"))

      // Map to new schema
      const session = mapLegacySession(data)
      await targetDb.sessions.create(session)
      result.sessions++

      // Migrate messages
      const messageFiles = await glob(`${sourceDir}/message/${data.id}/*.json`)
      for (const msgFile of messageFiles) {
        const msgData = JSON.parse(await fs.readFile(msgFile, "utf-8"))
        const message = mapLegacyMessage(msgData)
        await targetDb.messages.create(message)
        result.messages++

        // Migrate parts
        const partFiles = await glob(`${sourceDir}/part/${msgData.id}/*.json`)
        for (const partFile of partFiles) {
          const partData = JSON.parse(await fs.readFile(partFile, "utf-8"))
          const part = mapLegacyPart(partData)
          await targetDb.messageParts.create(part)
          result.parts++
        }
      }
    } catch (error) {
      result.errors.push({ file, error: error.message })
    }
  }

  return result
}
```

## Backup & Recovery

### Backup Strategy

```typescript
interface BackupConfig {
  // PostgreSQL backups
  database: {
    schedule: "0 */6 * * *",    // Every 6 hours
    retention: 30,              // Days
    method: "pg_dump" | "wal",
  },
  // Object storage
  objects: {
    versioning: true,
    retention: 90,              // Days
    replication: "cross-region",
  },
}
```

### Point-in-Time Recovery

```sql
-- Enable WAL archiving for PITR
ALTER SYSTEM SET archive_mode = on;
ALTER SYSTEM SET archive_command = 'aws s3 cp %p s3://backups/wal/%f';
ALTER SYSTEM SET wal_level = replica;
```

## Data Retention

### Retention Policies

```typescript
interface RetentionPolicy {
  // Session data
  sessions: {
    active: "indefinite",
    archived: 365,           // Days
    deleted: 30,             // Soft delete grace period
  },
  // Usage records
  usage: {
    detailed: 90,            // Days
    aggregated: 730,         // 2 years
  },
  // Audit logs
  audit: {
    security: 730,           // 2 years
    general: 90,             // Days
  },
}
```

### Cleanup Jobs

```typescript
// Scheduled cleanup job
async function cleanupExpiredData(): Promise<void> {
  const cutoff = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)

  // Delete soft-deleted sessions
  await db.query`
    DELETE FROM sessions
    WHERE status = 'deleted'
      AND updated_at < ${cutoff}
  `

  // Archive old usage records
  await db.query`
    INSERT INTO usage_records_archive
    SELECT * FROM usage_records
    WHERE created_at < ${cutoff}
  `

  await db.query`
    DELETE FROM usage_records
    WHERE created_at < ${cutoff}
  `

  // Clean up orphaned object storage
  await cleanupOrphanedObjects()
}
```

## Performance Optimization

### Query Optimization

```typescript
// Efficient message loading with pagination
async function loadMessages(
  sessionId: string,
  cursor?: string,
  limit: number = 50
): Promise<{ messages: MessageWithParts[]; nextCursor?: string }> {
  const messages = await db.query<Message>`
    SELECT m.*,
           json_agg(
             json_build_object(
               'id', mp.id,
               'type', mp.type,
               'content', mp.content,
               'order', mp.sort_order
             ) ORDER BY mp.sort_order
           ) as parts
    FROM messages m
    LEFT JOIN message_parts mp ON mp.message_id = m.id
    WHERE m.session_id = ${sessionId}
      ${cursor ? sql`AND m.id < ${cursor}` : sql``}
    GROUP BY m.id
    ORDER BY m.created_at DESC
    LIMIT ${limit + 1}
  `.all()

  const hasMore = messages.length > limit
  if (hasMore) messages.pop()

  return {
    messages,
    nextCursor: hasMore ? messages[messages.length - 1].id : undefined,
  }
}
```

### Connection Pooling

```typescript
// PostgreSQL connection pool config
const poolConfig = {
  min: 5,
  max: 20,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 5000,
  // Read replicas for queries
  replicas: [
    { host: "replica-1.db.internal", port: 5432 },
    { host: "replica-2.db.internal", port: 5432 },
  ],
}
```
