# Scaling & Deployment

## Overview

This document covers horizontal scaling strategies, deployment patterns, and operational considerations for running OpenCode as a production web service.

## Scaling Architecture

### Horizontal Scaling Model

```
                         ┌─────────────────┐
                         │  Global LB      │
                         │ (Cloudflare)    │
                         └────────┬────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              │                   │                   │
     ┌────────▼────────┐ ┌────────▼────────┐ ┌────────▼────────┐
     │   Region: US    │ │   Region: EU    │ │   Region: APAC  │
     └────────┬────────┘ └────────┬────────┘ └────────┬────────┘
              │                   │                   │
     ┌────────▼────────┐ ┌────────▼────────┐ ┌────────▼────────┐
     │ K8s Cluster     │ │ K8s Cluster     │ │ K8s Cluster     │
     │ ┌─────────────┐ │ │ ┌─────────────┐ │ │ ┌─────────────┐ │
     │ │ API Pods    │ │ │ │ API Pods    │ │ │ │ API Pods    │ │
     │ │ (3-20)      │ │ │ │ (3-20)      │ │ │ │ (3-20)      │ │
     │ └─────────────┘ │ │ └─────────────┘ │ │ └─────────────┘ │
     │ ┌─────────────┐ │ │ ┌─────────────┐ │ │ ┌─────────────┐ │
     │ │ Worker Pods │ │ │ │ Worker Pods │ │ │ │ Worker Pods │ │
     │ │ (2-10)      │ │ │ │ (2-10)      │ │ │ │ (2-10)      │ │
     │ └─────────────┘ │ │ └─────────────┘ │ │ └─────────────┘ │
     └─────────────────┘ └─────────────────┘ └─────────────────┘
```

### Component Scaling Characteristics

| Component | Scaling Type | Trigger | Min/Max |
|-----------|-------------|---------|---------|
| API Server | Horizontal | CPU/Memory | 3/50 |
| Tool Workers | Horizontal | Queue depth | 2/20 |
| WebSocket Handlers | Horizontal | Connection count | 2/20 |
| PostgreSQL | Vertical + Read Replicas | CPU/Connections | 1 primary |
| Redis | Cluster | Memory | 3 nodes |

## Kubernetes Deployment

### Namespace Structure

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: opencode
  labels:
    istio-injection: enabled
---
apiVersion: v1
kind: Namespace
metadata:
  name: opencode-workers
  labels:
    istio-injection: enabled
```

### API Server Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opencode-api
  namespace: opencode
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: opencode-api
  template:
    metadata:
      labels:
        app: opencode-api
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      serviceAccountName: opencode-api
      containers:
      - name: api
        image: ghcr.io/opencode/api:latest
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: 3000
        - name: metrics
          containerPort: 9090
        env:
        - name: NODE_ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: opencode-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: opencode-secrets
              key: redis-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: opencode-secrets
              key: jwt-secret
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 3000
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 10"]
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: opencode-api
              topologyKey: kubernetes.io/hostname
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        whenUnsatisfiable: ScheduleAnyway
        labelSelector:
          matchLabels:
            app: opencode-api
```

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: opencode-api-hpa
  namespace: opencode
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: opencode-api
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
      - type: Pods
        value: 4
        periodSeconds: 15
      selectPolicy: Max
```

### Tool Worker Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opencode-worker
  namespace: opencode-workers
spec:
  replicas: 2
  selector:
    matchLabels:
      app: opencode-worker
  template:
    metadata:
      labels:
        app: opencode-worker
    spec:
      serviceAccountName: opencode-worker
      containers:
      - name: worker
        image: ghcr.io/opencode/worker:latest
        env:
        - name: WORKER_TYPE
          value: "tool-execution"
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: opencode-secrets
              key: redis-url
        resources:
          requests:
            memory: "1Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "4000m"
        securityContext:
          privileged: false
          runAsNonRoot: true
          readOnlyRootFilesystem: true
        volumeMounts:
        - name: workspace
          mountPath: /workspace
        - name: tmp
          mountPath: /tmp
      volumes:
      - name: workspace
        emptyDir:
          sizeLimit: 10Gi
      - name: tmp
        emptyDir:
          sizeLimit: 1Gi
```

