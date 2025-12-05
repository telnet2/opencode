# Security

## Overview

This document outlines security controls, threat mitigations, and compliance requirements for the OpenCode server-side deployment.

## Threat Model

### Assets to Protect

1. **User Data**: Sessions, messages, code, credentials
2. **Provider Keys**: API keys for LLM providers
3. **Infrastructure**: Servers, databases, networks
4. **Service Availability**: Protection against DoS

### Threat Actors

1. **External Attackers**: Unauthorized access attempts
2. **Malicious Users**: Abuse of legitimate access
3. **Compromised Accounts**: Stolen credentials
4. **Insider Threats**: Rogue employees/contractors

### Attack Vectors

| Vector | Risk | Mitigation |
|--------|------|------------|
| SQL Injection | High | Parameterized queries, ORM |
| XSS | Medium | Content Security Policy, sanitization |
| CSRF | Medium | SameSite cookies, CSRF tokens |
| Command Injection | Critical | Sandboxed execution |
| API Key Theft | High | Encryption at rest, KMS |
| Session Hijacking | High | Secure cookies, token rotation |
| DoS/DDoS | High | Rate limiting, CDN protection |

## Network Security

### Architecture

```
Internet
    │
    ▼
┌─────────────┐
│   WAF/CDN   │  ← DDoS protection, bot filtering
│ (Cloudflare)│
└──────┬──────┘
       │
    ┌──▼──┐
    │ VPC │
    │     │
    │  ┌──┴──────────────────────┐
    │  │    Public Subnet        │
    │  │  ┌─────────────────┐    │
    │  │  │  Load Balancer  │    │
    │  │  └────────┬────────┘    │
    │  └───────────┼─────────────┘
    │              │
    │  ┌───────────▼─────────────┐
    │  │    Private Subnet       │
    │  │  ┌─────────────────┐    │
    │  │  │   API Servers   │    │
    │  │  └────────┬────────┘    │
    │  │           │             │
    │  │  ┌────────▼────────┐    │
    │  │  │    Database     │    │
    │  │  └─────────────────┘    │
    │  └─────────────────────────┘
    └─────────────────────────────┘
```

### Firewall Rules

```yaml
# Network policies for Kubernetes
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-server-policy
  namespace: opencode
spec:
  podSelector:
    matchLabels:
      app: opencode-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - port: 3000
  egress:
  # Database
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - port: 5432
  # Redis
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - port: 6379
  # External LLM APIs
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
    ports:
    - port: 443
```

### TLS Configuration

```typescript
// Minimum TLS 1.2, prefer 1.3
const tlsConfig = {
  minVersion: "TLSv1.2",
  ciphers: [
    "TLS_AES_256_GCM_SHA384",
    "TLS_CHACHA20_POLY1305_SHA256",
    "TLS_AES_128_GCM_SHA256",
    "ECDHE-RSA-AES256-GCM-SHA384",
    "ECDHE-RSA-AES128-GCM-SHA256",
  ].join(":"),
  honorCipherOrder: true,
}
```

### mTLS for Internal Services

```yaml
# Istio PeerAuthentication for mTLS
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: opencode
spec:
  mtls:
    mode: STRICT
```

## Application Security

### Input Validation

```typescript
import { z } from "zod"

// Strict input validation schemas
const CreateSessionSchema = z.object({
  title: z.string()
    .min(1)
    .max(500)
    .regex(/^[\w\s\-.,!?]+$/),
  projectId: z.string().uuid(),
  model: z.object({
    providerId: z.enum(["anthropic", "openai", "google"]),
    modelId: z.string().max(100),
  }),
})

const MessageSchema = z.object({
  content: z.string().max(100000), // 100KB limit
  files: z.array(z.object({
    name: z.string().max(255),
    size: z.number().max(10 * 1024 * 1024), // 10MB
    mimeType: z.string().regex(/^[\w\-]+\/[\w\-+.]+$/),
  })).max(10).optional(),
})

// Middleware for validation
function validate<T>(schema: z.ZodSchema<T>) {
  return async (c: Context, next: Next) => {
    const result = schema.safeParse(await c.req.json())
    if (!result.success) {
      throw new ValidationError(result.error)
    }
    c.set("body", result.data)
    await next()
  }
}
```

