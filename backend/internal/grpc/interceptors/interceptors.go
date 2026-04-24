// Package interceptors provides gRPC server-side middleware (interceptors).
package interceptors

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ============================================================
// Prometheus Metrics
// ============================================================

var (
	grpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total number of gRPC requests received.",
	}, []string{"method", "code"})

	grpcRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "grpc_request_duration_seconds",
		Help:    "Duration of gRPC requests.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
)

// ============================================================
// Logging Interceptor
// ============================================================

// UnaryLogging logs each unary RPC call with duration and status code.
func UnaryLogging(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		code := codes.OK
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = s.Code()
			}
		}

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", code.String()),
		}

		if err != nil && code != codes.NotFound {
			logger.Error("grpc: request failed", append(fields, zap.Error(err))...)
		} else {
			logger.Info("grpc: request completed", fields...)
		}

		return resp, err
	}
}

// StreamLogging logs each streaming RPC call.
func StreamLogging(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		logger.Info("grpc: stream completed",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return err
	}
}

// ============================================================
// Recovery Interceptor (panic → Internal error)
// ============================================================

func UnaryRecovery(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("grpc: panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// ============================================================
// Metrics Interceptor
// ============================================================

func UnaryMetrics() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := codes.OK
		if s, ok := status.FromError(err); ok {
			code = s.Code()
		}

		grpcRequestsTotal.WithLabelValues(info.FullMethod, code.String()).Inc()
		grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())

		return resp, err
	}
}

// ============================================================
// Auth Interceptor (JWT validation)
// ============================================================

// publicMethods are gRPC methods that skip JWT authentication.
var publicMethods = map[string]bool{
	"/user.v1.UserService/Login":        true,
	"/user.v1.UserService/RefreshToken": true,
}

// AuthClaims is stored in context after successful JWT validation.
type AuthClaims struct {
	UserID string
	Email  string
	Role   string
}

type contextKey string

const claimsKey contextKey = "auth_claims"

// UnaryAuth validates JWT tokens for protected gRPC methods.
func UnaryAuth(accessSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		claims, err := extractAndValidateToken(ctx, accessSecret)
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, claimsKey, claims)
		return handler(ctx, req)
	}
}

func extractAndValidateToken(ctx context.Context, secret string) (*AuthClaims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader[0], "Bearer ")
	if tokenStr == authHeader[0] {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format, expected Bearer token")
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token claims")
	}

	if mapClaims["type"] != "access" {
		return nil, status.Errorf(codes.Unauthenticated, "wrong token type")
	}

	return &AuthClaims{
		UserID: fmt.Sprintf("%v", mapClaims["sub"]),
		Email:  fmt.Sprintf("%v", mapClaims["email"]),
		Role:   fmt.Sprintf("%v", mapClaims["role"]),
	}, nil
}

// ClaimsFromContext extracts auth claims injected by the auth interceptor.
func ClaimsFromContext(ctx context.Context) (*AuthClaims, bool) {
	c, ok := ctx.Value(claimsKey).(*AuthClaims)
	return c, ok
}

// ============================================================
// Request ID Interceptor
// ============================================================

func UnaryRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		requestID := ""
		if ok {
			ids := md.Get("x-request-id")
			if len(ids) > 0 {
				requestID = ids[0]
			}
		}

		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		// Attach to outgoing response metadata
		grpc.SetHeader(ctx, metadata.Pairs("x-request-id", requestID)) //nolint:errcheck
		return handler(ctx, req)
	}
}
