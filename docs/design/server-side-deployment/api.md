# API Design

## Overview

This document specifies the API design for the OpenCode server-side deployment, including versioning strategy, authentication, error handling, and endpoint specifications.

## API Versioning

### Versioning Strategy

Use URL path versioning for major versions with header-based minor versioning:

```
https://api.opencode.io/v1/sessions
                        ^^
                     Major version

Accept: application/json; version=1.2
                                  ^^^
                            Minor version
```

### Version Lifecycle

| Status | Description | Support |
|--------|-------------|---------|
| Current | Latest stable version | Full support |
| Deprecated | Previous version | 6 months |
| Sunset | End of life | No support |

### Deprecation Headers

```typescript
// Response headers for deprecated endpoints
c.header("Deprecation", "Sun, 01 Jan 2025 00:00:00 GMT")
c.header("Sunset", "Sun, 01 Jul 2025 00:00:00 GMT")
c.header("Link", '</v2/sessions>; rel="successor-version"')
```

## Authentication

### Request Authentication

```typescript
// Bearer token authentication
app.use("/api/*", async (c, next) => {
  const authHeader = c.req.header("Authorization")

  if (!authHeader?.startsWith("Bearer ")) {
    throw new AuthError("Missing authorization header", "MISSING_AUTH")
  }

  const token = authHeader.substring(7)

  // Check if API key or JWT
  if (token.startsWith("oc_")) {
    // API key authentication
    const apiKey = await validateApiKey(token)
    c.set("auth", { type: "apikey", ...apiKey })
  } else {
    // JWT authentication
    const jwt = await validateJwt(token)
    c.set("auth", { type: "jwt", ...jwt })
  }

  await next()
})
```

### API Key Format

```
oc_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
^^ ^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^
|   |            |
|   |            +-- 24 bytes base64url
|   +-- Environment (live/test)
+-- Prefix
```

## Request/Response Format

### Request Headers

```
Content-Type: application/json
Authorization: Bearer <token>
Accept: application/json
X-Request-ID: <uuid>           # Optional, for tracing
X-Idempotency-Key: <key>       # Optional, for idempotent operations
```

### Response Headers

```
Content-Type: application/json
X-Request-ID: <uuid>
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1609459200
```

### Pagination

```typescript
// Cursor-based pagination
interface PaginatedResponse<T> {
  data: T[]
  pagination: {
    cursor?: string
    hasMore: boolean
    total?: number
  }
}

// Query parameters
interface PaginationParams {
  cursor?: string   // Opaque cursor
  limit?: number    // Default: 50, Max: 100
}

// Example request
// GET /api/v1/sessions?limit=20&cursor=eyJpZCI6IjEyMyJ9
```

### Filtering & Sorting

```typescript
// Query parameter format
interface ListParams {
  // Filtering
  filter?: {
    status?: string[]
    createdAfter?: string    // ISO 8601
    createdBefore?: string
  }
  // Sorting
  sort?: string              // Field name
  order?: "asc" | "desc"
}

// Example
// GET /api/v1/sessions?filter[status]=active&sort=createdAt&order=desc
```

## Error Handling

### Error Response Format

```typescript
interface ErrorResponse {
  error: {
    code: string           // Machine-readable error code
    message: string        // Human-readable message
    details?: unknown      // Additional context
    requestId: string      // For support reference
    docs?: string          // Link to documentation
  }
}
```

### Error Codes