### Output Encoding

```typescript
// Sanitize output for different contexts
import DOMPurify from "isomorphic-dompurify"

function sanitizeForHtml(input: string): string {
  return DOMPurify.sanitize(input, {
    ALLOWED_TAGS: ["b", "i", "em", "strong", "code", "pre", "a"],
    ALLOWED_ATTR: ["href"],
  })
}

function sanitizeForJson(input: unknown): unknown {
  // Remove any prototype pollution attempts
  return JSON.parse(JSON.stringify(input, (key, value) => {
    if (key === "__proto__" || key === "constructor" || key === "prototype") {
      return undefined
    }
    return value
  }))
}
```

### Content Security Policy

```typescript
// CSP headers for web UI
const cspPolicy = {
  "default-src": ["'self'"],
  "script-src": ["'self'", "'wasm-unsafe-eval'"],
  "style-src": ["'self'", "'unsafe-inline'"],
  "img-src": ["'self'", "data:", "https:"],
  "connect-src": [
    "'self'",
    "https://api.anthropic.com",
    "https://api.openai.com",
  ],
  "frame-ancestors": ["'none'"],
  "form-action": ["'self'"],
  "base-uri": ["'self'"],
  "object-src": ["'none'"],
}

app.use((c, next) => {
  const csp = Object.entries(cspPolicy)
    .map(([key, values]) => `${key} ${values.join(" ")}`)
    .join("; ")
  c.header("Content-Security-Policy", csp)
  return next()
})
```

### Security Headers

```typescript
// Security headers middleware
app.use((c, next) => {
  // Prevent MIME sniffing
  c.header("X-Content-Type-Options", "nosniff")

  // Clickjacking protection
  c.header("X-Frame-Options", "DENY")

  // XSS protection (legacy browsers)
  c.header("X-XSS-Protection", "1; mode=block")

  // Referrer policy
  c.header("Referrer-Policy", "strict-origin-when-cross-origin")

  // Permissions policy
  c.header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

  // HSTS (1 year)
  c.header(
    "Strict-Transport-Security",
    "max-age=31536000; includeSubDomains; preload"
  )

  return next()
})
```

## Sandboxed Code Execution

### Isolation Strategy

Tool execution (Bash, file operations) runs in isolated containers to prevent:
- Filesystem escape
- Network access to internal services
- Resource exhaustion
- Privilege escalation

### Container Security

```yaml
# Security context for worker pods
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: sandbox
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
    resources:
      limits:
        cpu: "1"
        memory: "512Mi"
        ephemeral-storage: "1Gi"
```

### Firecracker/gVisor Integration

```typescript
interface SandboxConfig {
  // Firecracker microVM settings
  firecracker: {
    kernelPath: string
    rootfsPath: string
    vcpuCount: number
    memSizeMib: number
    networkInterface?: {
      hostDevName: string
      guestMac: string
    }
  }
  // Or gVisor runtime
  gvisor: {
    platform: "ptrace" | "kvm"
    network: "none" | "host"
  }
}
```

### Command Filtering

```typescript
// Block dangerous commands
const blockedCommands = [
  /\brm\s+-rf\s+\//, // rm -rf /
  /\bmkfs\b/,
  /\bdd\b.*of=\/dev/,
  /\b(sudo|su)\b/,
  /\bchmod\s+777/,
  /\bcurl\b.*\|\s*(bash|sh)/,
  /\bwget\b.*\|\s*(bash|sh)/,
]

function validateCommand(cmd: string): boolean {
  for (const pattern of blockedCommands) {
    if (pattern.test(cmd)) {
      return false
    }
  }
  return true
}
```

## Data Protection

### Encryption at Rest

