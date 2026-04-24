# 🚀 Enterprise Microservice Platform

> **Production-grade** microservice architecture — React 18 frontend, Go 1.22 gRPC backend, PostgreSQL + MongoDB, Redis, Kubernetes-ready.

[![CI](https://github.com/yourorg/microservice-platform/actions/workflows/ci.yml/badge.svg)](https://github.com/yourorg/microservice-platform/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourorg/microservice-platform/backend)](https://goreportcard.com/report/github.com/yourorg/microservice-platform/backend)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser / Mobile                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │ HTTPS
┌─────────────────────────▼───────────────────────────────────────┐
│              React 18 + TypeScript + Vite SPA                    │
│  Zustand │ React Query │ React Router │ Tailwind │ Recharts      │
└─────────────────────────┬───────────────────────────────────────┘
                          │ REST / JSON  (gRPC-Gateway)
┌─────────────────────────▼───────────────────────────────────────┐
│            Go 1.22 Backend — gRPC + gRPC-Gateway                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ UserService  │  │ProductService│  │   Auth / JWT / RBAC  │   │
│  └──────┬───────┘  └──────┬───────┘  └──────────────────────┘   │
│         │  Domain         │  Domain                              │
│  ┌──────▼───────┐  ┌──────▼────────────────────┐               │
│  │  PostgreSQL  │  │ PostgreSQL + MongoDB (meta)│               │
│  │  Repository  │  │     Repository              │               │
│  └──────┬───────┘  └──────┬────────────────────┘               │
└─────────┼─────────────────┼───────────────────────────────────-─┘
          │                 │
┌─────────▼──────┐ ┌────────▼──────────────────────┐
│ PostgreSQL 16  │ │ MongoDB 7                      │
│ (relational)   │ │ (product metadata, audit logs) │
└────────────────┘ └────────────────────────────────┘
                   ┌────────────────────────────────┐
                   │ Redis 7 (cache + pub/sub)       │
                   └────────────────────────────────┘
                   ┌────────────────────────────────┐
                   │ Jaeger · Prometheus · Grafana   │
                   └────────────────────────────────┘
```

## Database Strategy

| Data Type | Database | Reason |
|-----------|----------|--------|
| Users, Orders, Order Items | **PostgreSQL** | ACID, foreign keys, relational integrity |
| Product core (price, stock) | **PostgreSQL** | Inventory transactions, consistent stock updates |
| Product metadata (tags, SEO, attributes) | **MongoDB** | Schema-flexible, varies per category |
| Audit logs, events | **MongoDB (capped)** | Append-only, no schema rigidity needed |
| Session cache, rate limits | **Redis** | TTL, in-memory speed |
| Domain events | **Redis Pub/Sub** | Decoupled, lightweight messaging |

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | React 18, TypeScript, Vite, Zustand, TanStack Query, Tailwind CSS, Recharts |
| Backend | Go 1.22, gRPC, gRPC-Gateway (REST proxy) |
| Auth | JWT (access + refresh tokens), bcrypt |
| Relational DB | PostgreSQL 16 + pgx/v5 (no ORM — raw SQL) |
| Non-Relational DB | MongoDB 7 + mongo-driver |
| Cache / Pub-Sub | Redis 7 |
| Observability | OpenTelemetry → Jaeger, Prometheus, Grafana |
| Containerisation | Docker (distroless prod images), Docker Compose |
| Orchestration | Kubernetes + HPA + zero-downtime rolling deploys |
| CI/CD | GitHub Actions |
| Linting | golangci-lint (Go), ESLint + tsc (TS) |

## Project Structure

```
microservice-platform/
├── .github/workflows/ci.yml          # CI/CD pipeline
├── docker-compose.yml                # Full local stack
├── Makefile                          # All dev/build/deploy commands
│
├── frontend/                         # React + TypeScript SPA
│   ├── src/
│   │   ├── components/
│   │   │   ├── layout/AppLayout.tsx  # Sidebar + routing shell
│   │   │   ├── features/             # Page-level components
│   │   │   │   ├── auth/LoginPage.tsx
│   │   │   │   ├── dashboard/DashboardPage.tsx
│   │   │   │   ├── users/UsersPage.tsx
│   │   │   │   └── products/ProductsPage.tsx
│   │   │   └── ui/                   # Shared UI primitives
│   │   ├── hooks/                    # React Query hooks
│   │   ├── services/                 # API clients (Axios)
│   │   ├── store/                    # Zustand stores
│   │   ├── types/                    # TypeScript domain types
│   │   └── utils/                   # Helpers (cn, etc.)
│   ├── Dockerfile                    # Multi-stage nginx build
│   ├── nginx.conf                    # SPA routing + security headers
│   └── vite.config.ts
│
└── backend/                          # Go microservice
    ├── cmd/server/main.go            # Entry point + DI wiring
    ├── internal/
    │   ├── config/config.go          # Env-var config with Viper
    │   ├── domain/                   # DDD: aggregates + ports
    │   │   ├── user/user.go
    │   │   └── product/product.go
    │   ├── grpc/
    │   │   ├── handlers/             # gRPC service implementations
    │   │   └── interceptors/         # Auth, logging, metrics, recovery
    │   ├── http/                     # gRPC-Gateway REST handlers
    │   ├── repository/
    │   │   ├── postgres/             # PostgreSQL adapters (pgx/v5)
    │   │   └── mongodb/              # MongoDB adapters
    │   ├── service/                  # Application business logic
    │   └── messaging/redis.go        # Cache + Event Bus
    ├── pkg/
    │   ├── errors/errors.go          # Typed application errors
    │   ├── logger/logger.go          # Structured zap logger
    │   ├── tracing/tracing.go        # OpenTelemetry setup
    │   └── validator/validator.go    # Input validation
    ├── proto/                        # Protobuf definitions
    │   ├── user.proto
    │   └── product.proto
    ├── migrations/
    │   ├── postgres/                 # SQL migrations (up + down)
    │   └── mongodb/init.js           # Collection + index init
    └── deployments/
        ├── docker/
        │   ├── Dockerfile            # Multi-stage, distroless
        │   └── prometheus.yml
        └── k8s/backend.yaml          # Deployment + Service + HPA + Ingress
```

## Quick Start

### Prerequisites

- Docker + Docker Compose v2
- Go 1.22+, Node 20+, protoc + plugins (for development)

### 1 — Clone & configure

```bash
git clone https://github.com/yourorg/microservice-platform
cd microservice-platform

cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
# Edit backend/.env — set secrets (JWT, passwords)
```

### 2 — Start everything

```bash
make docker-up
```

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| HTTP API (REST) | http://localhost:8080/api/v1 |
| gRPC | localhost:50051 |
| Prometheus | http://localhost:9091 |
| Grafana | http://localhost:3001 (admin/admin) |
| Jaeger | http://localhost:16686 |

### 3 — Run database migrations

```bash
make migrate-up
```

### 4 — Regenerate proto files (after editing .proto)

```bash
make proto-gen
```

## Development

```bash
# Backend — hot reload with air
cd backend && air

# Frontend — Vite HMR
make frontend-dev

# Run all tests
make test

# Full CI pipeline locally
make ci
```

## gRPC API Usage

```bash
# List users (requires auth token)
grpcurl -H "Authorization: Bearer $TOKEN" \
  -d '{"page":1,"page_size":10}' \
  localhost:50051 user.v1.UserService/ListUsers

# Login
grpcurl -plaintext \
  -d '{"email":"admin@example.com","password":"YourPass1!"}' \
  localhost:50051 user.v1.UserService/Login
```

## Environment Variables

All configuration is driven by environment variables. See `backend/.env.example` for the complete reference. No value is hardcoded — all defaults are overridable.

Key groups:
- `APP_*` — application metadata
- `GRPC_*` — gRPC server settings
- `HTTP_*` — REST gateway settings
- `POSTGRES_*` — PostgreSQL connection pool
- `MONGODB_*` — MongoDB connection pool
- `REDIS_*` — Redis connection + pool
- `JWT_*` — token secrets and expiry
- `TLS_*` — certificate paths
- `RATE_LIMIT_*` — RPS limits
- `CORS_*` — allowed origins

## Production Checklist

- [ ] Rotate all secrets in `.env` / Kubernetes Secrets
- [ ] Enable TLS (`TLS_ENABLED=true`) with valid certificates
- [ ] Set `GRPC_REFLECTION_ENABLED=false`
- [ ] Point `OTEL_EXPORTER_OTLP_ENDPOINT` to your Jaeger/OTLP collector
- [ ] Configure `CORS_ALLOWED_ORIGINS` to your domain only
- [ ] Replace Kubernetes `stringData` secrets with Sealed Secrets or Vault
- [ ] Set resource requests/limits in `k8s/backend.yaml`
- [ ] Configure HPA min/max replicas for your traffic profile
- [ ] Enable PodDisruptionBudget for zero-downtime maintenance

## License

MIT
