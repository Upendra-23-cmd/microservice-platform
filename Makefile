# ============================================================
# Microservice Platform - Makefile
# 40-year engineer standard: every common task is one command
# ============================================================

.DEFAULT_GOAL := help
.PHONY: help proto-gen build run test lint migrate-up migrate-down \
        docker-build docker-up docker-down docker-logs clean \
        frontend-dev frontend-build coverage

# ── Variables ─────────────────────────────────────────────────
BINARY        := server
BUILD_DIR     := ./bin
CMD_DIR       := ./cmd/server
PROTO_DIR     := ./backend/proto
PB_OUT_DIR    := ./backend/internal/grpc/pb
MIGRATION_DIR := ./backend/migrations/postgres
DOCKER_IMAGE  := microservice-platform/backend

GIT_COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION       := $(shell git describe --tags --always 2>/dev/null || echo "v0.0.0")
BUILD_TIME    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.BuildTime=$(BUILD_TIME)

GO       := go
GOFLAGS  := -trimpath
GOTEST   := $(GO) test -race -timeout=120s

# ── Help ──────────────────────────────────────────────────────
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ── Proto ─────────────────────────────────────────────────────
proto-gen: ## Generate Go code from .proto files
	@echo "→ Generating protobuf files..."
	@mkdir -p $(PB_OUT_DIR)
	@protoc \
		-I $(PROTO_DIR) \
		-I /usr/local/include \
		--go_out=$(PB_OUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PB_OUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(PB_OUT_DIR) \
		--grpc-gateway_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto
	@echo "✓ Proto generation complete"

proto-lint: ## Lint proto files using buf
	@buf lint $(PROTO_DIR)

# ── Build ─────────────────────────────────────────────────────
build: ## Build the backend binary
	@echo "→ Building $(BINARY) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@cd backend && CGO_ENABLED=0 $(GO) build $(GOFLAGS) \
		-ldflags="$(LDFLAGS)" \
		-o ../$(BUILD_DIR)/$(BINARY) \
		$(CMD_DIR)
	@echo "✓ Built $(BUILD_DIR)/$(BINARY)"

build-linux: ## Cross-compile for Linux/amd64
	@mkdir -p $(BUILD_DIR)
	@cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) \
		-ldflags="$(LDFLAGS)" \
		-o ../$(BUILD_DIR)/$(BINARY)-linux-amd64 \
		$(CMD_DIR)

run: build ## Build and run the backend locally
	@$(BUILD_DIR)/$(BINARY)

# ── Test ──────────────────────────────────────────────────────
test: ## Run all tests
	@echo "→ Running tests..."
	@cd backend && $(GOTEST) ./...

test-unit: ## Run unit tests only (no integration)
	@cd backend && $(GOTEST) -short ./...

test-integration: ## Run integration tests (requires running infra)
	@cd backend && $(GOTEST) -run Integration ./...

coverage: ## Run tests with HTML coverage report
	@cd backend && $(GOTEST) -coverprofile=coverage.out ./...
	@cd backend && $(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: backend/coverage.html"

# ── Lint ──────────────────────────────────────────────────────
lint: ## Run golangci-lint
	@echo "→ Linting..."
	@cd backend && golangci-lint run --timeout=5m ./...
	@echo "✓ Lint passed"

fmt: ## Format all Go files
	@cd backend && $(GO) fmt ./...
	@cd backend && goimports -w .

vet: ## Run go vet
	@cd backend && $(GO) vet ./...

# ── Migrations ────────────────────────────────────────────────
migrate-up: ## Run all pending PostgreSQL migrations
	@echo "→ Running migrations UP..."
	@migrate -path $(MIGRATION_DIR) \
		-database "$(shell grep POSTGRES_DSN backend/.env | cut -d= -f2-)" \
		up
	@echo "✓ Migrations applied"

migrate-down: ## Roll back last PostgreSQL migration
	@migrate -path $(MIGRATION_DIR) \
		-database "$(shell grep POSTGRES_DSN backend/.env | cut -d= -f2-)" \
		down 1

migrate-status: ## Show migration status
	@migrate -path $(MIGRATION_DIR) \
		-database "$(shell grep POSTGRES_DSN backend/.env | cut -d= -f2-)" \
		version

# ── Docker ────────────────────────────────────────────────────
docker-build: ## Build Docker image for backend
	@echo "→ Building Docker image $(DOCKER_IMAGE):$(VERSION)..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-f backend/deployments/docker/Dockerfile \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		./backend
	@echo "✓ Image built: $(DOCKER_IMAGE):$(VERSION)"

docker-up: ## Start all services via Docker Compose
	@docker compose up -d --build
	@echo "✓ Services started"
	@echo "  Frontend:    http://localhost:3000"
	@echo "  HTTP API:    http://localhost:8080"
	@echo "  gRPC:        localhost:50051"
	@echo "  Prometheus:  http://localhost:9091"
	@echo "  Grafana:     http://localhost:3001"
	@echo "  Jaeger:      http://localhost:16686"

docker-down: ## Stop and remove all services
	@docker compose down -v
	@echo "✓ Services stopped"

docker-logs: ## Tail logs for all services
	@docker compose logs -f

docker-ps: ## Show running service status
	@docker compose ps

# ── Frontend ──────────────────────────────────────────────────
frontend-install: ## Install frontend dependencies
	@cd frontend && npm ci

frontend-dev: ## Start frontend dev server
	@cd frontend && npm run dev

frontend-build: ## Build frontend for production
	@cd frontend && npm run build

frontend-lint: ## Lint frontend code
	@cd frontend && npm run lint

frontend-test: ## Run frontend tests
	@cd frontend && npm run test

# ── Database Utilities ────────────────────────────────────────
db-shell: ## Open psql shell
	@docker compose exec postgres psql -U app_user -d microservice_db

mongo-shell: ## Open mongosh shell
	@docker compose exec mongodb mongosh -u app_user -p changeme microservice_db

redis-cli: ## Open Redis CLI
	@docker compose exec redis redis-cli -a changeme

# ── Security ──────────────────────────────────────────────────
security-scan: ## Run govulncheck for known vulnerabilities
	@cd backend && govulncheck ./...

trivy-scan: ## Scan Docker image for vulnerabilities
	@trivy image $(DOCKER_IMAGE):latest

# ── Clean ─────────────────────────────────────────────────────
clean: ## Remove build artifacts
	@rm -rf $(BUILD_DIR) backend/coverage.out backend/coverage.html
	@echo "✓ Clean complete"

# ── CI helpers ────────────────────────────────────────────────
ci: fmt vet lint test build ## Run full CI pipeline locally
	@echo "✓ CI pipeline passed"
