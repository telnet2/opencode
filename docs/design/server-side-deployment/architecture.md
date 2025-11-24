# System Architecture

## Component Overview

### 1. API Gateway Layer

**Purpose**: Entry point for all client requests, handling routing, rate limiting, and initial authentication.

```typescript
interface GatewayConfig {
  rateLimiting: {
    requests: number      // per window
    window: "second" | "minute" | "hour"
    byUser: boolean       // per-user limits
    byOrg: boolean        // per-org limits
  }
  cors: {
    origins: string[]
    credentials: boolean
  }
  tls: {
    minVersion: "1.2" | "1.3"
    ciphers: string[]
  }
}
```

**Responsibilities**:
- TLS termination
- Request routing
- Rate limiting (token bucket algorithm)
- Request/response logging
- CORS handling
- Request ID injection

### 2. API Server (Hono)

**Purpose**: Core business logic, session management, and LLM orchestration.

```typescript
// Server initialization with multi-tenant support
export function createServer(config: ServerConfig) {
  const app = new Hono()

  // Middleware stack
  app.use(requestId())
  app.use(logger())
  app.use(authenticate())      // JWT validation
  app.use(tenantContext())     // Inject user/org context
  app.use(rateLimitMiddleware())

  // Routes
  app.route("/api/v1/sessions", sessionRoutes)
  app.route("/api/v1/projects", projectRoutes)
  app.route("/api/v1/workspaces", workspaceRoutes)
  app.route("/api/v1/providers", providerRoutes)

  return app
}
```

**Key Modifications from Current Architecture**:

| Current | Server-Side |
|---------|-------------|
| `Instance.provide({ directory })` | `TenantContext.provide({ userId, orgId, workspaceId })` |
| File-based storage | Database + Object storage |
| Single event bus | Redis Pub/Sub |
| Local Git operations | Remote Git service integration |

### 3. Session Orchestrator

**Purpose**: Manages AI sessions, tool execution, and streaming responses.

```typescript
interface SessionOrchestrator {
  // Create new session in workspace
  create(ctx: TenantContext, input: CreateSessionInput): Promise<Session>

  // Send message and stream response
  chat(ctx: TenantContext, sessionId: string, message: Message): AsyncGenerator<StreamEvent>

  // Execute tool with sandboxing
  executeTool(ctx: TenantContext, sessionId: string, tool: ToolCall): Promise<ToolResult>

  // Abort running session
  abort(ctx: TenantContext, sessionId: string): Promise<void>
}
```

**Session Lifecycle**:
```
┌─────────┐     ┌──────────┐     ┌─────────┐     ┌───────────┐
│ Created │ ──▶ │ Active   │ ──▶ │ Idle    │ ──▶ │ Archived  │
└─────────┘     └──────────┘     └─────────┘     └───────────┘
                     │                │
                     ▼                ▼
                ┌──────────┐    ┌──────────┐
                │ Aborted  │    │ Expired  │
                └──────────┘    └──────────┘
```

### 4. Tool Execution Engine

**Purpose**: Sandboxed execution of code tools (Bash, file operations, etc.)

```typescript
interface ToolExecutionConfig {
  sandbox: {
    type: "docker" | "firecracker" | "gvisor"
    image: string
    resources: {
      cpuLimit: string      // "1000m"
      memoryLimit: string   // "512Mi"
      diskLimit: string     // "1Gi"
      timeout: number       // ms
    }
    network: {
      enabled: boolean
      egress: string[]      // allowed domains
    }
  }
  workspace: {
    mount: string           // /workspace
    readonly: string[]      // paths
  }
}
```

**Execution Flow**:
```
Tool Request ──▶ Validate ──▶ Acquire Sandbox ──▶ Mount Workspace
                                                        │
                                                        ▼
Tool Response ◀── Cleanup ◀── Capture Output ◀── Execute Command
```

### 5. Provider Gateway

**Purpose**: Manages LLM provider connections with key rotation and failover.

