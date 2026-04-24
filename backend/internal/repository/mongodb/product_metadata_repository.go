// Package mongodb contains MongoDB implementations of domain repository interfaces.
// MongoDB stores rich, schema-flexible data (product metadata, audit logs, events).
package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/yourorg/microservice-platform/backend/internal/config"
	"github.com/yourorg/microservice-platform/backend/internal/domain/product"
)

// ============================================================
// Client Factory
// ============================================================

// NewClient creates a validated MongoDB client from application config.
func NewClient(ctx context.Context, cfg config.MongoDBConfig, logger *zap.Logger) (*mongo.Client, error) {
	opts := options.Client().
		ApplyURI(cfg.URI).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetServerSelectionTimeout(cfg.ServerSelectionTimeout).
		SetConnectTimeout(cfg.ConnectTimeout)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}

	if err := pingWithRetry(ctx, client, 5, 2*time.Second, logger); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongodb: ping failed: %w", err)
	}

	logger.Info("mongodb: connected", zap.String("db", cfg.DB))
	return client, nil
}

func pingWithRetry(ctx context.Context, client *mongo.Client, attempts int, delay time.Duration, logger *zap.Logger) error {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		if err := client.Ping(ctx, nil); err != nil {
			lastErr = err
			logger.Warn("mongodb: ping attempt failed",
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
// Product Metadata Repository
// ============================================================

const productMetaCollection = "product_metadata"

// ProductMetadataRepository implements product.MetadataRepository using MongoDB.
type ProductMetadataRepository struct {
	col    *mongo.Collection
	logger *zap.Logger
}

func NewProductMetadataRepository(client *mongo.Client, dbName string, logger *zap.Logger) *ProductMetadataRepository {
	col := client.Database(dbName).Collection(productMetaCollection)
	return &ProductMetadataRepository{
		col:    col,
		logger: logger.Named("product-meta-repo-mongo"),
	}
}

// EnsureIndexes creates necessary indexes. Call once at startup.
func (r *ProductMetadataRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "product_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("product_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "tags", Value: 1}},
			Options: options.Index().SetName("tags_idx"),
		},
		{
			Keys:    bson.D{{Key: "seo.slug", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("seo_slug_idx"),
		},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("ProductMetadataRepository.EnsureIndexes: %w", err)
	}
	r.logger.Info("mongodb: product_metadata indexes ensured")
	return nil
}

// Upsert inserts or updates metadata for a product.
func (r *ProductMetadataRepository) Upsert(ctx context.Context, meta *product.ProductMetadata) error {
	meta.UpdatedAt = time.Now().UTC()

	filter := bson.M{"product_id": meta.ProductID}
	update := bson.M{"$set": meta}
	opts := options.Update().SetUpsert(true)

	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("ProductMetadataRepository.Upsert: %w", err)
	}
	return nil
}

// GetByProductID retrieves metadata for a specific product.
func (r *ProductMetadataRepository) GetByProductID(ctx context.Context, productID string) (*product.ProductMetadata, error) {
	filter := bson.M{"product_id": productID}

	var meta product.ProductMetadata
	if err := r.col.FindOne(ctx, filter).Decode(&meta); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Non-fatal: metadata is optional
		}
		return nil, fmt.Errorf("ProductMetadataRepository.GetByProductID: %w", err)
	}
	return &meta, nil
}

// Delete removes metadata for a product (cascade from product deletion).
func (r *ProductMetadataRepository) Delete(ctx context.Context, productID string) error {
	filter := bson.M{"product_id": productID}
	_, err := r.col.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("ProductMetadataRepository.Delete: %w", err)
	}
	return nil
}

// SearchByTags finds product IDs matching all given tags.
func (r *ProductMetadataRepository) SearchByTags(ctx context.Context, tags []string, limit int) ([]string, error) {
	filter := bson.M{"tags": bson.M{"$all": tags}}
	opts := options.Find().
		SetProjection(bson.M{"product_id": 1}).
		SetLimit(int64(limit))

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("ProductMetadataRepository.SearchByTags: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		ProductID string `bson:"product_id"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("ProductMetadataRepository.SearchByTags decode: %w", err)
	}

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ProductID
	}
	return ids, nil
}
