module github.com/yourorg/microservice-platform/backend

go 1.22

require (
	// gRPC + Protobuf
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.34.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0
	github.com/envoyproxy/protoc-gen-validate v1.0.4
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.3.0

	// Web framework / HTTP
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/go-chi/httprate v0.9.0

	// Configuration
	github.com/spf13/viper v1.18.2
	github.com/spf13/cobra v1.8.0

	// PostgreSQL
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jackc/pgxpool v0.0.0-20221121165030-8a5e56c60dca
	github.com/golang-migrate/migrate/v4 v4.17.1

	// MongoDB
	go.mongodb.org/mongo-driver v1.15.0

	// Redis
	github.com/redis/go-redis/v9 v9.5.1

	// Auth / Security
	github.com/golang-jwt/jwt/v5 v5.2.1
	golang.org/x/crypto v0.23.0

	// Observability
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0
	go.opentelemetry.io/otel/sdk v1.27.0
	go.opentelemetry.io/otel/trace v1.27.0
	github.com/prometheus/client_golang v1.19.1

	// Logging
	go.uber.org/zap v1.27.0
	go.uber.org/zap/exp v0.2.0

	// Validation
	github.com/go-playground/validator/v10 v10.22.0

	// Utilities
	github.com/google/uuid v1.6.0
	github.com/samber/lo v1.39.0
	golang.org/x/sync v0.7.0
)
