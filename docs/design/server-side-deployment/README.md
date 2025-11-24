# OpenCode Server-Side Web Service Design

## Overview

This document describes the architecture for deploying OpenCode as a multi-tenant web service, enabling organizations to provide AI-powered coding assistance to multiple users through a centralized, scalable deployment.

## Goals

1. **Multi-tenancy**: Support multiple users and organizations with proper isolation
2. **Scalability**: Handle thousands of concurrent users with horizontal scaling
3. **Security**: Enterprise-grade authentication, authorization, and data protection
4. **Reliability**: High availability with fault tolerance and disaster recovery
5. **Observability**: Comprehensive monitoring, logging, and tracing

## Current Architecture vs. Target Architecture

| Aspect | Current (Desktop/CLI) | Target (Web Service) |
|--------|----------------------|---------------------|
| Users | Single user | Multi-tenant |
| Storage | Local filesystem (JSON) | Distributed database |
| Auth | Provider API keys only | User auth + provider delegation |
| Scaling | Single instance | Horizontal scaling |
| State | Per-directory instance | Per-user/workspace scoped |
| Networking | Local only | Internet-facing |

## Design Documents

1. **[Architecture](./architecture.md)** - System architecture and component design
2. **[Authentication](./authentication.md)** - User authentication and authorization
3. **[Storage](./storage.md)** - Data persistence and caching strategies (PostgreSQL)
4. **[Storage - MySQL](./storage-mysql.md)** - Alternative MySQL design with BIGINT keys
5. **[Scaling](./scaling.md)** - Horizontal scaling and deployment patterns
6. **[Security](./security.md)** - Security controls and compliance
7. **[API](./api.md)** - API design and versioning

## High-Level Architecture

```
                                    ┌─────────────────┐
                                    │   CDN/WAF       │
                                    │  (Cloudflare)   │
                                    └────────┬────────┘
                                             │
                                    ┌────────▼────────┐
                                    │  Load Balancer  │
                                    │   (L7/HTTP)     │
                                    └────────┬────────┘
                                             │
                    ┌────────────────────────┼────────────────────────┐
                    │                        │                        │
           ┌────────▼────────┐      ┌────────▼────────┐      ┌────────▼────────┐
           │   API Server    │      │   API Server    │      │   API Server    │
           │   (Stateless)   │      │   (Stateless)   │      │   (Stateless)   │
           └────────┬────────┘      └────────┬────────┘      └────────┬────────┘
                    │                        │                        │
                    └────────────────────────┼────────────────────────┘
                                             │
              ┌──────────────┬───────────────┼───────────────┬──────────────┐
              │              │               │               │              │
     ┌────────▼────────┐ ┌───▼───┐   ┌───────▼───────┐  ┌────▼────┐  ┌──────▼──────┐
     │   PostgreSQL    │ │ Redis │   │  Object Store │  │  Queue  │  │   Metrics   │
     │   (Sessions)    │ │(Cache)│   │  (S3/R2/GCS)  │  │ (NATS)  │  │ (Prometheus)│
     └─────────────────┘ └───────┘   └───────────────┘  └─────────┘  └─────────────┘
```

## Key Design Decisions

### 1. Stateless API Servers
API servers are stateless, enabling horizontal scaling. Session state is stored in Redis, persistent data in PostgreSQL.

### 2. Workspace-Based Multi-Tenancy
Each user has isolated workspaces. Workspaces contain projects, sessions, and configurations.

### 3. Federated LLM Provider Access
Users can bring their own API keys or use organization-provided quotas with usage tracking.

### 4. Event-Driven Architecture
Real-time updates via Server-Sent Events (SSE) with Redis Pub/Sub for cross-instance coordination.

### 5. Git-First Project Model
Projects are identified by Git repositories. The service can integrate with GitHub/GitLab for workspace provisioning.

## Deployment Options

1. **Kubernetes** - Recommended for production (see [scaling.md](./scaling.md))
2. **Docker Compose** - Development and small deployments
3. **Serverless** - AWS Lambda/Cloudflare Workers for specific endpoints

## Getting Started

See the individual design documents for detailed specifications:

- Start with [Architecture](./architecture.md) for system overview
- Review [Authentication](./authentication.md) for auth implementation
- Check [Security](./security.md) for compliance requirements