### Service & Ingress

```yaml
apiVersion: v1
kind: Service
metadata:
  name: opencode-api
  namespace: opencode
spec:
  selector:
    app: opencode-api
  ports:
  - name: http
    port: 80
    targetPort: 3000
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: opencode-api
  namespace: opencode
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.opencode.io
    secretName: opencode-tls
  rules:
  - host: api.opencode.io
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: opencode-api
            port:
              number: 80
```

## Database Scaling

### PostgreSQL High Availability

```yaml
# Using CloudNativePG operator
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: opencode-postgres
  namespace: opencode
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised

  postgresql:
    parameters:
      max_connections: "200"
      shared_buffers: "2GB"
      effective_cache_size: "6GB"
      maintenance_work_mem: "512MB"
      checkpoint_completion_target: "0.9"
      wal_buffers: "64MB"
      default_statistics_target: "100"
      random_page_cost: "1.1"
      effective_io_concurrency: "200"
      work_mem: "10MB"
      min_wal_size: "1GB"
      max_wal_size: "4GB"

  storage:
    size: 100Gi
    storageClass: fast-ssd

  backup:
    barmanObjectStore:
      destinationPath: s3://opencode-backups/postgres
      s3Credentials:
        accessKeyId:
          name: aws-creds
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: aws-creds
          key: SECRET_ACCESS_KEY
      wal:
        compression: gzip
      data:
        compression: gzip
    retentionPolicy: "30d"

  monitoring:
    enablePodMonitor: true
```

### Read Replica Configuration

```typescript
// Database client with read replica routing
const db = createDatabase({
  primary: {
    connectionString: process.env.DATABASE_URL,
    poolSize: 10,
  },
  replicas: [
    {
      connectionString: process.env.DATABASE_REPLICA_1_URL,
      poolSize: 20,
    },
    {
      connectionString: process.env.DATABASE_REPLICA_2_URL,
      poolSize: 20,
    },
  ],
  // Route read queries to replicas
  router: (query) => {
    if (query.type === "SELECT" && !query.inTransaction) {
      return "replica"
    }
    return "primary"
  },
})
```

### Connection Pooling with PgBouncer

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbouncer
  namespace: opencode
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pgbouncer
  template:
    spec:
      containers:
      - name: pgbouncer
        image: pgbouncer/pgbouncer:latest
        ports:
        - containerPort: 5432
        env:
        - name: PGBOUNCER_POOL_MODE
          value: "transaction"
        - name: PGBOUNCER_MAX_CLIENT_CONN
          value: "1000"
        - name: PGBOUNCER_DEFAULT_POOL_SIZE
          value: "20"
        - name: PGBOUNCER_MIN_POOL_SIZE
          value: "5"
```

## Redis Scaling

### Redis Cluster

```yaml
apiVersion: redis.redis.opstreelabs.in/v1beta1
kind: RedisCluster
metadata:
  name: opencode-redis
  namespace: opencode
spec:
  clusterSize: 3
  clusterVersion: v7
  persistenceEnabled: true
  kubernetesConfig:
    image: redis:7-alpine
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
      limits:
        cpu: 1000m
        memory: 2Gi
  storage:
    volumeClaimTemplate:
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
  redisExporter:
    enabled: true
    image: oliver006/redis_exporter:latest
```

## Load Balancing

### Global Load Balancing (Cloudflare)

```typescript
// Cloudflare Worker for intelligent routing
export default {
  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url)

    // Determine best region based on latency
    const region = request.cf?.region || "us"
    const backend = getBackendForRegion(region)

    // Add request tracing
    const headers = new Headers(request.headers)
    headers.set("x-request-id", crypto.randomUUID())
    headers.set("x-forwarded-region", region)

    return fetch(backend + url.pathname + url.search, {
      method: request.method,
      headers,
      body: request.body,
    })
  },
}