```typescript
// All sensitive data encrypted with AES-256-GCM
interface EncryptionConfig {
  algorithm: "aes-256-gcm"
  keyManagement: "aws-kms" | "hashicorp-vault" | "gcp-kms"
  keyRotationDays: 90
}

// Encrypt provider API keys
async function encryptApiKey(key: string): Promise<EncryptedKey> {
  const kmsKeyId = process.env.KMS_KEY_ID
  const { CiphertextBlob, KeyId } = await kms.encrypt({
    KeyId: kmsKeyId,
    Plaintext: Buffer.from(key),
    EncryptionContext: {
      purpose: "provider-api-key",
    },
  })

  return {
    ciphertext: CiphertextBlob.toString("base64"),
    keyId: KeyId,
  }
}
```

### Encryption in Transit

- TLS 1.2+ for all external connections
- mTLS for internal service communication
- Certificate pinning for LLM provider connections

### Data Classification

| Classification | Examples | Controls |
|---------------|----------|----------|
| Public | Marketing content | None |
| Internal | Usage metrics | Access control |
| Confidential | User sessions | Encryption, audit logs |
| Restricted | API keys, PII | Encryption, KMS, strict access |

### Key Management

```typescript
// HashiCorp Vault integration
interface VaultConfig {
  address: string
  authMethod: "kubernetes" | "token" | "aws-iam"
  secretEngine: "kv-v2"
  transitEngine: "transit"
}

class VaultClient {
  // Get encryption key for data
  async getDataKey(purpose: string): Promise<Buffer> {
    const response = await this.client.write(
      `transit/datakey/plaintext/${purpose}`,
      { context: Buffer.from(purpose).toString("base64") }
    )
    return Buffer.from(response.plaintext, "base64")
  }

  // Encrypt with transit engine
  async encrypt(plaintext: string, keyName: string): Promise<string> {
    const response = await this.client.write(
      `transit/encrypt/${keyName}`,
      { plaintext: Buffer.from(plaintext).toString("base64") }
    )
    return response.ciphertext
  }
}
```

## Secret Management

### Secret Storage

```yaml
# External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: opencode-secrets
  namespace: opencode
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: vault-backend
  target:
    name: opencode-secrets
  data:
  - secretKey: database-url
    remoteRef:
      key: opencode/database
      property: url
  - secretKey: jwt-secret
    remoteRef:
      key: opencode/auth
      property: jwt-secret
  - secretKey: anthropic-api-key
    remoteRef:
      key: opencode/providers
      property: anthropic-key
```

### Secret Rotation

```typescript
// Automatic secret rotation
interface RotationConfig {
  // Database credentials
  database: {
    rotationSchedule: "0 0 * * 0", // Weekly
    maxAge: 90, // Days
  },
  // API keys
  apiKeys: {
    rotationSchedule: "0 0 1 * *", // Monthly
    maxAge: 365,
  },
  // JWT signing keys
  jwtKeys: {
    rotationSchedule: "0 0 1 */3 *", // Quarterly
    gracePeriod: 7, // Days to accept old key
  },
}
```

## Audit & Compliance

### Audit Logging

```typescript
// Comprehensive audit logging
interface AuditEvent {
  id: string
  timestamp: Date
  actor: {
    userId: string
    orgId: string
    ip: string
    userAgent: string
  }
  action: string
  resource: {
    type: string
    id: string
  }
  outcome: "success" | "failure"
  metadata: Record<string, unknown>
}

// Log security-sensitive actions
const auditableActions = [
  "user.login",
  "user.logout",
  "user.mfa_enabled",
  "user.mfa_disabled",
  "user.password_changed",
  "apikey.created",
  "apikey.deleted",
  "session.created",
  "session.deleted",
  "session.shared",
  "provider.key_added",
  "provider.key_removed",
  "org.member_added",
  "org.member_removed",
  "org.settings_changed",
]
```

### Log Aggregation

```yaml
# Fluent Bit for log collection
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush         5
        Log_Level     info
        Parsers_File  parsers.conf

    [INPUT]
        Name              tail
        Path              /var/log/containers/opencode-*.log
        Parser            docker
        Tag               opencode.*
        Mem_Buf_Limit     5MB

    [OUTPUT]
        Name              es
        Match             opencode.*
        Host              elasticsearch
        Port              9200
        Index             opencode-logs
        Type              _doc
```

