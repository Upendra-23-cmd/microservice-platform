// Package product contains the Product aggregate root.
// Product core data lives in PostgreSQL; rich metadata & tags live in MongoDB.
package product

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("product: not found")
	ErrAlreadyExists = errors.New("product: already exists")
	ErrOutOfStock    = errors.New("product: out of stock")
	ErrInvalidPrice  = errors.New("product: price must be greater than zero")
	ErrInvalidStock  = errors.New("product: stock cannot be negative")
)

// ============================================================
// Value Objects
// ============================================================

type StockOperation string

const (
	StockOpIncrement StockOperation = "increment"
	StockOpDecrement StockOperation = "decrement"
	StockOpSet       StockOperation = "set"
)

// Price is a value object ensuring invariants.
type Price struct {
	Amount   float64
	Currency string
}

func NewPrice(amount float64, currency string) (Price, error) {
	if amount <= 0 {
		return Price{}, ErrInvalidPrice
	}
	if currency == "" {
		currency = "USD"
	}
	return Price{Amount: amount, Currency: currency}, nil
}

// ============================================================
// Aggregate Root
// ============================================================

// Product core fields (stored in PostgreSQL for relational integrity)
type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       float64
	Stock       int32
	Category    string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Rich data stored in MongoDB (joined at service layer)
	Metadata *ProductMetadata
}

// ProductMetadata contains flexible, schema-less attributes stored in MongoDB.
type ProductMetadata struct {
	ProductID   string            `bson:"product_id"`
	Tags        []string          `bson:"tags"`
	Attributes  map[string]string `bson:"attributes"`
	Images      []string          `bson:"images"`
	SEO         SEOData           `bson:"seo"`
	UpdatedAt   time.Time         `bson:"updated_at"`
}

type SEOData struct {
	Slug        string   `bson:"slug"`
	MetaTitle   string   `bson:"meta_title"`
	MetaDesc    string   `bson:"meta_description"`
	Keywords    []string `bson:"keywords"`
}

// ApplyStockOperation mutates stock based on operation type.
func (p *Product) ApplyStockOperation(quantity int32, op StockOperation) error {
	switch op {
	case StockOpIncrement:
		p.Stock += quantity
	case StockOpDecrement:
		if p.Stock < quantity {
			return ErrOutOfStock
		}
		p.Stock -= quantity
	case StockOpSet:
		if quantity < 0 {
			return ErrInvalidStock
		}
		p.Stock = quantity
	}
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// IsAvailable returns true if the product can be purchased.
func (p *Product) IsAvailable(requested int32) bool {
	return p.IsActive && p.Stock >= requested
}

// ============================================================
// Repository Interfaces
// ============================================================

// Repository handles core product data in PostgreSQL.
type Repository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ListFilter) ([]*Product, int64, error)
	UpdateStock(ctx context.Context, id uuid.UUID, quantity int32, op StockOperation) (int32, error)
}

// MetadataRepository handles rich product data in MongoDB.
type MetadataRepository interface {
	Upsert(ctx context.Context, meta *ProductMetadata) error
	GetByProductID(ctx context.Context, productID string) (*ProductMetadata, error)
	Delete(ctx context.Context, productID string) error
}

type ListFilter struct {
	Page      int
	PageSize  int
	Category  string
	Search    string
	MinPrice  float64
	MaxPrice  float64
	SortBy    string
	Order     string
	IsActive  *bool
}

// ============================================================
// Service Interface
// ============================================================

type Service interface {
	Create(ctx context.Context, cmd CreateCommand) (*Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Update(ctx context.Context, cmd UpdateCommand) (*Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ListFilter) ([]*Product, int64, error)
	UpdateStock(ctx context.Context, id uuid.UUID, quantity int32, op StockOperation) (int32, error)
}

// ============================================================
// Commands
// ============================================================

type CreateCommand struct {
	Name        string
	Description string
	Price       float64
	Stock       int32
	Category    string
	Tags        []string
	Metadata    map[string]string
}

type UpdateCommand struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       float64
	Stock       int32
	Category    string
	Tags        []string
	Metadata    map[string]string
}
