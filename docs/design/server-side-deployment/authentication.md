# Authentication & Authorization

## Overview

The server-side deployment requires a comprehensive auth system supporting multiple authentication methods, organization-based multi-tenancy, and fine-grained access control.

## Authentication Methods

### 1. OAuth 2.0 / OIDC

Primary authentication method for web and desktop clients.

```typescript
interface OAuthConfig {
  providers: {
    github: {
      clientId: string
      clientSecret: string
      scopes: ["user:email", "read:org"]
    }
    google: {
      clientId: string
      clientSecret: string
      scopes: ["email", "profile"]
    }
    microsoft: {
      clientId: string
      clientSecret: string
      tenant: string
    }
    // Custom OIDC provider for enterprise
    oidc?: {
      issuer: string
      clientId: string
      clientSecret: string
      scopes: string[]
    }
  }
}
```

**OAuth Flow**:
```
1. Client redirects to /auth/login/:provider
2. Server redirects to provider authorization URL
3. User authenticates with provider
4. Provider redirects to /auth/callback/:provider
5. Server exchanges code for tokens
6. Server creates/updates user record
7. Server issues JWT + refresh token
8. Client stores tokens securely
```

### 2. API Keys

For programmatic access (CI/CD, SDK, CLI).

```typescript
interface ApiKey {
  id: string
  orgId: string
  userId: string
  name: string
  prefix: string           // First 8 chars for identification
  hash: string             // Argon2 hash of full key
  scopes: Scope[]
  rateLimit?: RateLimit
  expiresAt?: Date
  lastUsedAt?: Date
  createdAt: Date
}

// Key format: oc_live_xxxxxxxxxxxxxxxxxxxx
// Prefix identifies key type (live/test)
```

**Key Generation**:
```typescript
async function generateApiKey(input: CreateKeyInput): Promise<{ key: string; record: ApiKey }> {
  const key = `oc_live_${crypto.randomBytes(24).toString('base64url')}`
  const hash = await argon2.hash(key)

  const record: ApiKey = {
    id: generateId(),
    orgId: input.orgId,
    userId: input.userId,
    name: input.name,
    prefix: key.substring(0, 16),
    hash,
    scopes: input.scopes,
    createdAt: new Date(),
  }

  await db.apiKeys.insert(record)

  return { key, record } // Return full key only once
}
```

### 3. Personal Access Tokens (PAT)

User-scoped tokens with limited lifetime.

```typescript
interface PersonalAccessToken {
  id: string
  userId: string
  name: string
  hash: string
  scopes: Scope[]
  expiresAt: Date
  createdAt: Date
}
```

## Token Management

### JWT Structure

```typescript
interface JWTPayload {
  // Standard claims
  iss: string              // Issuer
  sub: string              // User ID
  aud: string[]            // Audience
  exp: number              // Expiration
  iat: number              // Issued at
  jti: string              // Token ID

  // Custom claims
  org_id: string           // Organization ID
  org_role: OrgRole        // Role in organization
  scopes: string[]         // Granted scopes
  session_id?: string      // For session-specific tokens
}
```

### Token Lifecycle

```typescript
const tokenConfig = {
  access: {
    ttl: 15 * 60,          // 15 minutes
    algorithm: "RS256",
  },
  refresh: {
    ttl: 7 * 24 * 60 * 60, // 7 days
    rotation: true,        // Single-use refresh tokens
    family: true,          // Track token families
  },
}
```

**Refresh Token Rotation**:
```typescript
async function refreshTokens(refreshToken: string): Promise<TokenPair> {
  const payload = await verifyRefreshToken(refreshToken)

  // Check if token was already used (replay attack)
  const tokenRecord = await db.refreshTokens.findById(payload.jti)
  if (tokenRecord.used) {
    // Token reuse detected - revoke entire family
    await db.refreshTokens.revokeFamily(tokenRecord.familyId)
    throw new AuthError("Token reuse detected", "TOKEN_REUSE")
  }

  // Mark current token as used
  await db.refreshTokens.markUsed(payload.jti)

  // Issue new token pair
  return issueTokens(payload.sub, {
    familyId: tokenRecord.familyId,
  })
}
```

