package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/yourorg/microservice-platform/backend/internal/config"
	"github.com/yourorg/microservice-platform/backend/internal/domain/product"
	pkgerrors "github.com/yourorg/microservice-platform/backend/pkg/errors"
)

// ProductService implements product.Service.
// Core data (price, stock) → PostgreSQL.
// Rich data (tags, metadata) → MongoDB (joined here at the service layer).
type ProductService struct {
	repo     product.Repository
	metaRepo product.MetadataRepository
	cfg      *config.Config
	logger   *zap.Logger
}

func NewProductService(
	repo product.Repository,
	metaRepo product.MetadataRepository,
	cfg *config.Config,
	logger *zap.Logger,
) *ProductService {
	return &ProductService{
		repo:     repo,
		metaRepo: metaRepo,
		cfg:      cfg,
		logger:   logger.Named("product-service"),
	}
}

// ── Write Operations ─────────────────────────────────────────

func (s *ProductService) Create(ctx context.Context, cmd product.CreateCommand) (*product.Product, error) {
	if cmd.Price <= 0 {
		return nil, pkgerrors.Validation("price must be greater than zero")
	}

	p := &product.Product{
		ID:          uuid.New(),
		Name:        cmd.Name,
		Description: cmd.Description,
		Price:       cmd.Price,
		Stock:       cmd.Stock,
		Category:    cmd.Category,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, pkgerrors.Internal("create product", err)
	}

	// Persist rich metadata to MongoDB (non-fatal if this fails)
	if len(cmd.Tags) > 0 || len(cmd.Metadata) > 0 {
		meta := &product.ProductMetadata{
			ProductID:  p.ID.String(),
			Tags:       cmd.Tags,
			Attributes: cmd.Metadata,
			UpdatedAt:  time.Now().UTC(),
		}
		if err := s.metaRepo.Upsert(ctx, meta); err != nil {
			s.logger.Warn("failed to persist product metadata",
				zap.String("product_id", p.ID.String()),
				zap.Error(err),
			)
		} else {
			p.Metadata = meta
		}
	}

	s.logger.Info("product created", zap.String("id", p.ID.String()), zap.String("name", p.Name))
	return p, nil
}

func (s *ProductService) Update(ctx context.Context, cmd product.UpdateCommand) (*product.Product, error) {
	p, err := s.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, s.mapRepoError(err)
	}

	if cmd.Name != ""        { p.Name = cmd.Name }
	if cmd.Description != "" { p.Description = cmd.Description }
	if cmd.Price > 0         { p.Price = cmd.Price }
	if cmd.Stock >= 0        { p.Stock = cmd.Stock }
	if cmd.Category != ""    { p.Category = cmd.Category }
	p.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, pkgerrors.Internal("update product", err)
	}

	// Update MongoDB metadata
	if len(cmd.Tags) > 0 || len(cmd.Metadata) > 0 {
		meta := &product.ProductMetadata{
			ProductID:  p.ID.String(),
			Tags:       cmd.Tags,
			Attributes: cmd.Metadata,
			UpdatedAt:  time.Now().UTC(),
		}
		if err := s.metaRepo.Upsert(ctx, meta); err != nil {
			s.logger.Warn("failed to update product metadata", zap.Error(err))
		}
	}

	return p, nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return s.mapRepoError(err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return pkgerrors.Internal("delete product", err)
	}

	// Cascade delete metadata from MongoDB
	if err := s.metaRepo.Delete(ctx, id.String()); err != nil {
		s.logger.Warn("failed to delete product metadata", zap.String("id", id.String()), zap.Error(err))
	}

	s.logger.Info("product deleted", zap.String("id", id.String()))
	return nil
}

// ── Read Operations ──────────────────────────────────────────

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, s.mapRepoError(err)
	}

	// Enrich with MongoDB metadata (non-fatal)
	meta, err := s.metaRepo.GetByProductID(ctx, id.String())
	if err != nil {
		s.logger.Warn("failed to fetch product metadata", zap.Error(err))
	} else {
		p.Metadata = meta
	}

	return p, nil
}

func (s *ProductService) List(ctx context.Context, filter product.ListFilter) ([]*product.Product, int64, error) {
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	products, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, pkgerrors.Internal("list products", err)
	}
	return products, total, nil
}

func (s *ProductService) UpdateStock(ctx context.Context, id uuid.UUID, quantity int32, op product.StockOperation) (int32, error) {
	newStock, err := s.repo.UpdateStock(ctx, id, quantity, op)
	if err != nil {
		if err == product.ErrOutOfStock {
			return 0, pkgerrors.Validation(fmt.Sprintf("insufficient stock for operation %s with quantity %d", op, quantity))
		}
		return 0, pkgerrors.Internal("update stock", err)
	}
	return newStock, nil
}

func (s *ProductService) mapRepoError(err error) error {
	if err == product.ErrNotFound {
		return pkgerrors.NotFound("product")
	}
	return pkgerrors.Internal("repository", err)
}