```typescript
interface ProviderGateway {
  // Route request to appropriate provider
  route(ctx: TenantContext, request: LLMRequest): Promise<LLMResponse>

  // Stream response from provider
  stream(ctx: TenantContext, request: LLMRequest): AsyncGenerator<LLMChunk>

  // Get available models for user
  models(ctx: TenantContext): Promise<Model[]>
}

interface ProviderConfig {
  anthropic: {
    apiKey: string | { vault: string }
    baseUrl?: string
    rateLimit: RateLimit
  }
  openai: {
    apiKey: string | { vault: string }
    organization?: string
    rateLimit: RateLimit
  }
  // ... other providers
}
```

**Key Management**:
- Organization-level keys stored in Vault/KMS
- User BYOK (Bring Your Own Key) with encryption at rest
- Automatic key rotation support
- Usage attribution per key

## Data Models

### Tenant Hierarchy

```
Organization
├── Users (members)
├── Teams
├── API Keys
├── Provider Configs
└── Workspaces
    ├── Projects
    │   ├── Git Config
    │   └── Project Settings
    └── Sessions
        ├── Messages
        │   └── Parts
        └── Diffs
```

### Core Entities

```typescript
// Organization - top-level tenant
interface Organization {
  id: string
  name: string
  slug: string
  plan: "free" | "team" | "enterprise"
  settings: OrgSettings
  createdAt: Date
  updatedAt: Date
}

// User within organization
interface User {
  id: string
  orgId: string
  email: string
  name: string
  role: "owner" | "admin" | "member"
  preferences: UserPreferences
  createdAt: Date
  lastActiveAt: Date
}

// Workspace - isolated environment
interface Workspace {
  id: string
  orgId: string
  name: string
  description?: string
  gitConfig?: {
    provider: "github" | "gitlab" | "bitbucket"
    repoUrl: string
    branch: string
    credentials: EncryptedCredentials
  }
  settings: WorkspaceSettings
  createdAt: Date
  updatedAt: Date
}

// Project within workspace
interface Project {
  id: string
  workspaceId: string
  name: string
  path: string
  gitCommit?: string
  settings: ProjectSettings
  createdAt: Date
  updatedAt: Date
}

// Session (conversation)
interface Session {
  id: string
  projectId: string
  userId: string
  title: string
  status: SessionStatus
  model: {
    providerId: string
    modelId: string
  }
  summary?: SessionSummary
  createdAt: Date
  updatedAt: Date
  expiresAt?: Date
}

// Message within session
interface Message {
  id: string
  sessionId: string
  role: "user" | "assistant" | "system"
  content: MessageContent
  metadata: MessageMetadata
  createdAt: Date
}

// Message part (text, tool, file, etc.)
interface MessagePart {
  id: string
  messageId: string
  type: PartType
  content: PartContent
  order: number
  createdAt: Date
}
```

## Request Flow

### Chat Request Flow

```
1. Client sends POST /api/v1/sessions/:id/messages
   │
2. API Gateway validates JWT, applies rate limit
   │
3. API Server receives request
   │  ├── Validate session ownership
   │  ├── Load session context from DB
   │  └── Check user quota
   │
4. Session Orchestrator processes message
   │  ├── Build prompt with history
   │  ├── Select provider/model
   │  └── Apply system prompts
   │
5. Provider Gateway streams to LLM
   │  ├── Apply org/user API key
   │  ├── Track token usage
   │  └── Handle retries/failover
   │
6. Tool Execution (if needed)
   │  ├── Spawn sandboxed container
   │  ├── Mount workspace files
   │  ├── Execute tool
   │  └── Capture output
   │
7. Stream response to client
   │  ├── Publish events to Redis
   │  ├── Persist to database
   │  └── SSE to client
   │
8. Update usage metrics
```

### Event Distribution

```typescript
// Cross-instance event distribution
interface EventDistributor {
  // Publish event to all subscribers
  publish(channel: string, event: Event): Promise<void>

  // Subscribe to events for user/session
  subscribe(channel: string, handler: EventHandler): Unsubscribe
}

// Redis Pub/Sub channels
const channels = {
  session: (sessionId: string) => `session:${sessionId}`,
  user: (userId: string) => `user:${userId}`,
  workspace: (workspaceId: string) => `workspace:${workspaceId}`,
}
```