```typescript
// Error code hierarchy
const ErrorCodes = {
  // Authentication errors (401)
  AUTH_MISSING_TOKEN: "Missing authentication token",
  AUTH_INVALID_TOKEN: "Invalid or expired token",
  AUTH_INSUFFICIENT_SCOPE: "Token lacks required scope",

  // Authorization errors (403)
  FORBIDDEN: "Access denied",
  ORG_ACCESS_DENIED: "Not a member of this organization",
  RESOURCE_ACCESS_DENIED: "No access to this resource",

  // Validation errors (400)
  VALIDATION_ERROR: "Request validation failed",
  INVALID_PARAMETER: "Invalid parameter value",
  MISSING_PARAMETER: "Required parameter missing",

  // Not found errors (404)
  NOT_FOUND: "Resource not found",
  SESSION_NOT_FOUND: "Session not found",
  PROJECT_NOT_FOUND: "Project not found",

  // Conflict errors (409)
  CONFLICT: "Resource conflict",
  SESSION_ALREADY_EXISTS: "Session already exists",
  CONCURRENT_MODIFICATION: "Resource was modified",

  // Rate limiting (429)
  RATE_LIMITED: "Too many requests",
  QUOTA_EXCEEDED: "Usage quota exceeded",

  // Server errors (500)
  INTERNAL_ERROR: "Internal server error",
  SERVICE_UNAVAILABLE: "Service temporarily unavailable",
  PROVIDER_ERROR: "LLM provider error",
}
```

### HTTP Status Codes

| Code | Usage |
|------|-------|
| 200 | Success with body |
| 201 | Resource created |
| 204 | Success, no body |
| 400 | Validation error |
| 401 | Authentication required |
| 403 | Authorization denied |
| 404 | Resource not found |
| 409 | Conflict |
| 422 | Unprocessable entity |
| 429 | Rate limited |
| 500 | Server error |
| 503 | Service unavailable |

## Streaming Responses

### Server-Sent Events

```typescript
// SSE endpoint for real-time events
app.get("/api/v1/events", async (c) => {
  return streamSSE(c, async (stream) => {
    // Connection established
    await stream.writeSSE({
      event: "connected",
      data: JSON.stringify({ timestamp: Date.now() }),
    })

    // Subscribe to events
    const unsub = eventBus.subscribe(c.get("userId"), async (event) => {
      await stream.writeSSE({
        event: event.type,
        data: JSON.stringify(event.payload),
        id: event.id,
      })
    })

    // Heartbeat every 30 seconds
    const heartbeat = setInterval(() => {
      stream.writeSSE({ event: "ping", data: "" })
    }, 30000)

    // Cleanup on disconnect
    stream.onAbort(() => {
      clearInterval(heartbeat)
      unsub()
    })
  })
})
```

### Streaming Chat Response

```typescript
// POST /api/v1/sessions/:id/messages
// Returns streaming response
app.post("/api/v1/sessions/:id/messages", async (c) => {
  const { id } = c.req.param()
  const body = await c.req.json()

  return streamSSE(c, async (stream) => {
    const generator = sessionOrchestrator.chat(id, body)

    for await (const event of generator) {
      await stream.writeSSE({
        event: event.type,
        data: JSON.stringify(event),
      })
    }

    // Signal completion
    await stream.writeSSE({
      event: "done",
      data: JSON.stringify({ messageId: "..." }),
    })
  })
})
```

### Event Types

```typescript
type StreamEvent =
  | { type: "message.start"; messageId: string }
  | { type: "text.delta"; content: string }
  | { type: "text.done"; content: string }
  | { type: "tool.start"; toolId: string; name: string }
  | { type: "tool.input"; content: string }
  | { type: "tool.output"; content: string }
  | { type: "tool.done"; result: unknown }
  | { type: "message.done"; usage: Usage }
  | { type: "error"; error: Error }
```

## API Endpoints

### Sessions