function getBackendForRegion(region: string): string {
  const backends = {
    us: "https://us.api.opencode.io",
    eu: "https://eu.api.opencode.io",
    apac: "https://apac.api.opencode.io",
  }
  return backends[region] || backends.us
}
```

### Internal Load Balancing

```yaml
# Istio VirtualService for traffic management
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: opencode-api
  namespace: opencode
spec:
  hosts:
  - opencode-api
  http:
  - match:
    - headers:
        x-api-version:
          exact: "v2"
    route:
    - destination:
        host: opencode-api-v2
        port:
          number: 80
  - route:
    - destination:
        host: opencode-api
        port:
          number: 80
      weight: 100
    retries:
      attempts: 3
      perTryTimeout: 10s
      retryOn: 5xx,reset,connect-failure
    timeout: 300s
```

## SSE Connection Scaling

### Sticky Sessions for SSE

```yaml
# Nginx Ingress with sticky sessions
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: opencode-events
  annotations:
    nginx.ingress.kubernetes.io/affinity: "cookie"
    nginx.ingress.kubernetes.io/session-cookie-name: "opencode-route"
    nginx.ingress.kubernetes.io/session-cookie-expires: "172800"
    nginx.ingress.kubernetes.io/session-cookie-max-age: "172800"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
spec:
  rules:
  - host: events.opencode.io
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: opencode-api
            port:
              number: 80
```

### Connection Draining

```typescript
// Graceful shutdown with connection draining
const connections = new Set<SSEConnection>()

async function gracefulShutdown(): Promise<void> {
  // Stop accepting new connections
  server.close()

  // Notify existing connections
  for (const conn of connections) {
    conn.send({ type: "server.shutdown", reconnectIn: 5000 })
  }

  // Wait for connections to drain (max 30s)
  const deadline = Date.now() + 30000
  while (connections.size > 0 && Date.now() < deadline) {
    await sleep(1000)
  }

  // Force close remaining
  for (const conn of connections) {
    conn.close()
  }

  process.exit(0)
}

process.on("SIGTERM", gracefulShutdown)
```

## Monitoring & Observability

### Prometheus Metrics

```typescript
import { Registry, Counter, Histogram, Gauge } from "prom-client"

const registry = new Registry()

// Request metrics
const httpRequestsTotal = new Counter({
  name: "http_requests_total",
  help: "Total HTTP requests",
  labelNames: ["method", "path", "status"],
  registers: [registry],
})

const httpRequestDuration = new Histogram({
  name: "http_request_duration_seconds",
  help: "HTTP request duration",
  labelNames: ["method", "path"],
  buckets: [0.01, 0.05, 0.1, 0.5, 1, 5, 10],
  registers: [registry],
})

// Business metrics
const activeSessions = new Gauge({
  name: "opencode_active_sessions",
  help: "Number of active sessions",
  registers: [registry],
})

const llmTokensTotal = new Counter({
  name: "opencode_llm_tokens_total",
  help: "Total LLM tokens consumed",
  labelNames: ["provider", "model", "type"],
  registers: [registry],
})

const toolExecutionDuration = new Histogram({
  name: "opencode_tool_execution_seconds",
  help: "Tool execution duration",
  labelNames: ["tool"],
  buckets: [0.1, 0.5, 1, 5, 10, 30, 60],
  registers: [registry],
})
```

### Grafana Dashboards

```json
{
  "title": "OpenCode Overview",
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(http_requests_total[5m])) by (status)",
          "legendFormat": "{{status}}"
        }
      ]
    },
    {
      "title": "P99 Latency",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))"
        }
      ]
    },
    {
      "title": "Active Sessions",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(opencode_active_sessions)"
        }
      ]
    },
    {
      "title": "Token Usage",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(opencode_llm_tokens_total[1h])) by (provider)",
          "legendFormat": "{{provider}}"
        }
      ]
    }
  ]
}
```

### Distributed Tracing

```typescript
import { trace, SpanKind } from "@opentelemetry/api"