**SSE Connection Management**:
```typescript
// Server-Sent Events with Redis coordination
app.get("/api/v1/events", async (c) => {
  const { userId, sessionId } = c.get("tenant")

  return streamSSE(c, async (stream) => {
    // Subscribe to user's events
    const unsub = await eventDistributor.subscribe(
      channels.user(userId),
      async (event) => {
        await stream.writeSSE({ data: JSON.stringify(event) })
      }
    )

    // Heartbeat to keep connection alive
    const heartbeat = setInterval(() => {
      stream.writeSSE({ event: "ping", data: "" })
    }, 30000)

    stream.onAbort(() => {
      clearInterval(heartbeat)
      unsub()
    })
  })
})
```

## Service Dependencies

### Required Services

| Service | Purpose | Recommended |
|---------|---------|-------------|
| PostgreSQL | Primary database | PostgreSQL 15+ |
| Redis | Cache, pub/sub, sessions | Redis 7+ / Valkey |
| Object Storage | File storage, artifacts | S3/R2/GCS |
| Message Queue | Background jobs | NATS / Redis Streams |

### Optional Services

| Service | Purpose | Options |
|---------|---------|---------|
| Vault | Secret management | HashiCorp Vault, AWS KMS |
| Git Service | Repo management | GitHub, GitLab, Gitea |
| Metrics | Observability | Prometheus, Datadog |
| Tracing | Distributed tracing | Jaeger, Tempo |

## Configuration

### Environment Variables

```bash
# Server
PORT=3000
HOST=0.0.0.0
NODE_ENV=production

# Database
DATABASE_URL=postgresql://user:pass@host:5432/opencode
DATABASE_POOL_SIZE=20

# Redis
REDIS_URL=redis://host:6379
REDIS_CLUSTER=true

# Object Storage
STORAGE_PROVIDER=s3
STORAGE_BUCKET=opencode-files
STORAGE_REGION=us-east-1
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx

# Auth
JWT_SECRET=xxx
JWT_ISSUER=https://auth.opencode.io
OAUTH_GITHUB_CLIENT_ID=xxx
OAUTH_GITHUB_CLIENT_SECRET=xxx

# LLM Providers (org defaults)
ANTHROPIC_API_KEY=xxx
OPENAI_API_KEY=xxx

# Feature Flags
ENABLE_SANDBOXED_EXECUTION=true
ENABLE_GIT_INTEGRATION=true
MAX_CONCURRENT_SESSIONS=10
```

### Runtime Configuration

```typescript
interface ServerConfig {
  server: {
    port: number
    host: string
    trustProxy: boolean
  }
  database: {
    url: string
    poolSize: number
    ssl: boolean
  }
  redis: {
    url: string
    cluster: boolean
  }
  storage: {
    provider: "s3" | "r2" | "gcs" | "local"
    bucket: string
    region: string
  }
  auth: {
    jwtSecret: string
    jwtIssuer: string
    sessionTtl: number
  }
  limits: {
    maxSessionsPerUser: number
    maxMessagesPerSession: number
    maxFileSizeMb: number
    requestTimeoutMs: number
  }
  sandbox: {
    enabled: boolean
    provider: "docker" | "firecracker"
    poolSize: number
  }
}
```

## Deployment Architecture

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opencode-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: opencode-api
  template:
    metadata:
      labels:
        app: opencode-api
    spec:
      containers:
      - name: api
        image: opencode/api:latest
        ports:
        - containerPort: 3000
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: opencode-secrets
              key: database-url
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
        readinessProbe:
          httpGet:
            path: /ready
            port: 3000
```

### Service Mesh

For production deployments, consider:
- **Istio/Linkerd** for service mesh
- **mTLS** between services
- **Circuit breakers** for provider calls
- **Retry policies** with exponential backoff