```typescript
// List sessions
// GET /api/v1/sessions
interface ListSessionsResponse {
  data: Session[]
  pagination: Pagination
}

// Create session
// POST /api/v1/sessions
interface CreateSessionRequest {
  workspaceId: string
  title?: string
  model?: {
    providerId: string
    modelId: string
  }
}

// Get session
// GET /api/v1/sessions/:id
interface GetSessionResponse {
  data: Session
}

// Update session
// PATCH /api/v1/sessions/:id
interface UpdateSessionRequest {
  title?: string
}

// Delete session
// DELETE /api/v1/sessions/:id

// Send message (streaming)
// POST /api/v1/sessions/:id/messages
interface SendMessageRequest {
  content: string
  files?: FileAttachment[]
}

// List messages
// GET /api/v1/sessions/:id/messages
interface ListMessagesResponse {
  data: Message[]
  pagination: Pagination
}

// Abort session
// POST /api/v1/sessions/:id/abort

// Fork session
// POST /api/v1/sessions/:id/fork
interface ForkSessionRequest {
  messageId: string
}

// Share session
// POST /api/v1/sessions/:id/share
interface ShareSessionResponse {
  url: string
  expiresAt: string
}
```

### Workspaces

```typescript
// List workspaces
// GET /api/v1/workspaces
interface ListWorkspacesResponse {
  data: Workspace[]
  pagination: Pagination
}

// Create workspace
// POST /api/v1/workspaces
interface CreateWorkspaceRequest {
  name: string
  description?: string
  gitConfig?: {
    provider: "github" | "gitlab"
    repoUrl: string
    branch?: string
  }
}

// Get workspace
// GET /api/v1/workspaces/:id

// Update workspace
// PATCH /api/v1/workspaces/:id

// Delete workspace
// DELETE /api/v1/workspaces/:id

// List workspace projects
// GET /api/v1/workspaces/:id/projects
```

### Projects

```typescript
// List projects
// GET /api/v1/projects

// Create project
// POST /api/v1/projects
interface CreateProjectRequest {
  workspaceId: string
  name: string
  path?: string
}

// Get project
// GET /api/v1/projects/:id

// Update project
// PATCH /api/v1/projects/:id

// Delete project
// DELETE /api/v1/projects/:id
```

### Files

```typescript
// List files in workspace
// GET /api/v1/workspaces/:id/files
interface ListFilesRequest {
  path?: string    // Directory path
  pattern?: string // Glob pattern
}

// Get file content
// GET /api/v1/workspaces/:id/files/content
interface GetFileContentRequest {
  path: string
  encoding?: "utf8" | "base64"
}

// Search in files
// GET /api/v1/workspaces/:id/files/search
interface SearchFilesRequest {
  query: string
  path?: string
  type?: string    // File type filter
}

// Git status
// GET /api/v1/workspaces/:id/git/status
```

### Providers

```typescript
// List available providers
// GET /api/v1/providers
interface ListProvidersResponse {
  data: Provider[]
}

// List models for provider
// GET /api/v1/providers/:id/models
interface ListModelsResponse {
  data: Model[]
}

// Get user's provider config
// GET /api/v1/providers/:id/config

// Set provider API key (BYOK)
// PUT /api/v1/providers/:id/key
interface SetProviderKeyRequest {
  apiKey: string
}

// Delete provider key
// DELETE /api/v1/providers/:id/key
```

### Users & Organizations

```typescript
// Get current user
// GET /api/v1/users/me
interface GetCurrentUserResponse {
  data: User
}

// Update user preferences
// PATCH /api/v1/users/me
interface UpdateUserRequest {
  name?: string
  preferences?: UserPreferences
}

// Get organization
// GET /api/v1/organizations/:id

// List organization members
// GET /api/v1/organizations/:id/members

// Invite member
// POST /api/v1/organizations/:id/invitations

// Remove member
// DELETE /api/v1/organizations/:id/members/:userId
```

### API Keys

```typescript
// List API keys
// GET /api/v1/api-keys
interface ListApiKeysResponse {
  data: ApiKey[] // Keys shown with prefix only
}

// Create API key
// POST /api/v1/api-keys
interface CreateApiKeyRequest {
  name: string
  scopes: Scope[]
  expiresAt?: string
}
interface CreateApiKeyResponse {
  key: string // Full key shown once
  data: ApiKey
}

// Delete API key
// DELETE /api/v1/api-keys/:id
```