## Authorization Model

### Role-Based Access Control (RBAC)

```typescript
type OrgRole = "owner" | "admin" | "member" | "guest"

interface Permission {
  resource: Resource
  action: Action
}

type Resource =
  | "organization"
  | "workspace"
  | "project"
  | "session"
  | "user"
  | "api_key"
  | "provider"
  | "billing"

type Action =
  | "create"
  | "read"
  | "update"
  | "delete"
  | "manage"
  | "execute"
```

**Role Permissions Matrix**:

| Permission | Owner | Admin | Member | Guest |
|------------|-------|-------|--------|-------|
| org:manage | yes | no | no | no |
| org:read | yes | yes | yes | yes |
| workspace:create | yes | yes | no | no |
| workspace:delete | yes | yes | no | no |
| project:create | yes | yes | yes | no |
| session:create | yes | yes | yes | yes |
| session:read (own) | yes | yes | yes | yes |
| session:read (all) | yes | yes | no | no |
| api_key:create | yes | yes | yes | no |
| provider:manage | yes | yes | no | no |
| billing:manage | yes | no | no | no |

### Scope-Based Access (API Keys)

```typescript
type Scope =
  | "sessions:read"
  | "sessions:write"
  | "projects:read"
  | "projects:write"
  | "workspaces:read"
  | "workspaces:write"
  | "files:read"
  | "files:write"
  | "tools:execute"
  | "admin"
```

**Scope Validation**:
```typescript
function requireScopes(...required: Scope[]) {
  return async (c: Context, next: Next) => {
    const granted = c.get("scopes") as Scope[]

    for (const scope of required) {
      if (!granted.includes(scope) && !granted.includes("admin")) {
        throw new AuthError(`Missing scope: ${scope}`, "INSUFFICIENT_SCOPE")
      }
    }

    await next()
  }
}

// Usage
app.post("/sessions/:id/messages",
  requireScopes("sessions:write"),
  sessionController.sendMessage
)
```

### Resource-Level Authorization

```typescript
interface ResourcePolicy {
  check(ctx: TenantContext, resource: Resource, action: Action): Promise<boolean>
}

class SessionPolicy implements ResourcePolicy {
  async check(ctx: TenantContext, session: Session, action: Action): Promise<boolean> {
    // Owners can do anything
    if (ctx.orgRole === "owner") return true

    // Check if user owns the session
    const isOwner = session.userId === ctx.userId

    switch (action) {
      case "read":
        // Members can read own sessions, admins can read all
        return isOwner || ctx.orgRole === "admin"

      case "update":
      case "delete":
        // Only owner or admin can modify
        return isOwner || ctx.orgRole === "admin"

      case "execute":
        // Only owner can execute tools in session
        return isOwner

      default:
        return false
    }
  }
}
```

## Multi-Tenancy

### Tenant Context

```typescript
interface TenantContext {
  userId: string
  orgId: string
  orgRole: OrgRole
  workspaceId?: string
  sessionId?: string
  scopes: Scope[]
  metadata: {
    ip: string
    userAgent: string
    requestId: string
  }
}

// Middleware to inject tenant context
async function tenantContext(c: Context, next: Next) {
  const jwt = c.get("jwt") as JWTPayload

  const ctx: TenantContext = {
    userId: jwt.sub,
    orgId: jwt.org_id,
    orgRole: jwt.org_role,
    scopes: jwt.scopes,
    metadata: {
      ip: c.req.header("x-forwarded-for") || c.req.ip,
      userAgent: c.req.header("user-agent") || "",
      requestId: c.get("requestId"),
    },
  }

  c.set("tenant", ctx)
  await next()
}
```

### Organization Isolation