const tracer = trace.getTracer("opencode-api")

async function handleChatRequest(ctx: Context): Promise<Response> {
  return tracer.startActiveSpan(
    "chat.request",
    { kind: SpanKind.SERVER },
    async (span) => {
      try {
        span.setAttributes({
          "session.id": ctx.params.id,
          "user.id": ctx.get("tenant").userId,
        })

        // Process request with child spans
        const messages = await tracer.startActiveSpan(
          "load.messages",
          async (loadSpan) => {
            const result = await loadMessages(ctx.params.id)
            loadSpan.end()
            return result
          }
        )

        const response = await tracer.startActiveSpan(
          "llm.request",
          { kind: SpanKind.CLIENT },
          async (llmSpan) => {
            llmSpan.setAttributes({
              "llm.provider": "anthropic",
              "llm.model": "claude-3-sonnet",
            })
            const result = await callLLM(messages)
            llmSpan.setAttributes({
              "llm.tokens.input": result.tokens.input,
              "llm.tokens.output": result.tokens.output,
            })
            llmSpan.end()
            return result
          }
        )

        span.setStatus({ code: SpanStatusCode.OK })
        return ctx.json(response)
      } catch (error) {
        span.setStatus({ code: SpanStatusCode.ERROR, message: error.message })
        throw error
      } finally {
        span.end()
      }
    }
  )
}
```

## Deployment Strategies

### Blue-Green Deployment

```yaml
# Argo Rollouts for blue-green deployment
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: opencode-api
spec:
  replicas: 5
  strategy:
    blueGreen:
      activeService: opencode-api
      previewService: opencode-api-preview
      autoPromotionEnabled: false
      scaleDownDelaySeconds: 30
      previewReplicaCount: 2
      prePromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: opencode-api-preview
```

### Canary Deployment

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: opencode-api
spec:
  strategy:
    canary:
      steps:
      - setWeight: 5
      - pause: { duration: 5m }
      - setWeight: 20
      - pause: { duration: 10m }
      - setWeight: 50
      - pause: { duration: 10m }
      - setWeight: 100
      analysis:
        templates:
        - templateName: success-rate
        startingStep: 1
      canaryService: opencode-api-canary
      stableService: opencode-api
```

## Disaster Recovery

### Multi-Region Failover

```typescript
// Health check and failover logic
interface RegionHealth {
  region: string
  healthy: boolean
  latency: number
  lastCheck: Date
}

class RegionManager {
  private regions: Map<string, RegionHealth> = new Map()

  async checkHealth(region: string): Promise<RegionHealth> {
    const start = Date.now()
    try {
      const response = await fetch(`https://${region}.api.opencode.io/health`)
      return {
        region,
        healthy: response.ok,
        latency: Date.now() - start,
        lastCheck: new Date(),
      }
    } catch {
      return {
        region,
        healthy: false,
        latency: -1,
        lastCheck: new Date(),
      }
    }
  }

  getBestRegion(): string {
    const healthy = Array.from(this.regions.values())
      .filter((r) => r.healthy)
      .sort((a, b) => a.latency - b.latency)

    return healthy[0]?.region || "us" // fallback
  }
}
```

### RTO/RPO Targets

| Scenario | RTO | RPO |
|----------|-----|-----|
| Single pod failure | 0 (auto-recovery) | 0 |
| Node failure | 2 minutes | 0 |
| AZ failure | 5 minutes | 0 |
| Region failure | 15 minutes | 1 minute |
| Complete outage | 1 hour | 5 minutes |