### Usage & Billing

```typescript
// Get usage summary
// GET /api/v1/usage
interface GetUsageRequest {
  period?: "day" | "week" | "month"
  startDate?: string
  endDate?: string
}
interface GetUsageResponse {
  data: {
    tokens: {
      input: number
      output: number
      total: number
    }
    cost: number
    byProvider: Record<string, UsageByProvider>
    byModel: Record<string, UsageByModel>
  }
}

// Get usage breakdown
// GET /api/v1/usage/breakdown
interface UsageBreakdownResponse {
  data: UsageRecord[]
  pagination: Pagination
}
```

## Webhooks

### Webhook Configuration

```typescript
// Register webhook
// POST /api/v1/webhooks
interface CreateWebhookRequest {
  url: string
  events: WebhookEvent[]
  secret?: string
}

// Webhook events
type WebhookEvent =
  | "session.created"
  | "session.completed"
  | "session.error"
  | "message.created"
  | "usage.threshold"
```

### Webhook Payload

```typescript
interface WebhookPayload {
  id: string
  type: WebhookEvent
  timestamp: string
  data: unknown
}

// Signature verification
// X-Webhook-Signature: sha256=<hmac>
function verifyWebhook(payload: string, signature: string, secret: string): boolean {
  const expected = crypto
    .createHmac("sha256", secret)
    .update(payload)
    .digest("hex")
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(`sha256=${expected}`)
  )
}
```

## Rate Limiting

### Limits by Plan

| Plan | Requests/min | Messages/day | Tokens/month |
|------|-------------|--------------|--------------|
| Free | 20 | 100 | 100K |
| Team | 100 | 1,000 | 1M |
| Enterprise | Custom | Custom | Custom |

### Rate Limit Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1609459200
Retry-After: 30
```

### Rate Limit Response

```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Too many requests",
    "details": {
      "limit": 100,
      "remaining": 0,
      "reset": 1609459200,
      "retryAfter": 30
    },
    "requestId": "req_xxx"
  }
}
```

## SDK Examples

### TypeScript/JavaScript

```typescript
import { OpenCodeClient } from "@opencode/sdk"

const client = new OpenCodeClient({
  apiKey: "oc_live_xxx",
  baseUrl: "https://api.opencode.io",
})

// Create session
const session = await client.sessions.create({
  workspaceId: "ws_xxx",
  title: "Debug authentication",
})

// Send message and stream response
const stream = client.sessions.chat(session.id, {
  content: "Find and fix the authentication bug",
})

for await (const event of stream) {
  if (event.type === "text.delta") {
    process.stdout.write(event.content)
  }
}

// List sessions
const sessions = await client.sessions.list({
  limit: 20,
  filter: { status: ["active"] },
})
```

### Python

```python
from opencode import OpenCodeClient

client = OpenCodeClient(api_key="oc_live_xxx")

# Create session
session = client.sessions.create(
    workspace_id="ws_xxx",
    title="Debug authentication"
)

# Send message and stream response
stream = client.sessions.chat(
    session.id,
    content="Find and fix the authentication bug"
)

for event in stream:
    if event.type == "text.delta":
        print(event.content, end="", flush=True)
```

### cURL

```bash
# Create session
curl -X POST https://api.opencode.io/v1/sessions \
  -H "Authorization: Bearer oc_live_xxx" \
  -H "Content-Type: application/json" \
  -d '{"workspaceId": "ws_xxx", "title": "Debug auth"}'

# Send message (streaming)
curl -X POST https://api.opencode.io/v1/sessions/sess_xxx/messages \
  -H "Authorization: Bearer oc_live_xxx" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"content": "Find and fix the authentication bug"}'
```

## OpenAPI Specification

The complete OpenAPI 3.1 specification is available at:

```
GET /api/v1/openapi.json
GET /api/v1/openapi.yaml
```

Interactive documentation (Swagger UI):

```
GET /docs
```