```typescript
// Database queries automatically scoped to organization
class SessionRepository {
  constructor(private ctx: TenantContext) {}

  async findById(id: string): Promise<Session | null> {
    return db.sessions.findFirst({
      where: {
        id,
        project: {
          workspace: {
            orgId: this.ctx.orgId, // Automatic org scoping
          },
        },
      },
    })
  }

  async list(filter: SessionFilter): Promise<Session[]> {
    return db.sessions.findMany({
      where: {
        ...filter,
        project: {
          workspace: {
            orgId: this.ctx.orgId,
          },
        },
        // Non-admins only see own sessions
        ...(this.ctx.orgRole !== "admin" && {
          userId: this.ctx.userId,
        }),
      },
    })
  }
}
```

## LLM Provider Authentication

### User BYOK (Bring Your Own Key)

```typescript
interface UserProviderKey {
  id: string
  userId: string
  providerId: string
  encryptedKey: string     // AES-256-GCM encrypted
  keyId: string            // KMS key ID used
  createdAt: Date
  lastUsedAt?: Date
}

// Encrypt user's API key before storage
async function storeProviderKey(
  userId: string,
  providerId: string,
  apiKey: string
): Promise<void> {
  const { ciphertext, keyId } = await kms.encrypt(apiKey)

  await db.userProviderKeys.upsert({
    where: { userId, providerId },
    create: {
      id: generateId(),
      userId,
      providerId,
      encryptedKey: ciphertext,
      keyId,
      createdAt: new Date(),
    },
    update: {
      encryptedKey: ciphertext,
      keyId,
    },
  })
}
```

### Organization Default Keys

```typescript
interface OrgProviderConfig {
  orgId: string
  providerId: string
  encryptedKey: string
  rateLimit?: RateLimit
  allowUserOverride: boolean
  usageTracking: boolean
}

// Key resolution order
async function resolveProviderKey(
  ctx: TenantContext,
  providerId: string
): Promise<string> {
  // 1. Check user BYOK
  const userKey = await db.userProviderKeys.findFirst({
    where: { userId: ctx.userId, providerId },
  })
  if (userKey) {
    return kms.decrypt(userKey.encryptedKey, userKey.keyId)
  }

  // 2. Check org default
  const orgConfig = await db.orgProviderConfigs.findFirst({
    where: { orgId: ctx.orgId, providerId },
  })
  if (orgConfig) {
    return kms.decrypt(orgConfig.encryptedKey, orgConfig.keyId)
  }

  throw new AuthError(`No API key for provider: ${providerId}`, "NO_PROVIDER_KEY")
}
```

## Session Management

### Active Session Tracking

```typescript
interface UserSession {
  id: string
  userId: string
  tokenFamily: string
  device: string
  ip: string
  location?: string
  createdAt: Date
  lastActiveAt: Date
  expiresAt: Date
}

// Track active sessions per user
async function createUserSession(
  userId: string,
  metadata: SessionMetadata
): Promise<UserSession> {
  // Enforce max sessions per user
  const activeSessions = await db.userSessions.count({
    where: { userId, expiresAt: { gt: new Date() } },
  })

  if (activeSessions >= MAX_SESSIONS_PER_USER) {
    // Revoke oldest session
    const oldest = await db.userSessions.findFirst({
      where: { userId },
      orderBy: { lastActiveAt: "asc" },
    })
    if (oldest) {
      await revokeSession(oldest.id)
    }
  }

  return db.userSessions.create({
    data: {
      id: generateId(),
      userId,
      tokenFamily: generateId(),
      device: metadata.device,
      ip: metadata.ip,
      createdAt: new Date(),
      lastActiveAt: new Date(),
      expiresAt: new Date(Date.now() + SESSION_TTL),
    },
  })
}
```

### Session Revocation

