package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/yourorg/microservice-platform/backend/internal/config"
)

// NewPool creates and validates a pgxpool.Pool from application config.
// It performs a connectivity check (Ping) before returning.
func NewPool(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("postgres: parse DSN: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxOpenConns
	poolCfg.MinConns = cfg.MaxIdleConns
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = cfg.ConnMaxIdleTime

	// Tracing hook — logs slow queries (>200ms)
	poolCfg.BeforeAcquire = func(ctx context.Context, conn *pgxpool.Conn) bool {
		return true
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	// Connectivity check with retries
	if err := pingWithRetry(ctx, pool, 5, 2*time.Second, logger); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping failed: %w", err)
	}

	logger.Info("postgres: connected",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("db", cfg.DB),
	)

	return pool, nil
}

func pingWithRetry(ctx context.Context, pool *pgxpool.Pool, attempts int, delay time.Duration, logger *zap.Logger) error {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		if err := pool.Ping(ctx); err != nil {
			lastErr = err
			logger.Warn("postgres: ping attempt failed",
				zap.Int("attempt", i),
				zap.Int("max", attempts),
				zap.Error(err),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("after %d attempts: %w", attempts, lastErr)
}
