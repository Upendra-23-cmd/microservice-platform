# ============================================================
# Stage 1: Proto generation
# ============================================================
FROM golang:1.22-alpine AS proto-builder

RUN apk add --no-cache protobuf protobuf-dev curl unzip && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

WORKDIR /proto
COPY proto/ ./proto/
COPY go.mod go.sum ./

RUN mkdir -p internal/grpc/pb && \
    protoc \
      -I ./proto \
      -I /usr/include \
      --go_out=./internal/grpc/pb \
      --go_opt=paths=source_relative \
      --go-grpc_out=./internal/grpc/pb \
      --go-grpc_opt=paths=source_relative \
      --grpc-gateway_out=./internal/grpc/pb \
      --grpc-gateway_opt=paths=source_relative \
      ./proto/*.proto

# ============================================================
# Stage 2: Go dependency cache (separate layer for caching)
# ============================================================
FROM golang:1.22-alpine AS deps

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ============================================================
# Stage 3: Builder
# ============================================================
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

# Copy dependencies from cache layer
COPY --from=deps /go/pkg /go/pkg
COPY go.mod go.sum ./

# Copy generated proto files
COPY --from=proto-builder /proto/internal ./internal

# Copy source code
COPY . .

# Build the binary with hardened flags
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w \
      -X main.Version=${VERSION} \
      -X main.GitCommit=${GIT_COMMIT} \
      -X main.BuildTime=${BUILD_TIME}" \
    -o /bin/server \
    ./cmd/server

# Download grpc_health_probe for Docker health checks
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.25 && \
    wget -qO /bin/grpc_health_probe \
      "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64" && \
    chmod +x /bin/grpc_health_probe

# ============================================================
# Stage 4: Production image (distroless for minimal attack surface)
# ============================================================
FROM gcr.io/distroless/static-debian12:nonroot AS production

# Copy timezone data and CA certs
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy only the binary and health probe
COPY --from=builder /bin/server /bin/server
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

# Run as non-root (UID 65532 is distroless nonroot)
USER nonroot:nonroot

EXPOSE 8080 50051 9090

ENTRYPOINT ["/bin/server"]
