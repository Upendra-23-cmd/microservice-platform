// Package messaging provides Redis-backed caching and pub/sub event bus.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yourorg/microservice-platform/backend/internal/config"
)

// ============================================================
// Client Factory
// ============================================================

// NewRedisClient creates a validated Redis client from config.
func NewRedisClient(ctx context.Context, cfg config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	if err := pingWithRetry(ctx, rdb, 5, 2*time.Second, logger); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}

	logger.Info("redis: connected", zap.String("addr", cfg.Addr()))
	return rdb, nil
}

func pingWithRetry(ctx context.Context, rdb *redis.Client, attempts int, delay time.Duration, logger *zap.Logger) error {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		if err := rdb.Ping(ctx).Err(); err != nil {
			lastErr = err
			logger.Warn("redis: ping attempt failed",
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

// ============================================================
// Cache — typed generic wrapper around Redis
// ============================================================

// Cache is a generic typed cache backed by Redis.
type Cache[T any] struct {
	rdb        *redis.Client
	defaultTTL time.Duration
	logger     *zap.Logger
}

// NewCache creates a new generic Cache.
func NewCache[T any](rdb *redis.Client, defaultTTL time.Duration, logger *zap.Logger) *Cache[T] {
	return &Cache[T]{rdb: rdb, defaultTTL: defaultTTL, logger: logger}
}

// Set serialises the value to JSON and stores it with the given TTL.
// Passing ttl=0 uses the default TTL configured at construction.
func (c *Cache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache.Set marshal: %w", err)
	}
	if err := c.rdb.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("cache.Set redis: %w", err)
	}
	return nil
}

// Get retrieves and deserialises a cached value.
// Returns (zero, false, nil) when the key does not exist.
func (c *Cache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var zero T
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("cache.Get redis: %w", err)
	}
	var value T
	if err := json.Unmarshal(b, &value); err != nil {
		return zero, false, fmt.Errorf("cache.Get unmarshal: %w", err)
	}
	return value, true, nil
}

// Delete removes a key from the cache.
func (c *Cache[T]) Delete(ctx context.Context, key string) error {
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache.Delete: %w", err)
	}
	return nil
}

// Invalidate deletes all keys matching a pattern (e.g. "user:*").
// Use with care: SCAN-based, non-atomic, but safe for production read caches.
func (c *Cache[T]) Invalidate(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("cache.Invalidate scan: %w", err)
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				c.logger.Warn("cache.Invalidate partial failure", zap.Error(err))
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return nil
}

// ============================================================
// Event Bus — Redis Pub/Sub
// ============================================================

// EventBus publishes domain events via Redis pub/sub channels.
type EventBus struct {
	rdb    *redis.Client
	logger *zap.Logger
}

// DomainEvent is the envelope for all published events.
type DomainEvent struct {
	Type      string          `json:"type"`
	AggID     string          `json:"agg_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func NewEventBus(rdb *redis.Client, logger *zap.Logger) *EventBus {
	return &EventBus{rdb: rdb, logger: logger.Named("event-bus")}
}

// Publish serialises payload and publishes to channel.
func (b *EventBus) Publish(ctx context.Context, channel, eventType, aggID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("EventBus.Publish marshal payload: %w", err)
	}
	event := DomainEvent{
		Type:      eventType,
		AggID:     aggID,
		Timestamp: time.Now().UTC(),
		Payload:   raw,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("EventBus.Publish marshal event: %w", err)
	}
	if err := b.rdb.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("EventBus.Publish redis: %w", err)
	}
	b.logger.Debug("event published",
		zap.String("channel", channel),
		zap.String("type", eventType),
		zap.String("agg_id", aggID),
	)
	return nil
}

// Subscribe listens on a channel and calls handler for each message.
// Blocks until ctx is cancelled.
func (b *EventBus) Subscribe(ctx context.Context, channel string, handler func(event DomainEvent) error) error {
	sub := b.rdb.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()
	b.logger.Info("event-bus: subscribed", zap.String("channel", channel))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("EventBus.Subscribe: channel closed")
			}
			var event DomainEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				b.logger.Error("event-bus: unmarshal error", zap.Error(err), zap.String("raw", msg.Payload))
				continue
			}
			if err := handler(event); err != nil {
				b.logger.Error("event-bus: handler error",
					zap.String("channel", channel),
					zap.String("type", event.Type),
					zap.Error(err),
				)
			}
		}
	}
}