```typescript
// Revoke specific session
async function revokeSession(sessionId: string): Promise<void> {
  const session = await db.userSessions.findById(sessionId)
  if (!session) return

  // Revoke all tokens in family
  await db.refreshTokens.updateMany({
    where: { familyId: session.tokenFamily },
    data: { revoked: true },
  })

  // Delete session
  await db.userSessions.delete({ id: sessionId })

  // Publish revocation event
  await redis.publish(`user:${session.userId}:revoke`, {
    type: "session_revoked",
    sessionId,
  })
}

// Revoke all sessions for user
async function revokeAllSessions(userId: string): Promise<void> {
  const sessions = await db.userSessions.findMany({
    where: { userId },
  })

  for (const session of sessions) {
    await revokeSession(session.id)
  }
}
```

## Security Controls

### Rate Limiting

```typescript
interface RateLimitConfig {
  // Per-user limits
  user: {
    requests: number
    window: number
    burst?: number
  }
  // Per-organization limits
  org: {
    requests: number
    window: number
  }
  // Per-endpoint limits
  endpoints: {
    [path: string]: {
      requests: number
      window: number
    }
  }
}

// Example config
const rateLimitConfig: RateLimitConfig = {
  user: {
    requests: 100,
    window: 60,       // 100 req/min per user
    burst: 20,        // Allow burst of 20
  },
  org: {
    requests: 10000,
    window: 3600,     // 10k req/hour per org
  },
  endpoints: {
    "POST /sessions/:id/messages": {
      requests: 10,
      window: 60,     // 10 messages/min
    },
    "POST /auth/login": {
      requests: 5,
      window: 300,    // 5 attempts/5min
    },
  },
}
```

### Audit Logging

```typescript
interface AuditLog {
  id: string
  timestamp: Date
  userId: string
  orgId: string
  action: string
  resource: string
  resourceId?: string
  metadata: Record<string, unknown>
  ip: string
  userAgent: string
  status: "success" | "failure"
  errorCode?: string
}

// Log security-sensitive actions
async function auditLog(entry: Omit<AuditLog, "id" | "timestamp">): Promise<void> {
  await db.auditLogs.create({
    data: {
      id: generateId(),
      timestamp: new Date(),
      ...entry,
    },
  })
}

// Usage
await auditLog({
  userId: ctx.userId,
  orgId: ctx.orgId,
  action: "session.delete",
  resource: "session",
  resourceId: sessionId,
  metadata: { reason: "user_request" },
  ip: ctx.metadata.ip,
  userAgent: ctx.metadata.userAgent,
  status: "success",
})
```

### Brute Force Protection

```typescript
// Failed login tracking
interface FailedAttempt {
  identifier: string      // email or IP
  attempts: number
  lastAttempt: Date
  lockedUntil?: Date
}

async function checkBruteForce(identifier: string): Promise<void> {
  const record = await redis.get<FailedAttempt>(`failed:${identifier}`)

  if (record?.lockedUntil && record.lockedUntil > new Date()) {
    const waitTime = Math.ceil((record.lockedUntil.getTime() - Date.now()) / 1000)
    throw new AuthError(
      `Too many attempts. Try again in ${waitTime}s`,
      "RATE_LIMITED"
    )
  }
}

async function recordFailedAttempt(identifier: string): Promise<void> {
  const key = `failed:${identifier}`
  const record = await redis.get<FailedAttempt>(key) || {
    identifier,
    attempts: 0,
    lastAttempt: new Date(),
  }

  record.attempts++
  record.lastAttempt = new Date()

  // Progressive lockout
  if (record.attempts >= 5) {
    const lockoutMinutes = Math.min(Math.pow(2, record.attempts - 5), 60)
    record.lockedUntil = new Date(Date.now() + lockoutMinutes * 60 * 1000)
  }

  await redis.set(key, record, { ex: 3600 })
}
```

## Implementation Checklist

- [ ] OAuth 2.0 / OIDC integration
- [ ] API key generation and validation
- [ ] JWT issuance and validation
- [ ] Refresh token rotation
- [ ] Role-based access control
- [ ] Scope-based permissions
- [ ] Multi-tenant isolation
- [ ] Provider key management
- [ ] Session tracking
- [ ] Rate limiting
- [ ] Audit logging
- [ ] Brute force protection
