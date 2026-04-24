// Command server is the single binary entrypoint for the microservice.
// It wires all dependencies, starts the gRPC server and HTTP gateway,
// then performs graceful shutdown on OS signal.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/yourorg/microservice-platform/backend/internal/config"
	"github.com/yourorg/microservice-platform/backend/internal/grpc/handlers"
	"github.com/yourorg/microservice-platform/backend/internal/grpc/interceptors"
	pbuser "github.com/yourorg/microservice-platform/backend/internal/grpc/pb/user/v1"
	pbproduct "github.com/yourorg/microservice-platform/backend/internal/grpc/pb/product/v1"
	"github.com/yourorg/microservice-platform/backend/internal/repository/mongodb"
	"github.com/yourorg/microservice-platform/backend/internal/repository/postgres"
	"github.com/yourorg/microservice-platform/backend/internal/service"
	"github.com/yourorg/microservice-platform/backend/pkg/logger"
)

func main() {
	// ── Config ───────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: config: %v\n", err)
		os.Exit(1)
	}

	// ── Logger ───────────────────────────────────────────────
	log := logger.MustNew(cfg.App.LogLevel, cfg.App.Env)
	defer log.Sync() //nolint:errcheck

	log.Info("starting microservice",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env),
	)

	// ── Context for graceful shutdown ─────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── PostgreSQL ───────────────────────────────────────────
	pgPool, err := postgres.NewPool(ctx, cfg.Postgres, log)
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pgPool.Close()

	// ── MongoDB ──────────────────────────────────────────────
	mongoClient, err := mongodb.NewClient(ctx, cfg.MongoDB, log)
	if err != nil {
		log.Fatal("failed to connect to mongodb", zap.Error(err))
	}
	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = mongoClient.Disconnect(shutCtx)
	}()

	// ── Repositories ─────────────────────────────────────────
	userRepo := postgres.NewUserRepository(pgPool, log)
	productMetaRepo := mongodb.NewProductMetadataRepository(mongoClient, cfg.MongoDB.DB, log)

	// Ensure MongoDB indexes exist
	if err := productMetaRepo.EnsureIndexes(ctx); err != nil {
		log.Fatal("failed to ensure mongodb indexes", zap.Error(err))
	}

	// ── Services ─────────────────────────────────────────────
	userSvc := service.NewUserService(userRepo, cfg, log)
	productSvc := service.NewProductService(
		postgres.NewProductRepository(pgPool, log),
		productMetaRepo,
		cfg,
		log,
	)

	// ── gRPC Handlers ─────────────────────────────────────────
	userHandler := handlers.NewUserHandler(userSvc, log)
	productHandler := handlers.NewProductHandler(productSvc, log)

	// ── gRPC Server ───────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.GRPC.MaxSendMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.GRPC.KeepaliveTime,
			Timeout: cfg.GRPC.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRecovery(log),
			interceptors.UnaryRequestID(),
			interceptors.UnaryLogging(log),
			interceptors.UnaryMetrics(),
			interceptors.UnaryAuth(cfg.JWT.AccessSecret),
		),
		grpc.ChainStreamInterceptor(
			interceptors.StreamLogging(log),
		),
	)

	// Register service implementations
	pbuser.RegisterUserServiceServer(grpcServer, userHandler)
	pbproduct.RegisterProductServiceServer(grpcServer, productHandler)

	// Health check service
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection in non-production (useful for grpcurl)
	if cfg.GRPC.ReflectionEnabled || cfg.App.Env != "production" {
		reflection.Register(grpcServer)
		log.Warn("grpc reflection enabled — disable in production")
	}

	// ── HTTP Gateway (gRPC-Gateway) ────────────────────────────
	gwMux := runtime.NewServeMux(
		runtime.WithErrorHandler(runtime.DefaultHTTPErrorHandler),
	)

	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcAddr := cfg.GRPC.Addr()

	if err := pbuser.RegisterUserServiceHandlerFromEndpoint(ctx, gwMux, grpcAddr, dialOpts); err != nil {
		log.Fatal("failed to register user gateway", zap.Error(err))
	}
	if err := pbproduct.RegisterProductServiceHandlerFromEndpoint(ctx, gwMux, grpcAddr, dialOpts); err != nil {
		log.Fatal("failed to register product gateway", zap.Error(err))
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/api/", gwMux)
	httpMux.Handle("/metrics", promhttp.Handler())
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})
	httpMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check DB connectivity
		if err := pgPool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"postgres unavailable"}`)) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`)) //nolint:errcheck
	})

	httpServer := &http.Server{
		Addr:         cfg.HTTP.Addr(),
		Handler:      httpMux,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	// ── Start Servers ─────────────────────────────────────────
	g, gCtx := errgroup.WithContext(ctx)

	// gRPC listener
	g.Go(func() error {
		lis, err := net.Listen("tcp", cfg.GRPC.Addr())
		if err != nil {
			return fmt.Errorf("grpc: listen: %w", err)
		}
		log.Info("grpc server listening", zap.String("addr", cfg.GRPC.Addr()))
		return grpcServer.Serve(lis)
	})

	// HTTP gateway listener
	g.Go(func() error {
		log.Info("http gateway listening", zap.String("addr", cfg.HTTP.Addr()))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http: serve: %w", err)
		}
		return nil
	})

	// Shutdown watcher
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-quit:
			log.Info("received shutdown signal", zap.String("signal", sig.String()))
		case <-gCtx.Done():
		}

		log.Info("initiating graceful shutdown...")
		cancel()

		// Stop gRPC (drains in-flight RPCs)
		grpcServer.GracefulStop()
		log.Info("grpc server stopped")

		// Stop HTTP
		shutCtx, shutCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer shutCancel()
		if err := httpServer.Shutdown(shutCtx); err != nil {
			log.Error("http server shutdown error", zap.Error(err))
		}
		log.Info("http server stopped")

		return nil
	})

	if err := g.Wait(); err != nil {
		log.Error("server error", zap.Error(err))
		os.Exit(1)
	}

	log.Info("shutdown complete")
}