### Compliance Controls

#### SOC 2 Type II

- [ ] Access control policies
- [ ] Encryption at rest and in transit
- [ ] Audit logging
- [ ] Incident response plan
- [ ] Vulnerability management
- [ ] Change management

#### GDPR

- [ ] Data processing agreements
- [ ] Right to erasure (data deletion)
- [ ] Data portability (export)
- [ ] Consent management
- [ ] Privacy policy
- [ ] DPO appointment

#### HIPAA (if applicable)

- [ ] BAA with customers
- [ ] PHI encryption
- [ ] Access controls
- [ ] Audit trails
- [ ] Breach notification

## Vulnerability Management

### Dependency Scanning

```yaml
# GitHub Actions for dependency scanning
name: Security Scan
on:
  push:
    branches: [main]
  schedule:
    - cron: "0 0 * * *"

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        severity: 'CRITICAL,HIGH'
        exit-code: '1'

    - name: Run Snyk
      uses: snyk/actions/node@master
      env:
        SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
```

### Container Image Scanning

```yaml
# Scan images before deployment
- name: Scan container image
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: 'ghcr.io/opencode/api:${{ github.sha }}'
    format: 'sarif'
    output: 'trivy-results.sarif'

- name: Upload scan results
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: 'trivy-results.sarif'
```

### Penetration Testing

- Annual third-party penetration tests
- Quarterly internal security assessments
- Bug bounty program for external researchers

## Incident Response

### Incident Classification

| Severity | Description | Response Time |
|----------|-------------|--------------|
| P1 - Critical | Data breach, complete outage | 15 minutes |
| P2 - High | Partial outage, security vulnerability | 1 hour |
| P3 - Medium | Degraded service, minor vulnerability | 4 hours |
| P4 - Low | Cosmetic issues, minor bugs | 24 hours |

### Response Procedures

```typescript
interface IncidentResponse {
  // 1. Detection & Alerting
  detection: {
    source: "monitoring" | "user_report" | "automated_scan"
    alertChannels: ["pagerduty", "slack", "email"]
  }

  // 2. Triage & Classification
  triage: {
    severity: "P1" | "P2" | "P3" | "P4"
    impactAssessment: string
    affectedSystems: string[]
  }

  // 3. Containment
  containment: {
    isolateAffectedSystems: boolean
    preserveEvidence: boolean
    communicateToStakeholders: boolean
  }

  // 4. Eradication
  eradication: {
    rootCauseAnalysis: string
    remediationSteps: string[]
  }

  // 5. Recovery
  recovery: {
    restoreServices: boolean
    verifyIntegrity: boolean
    monitorForRecurrence: boolean
  }

  // 6. Post-Incident
  postIncident: {
    incidentReport: string
    lessonsLearned: string[]
    preventiveMeasures: string[]
  }
}
```

### Security Contacts

```yaml
# PagerDuty escalation policy
escalation_policy:
  name: "Security Incidents"
  escalation_rules:
    - escalation_delay_in_minutes: 5
      targets:
        - type: "user_reference"
          id: "security-oncall"
    - escalation_delay_in_minutes: 15
      targets:
        - type: "user_reference"
          id: "security-lead"
    - escalation_delay_in_minutes: 30
      targets:
        - type: "user_reference"
          id: "cto"
```

## Security Checklist

### Pre-Deployment

- [ ] Security review of architecture
- [ ] Threat modeling complete
- [ ] Penetration test passed
- [ ] Dependency vulnerabilities addressed
- [ ] Secrets rotated and secured
- [ ] Network policies configured
- [ ] TLS certificates valid
- [ ] Audit logging enabled
- [ ] Monitoring alerts configured
- [ ] Incident response plan tested

### Ongoing

- [ ] Weekly dependency updates
- [ ] Monthly security patches
- [ ] Quarterly access reviews
- [ ] Annual penetration tests
- [ ] Continuous vulnerability scanning
- [ ] Regular backup verification
- [ ] Incident response drills
