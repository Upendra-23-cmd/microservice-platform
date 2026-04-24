package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/yourorg/microservice-platform/backend/internal/domain/product"
)

// ProductRepository implements product.Repository using PostgreSQL.
type ProductRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewProductRepository(pool *pgxpool.Pool, logger *zap.Logger) *ProductRepository {
	return &ProductRepository{pool: pool, logger: logger.Named("product-repo-postgres")}
}

func (r *ProductRepository) Create(ctx context.Context, p *product.Product) error {
	const q = `
		INSERT INTO products (id, name, description, price, stock, category, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, q,
		p.ID, p.Name, p.Description, p.Price,
		p.Stock, p.Category, p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("ProductRepository.Create: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	const q = `
		SELECT id, name, description, price, stock, category, is_active, created_at, updated_at
		FROM products
		WHERE id = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, q, id)
	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, product.ErrNotFound
	}
	return p, err
}

func (r *ProductRepository) Update(ctx context.Context, p *product.Product) error {
	const q = `
		UPDATE products
		SET name = $2, description = $3, price = $4, stock = $5, category = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, p.ID, p.Name, p.Description, p.Price, p.Stock, p.Category, p.IsActive)
	if err != nil {
		return fmt.Errorf("ProductRepository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return product.ErrNotFound
	}
	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE products SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("ProductRepository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return product.ErrNotFound
	}
	return nil
}

func (r *ProductRepository) List(ctx context.Context, f product.ListFilter) ([]*product.Product, int64, error) {
	conds := []string{"deleted_at IS NULL"}
	args  := []any{}
	idx   := 1

	if f.Search != "" {
		conds = append(conds, fmt.Sprintf(
			"to_tsvector('english', name || ' ' || COALESCE(description,'')) @@ plainto_tsquery($%d)", idx,
		))
		args = append(args, f.Search)
		idx++
	}
	if f.Category != "" {
		conds = append(conds, fmt.Sprintf("category = $%d", idx))
		args = append(args, f.Category)
		idx++
	}
	if f.MinPrice > 0 {
		conds = append(conds, fmt.Sprintf("price >= $%d", idx))
		args = append(args, f.MinPrice)
		idx++
	}
	if f.MaxPrice > 0 {
		conds = append(conds, fmt.Sprintf("price <= $%d", idx))
		args = append(args, f.MaxPrice)
		idx++
	}
	if f.IsActive != nil {
		conds = append(conds, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *f.IsActive)
		idx++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM products "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ProductRepository.List count: %w", err)
	}

	sortBy := sanitizeSortColumn(f.SortBy, []string{"name", "price", "stock", "created_at", "category"}, "created_at")
	order  := "DESC"
	if strings.ToUpper(f.Order) == "ASC" {
		order = "ASC"
	}
	offset := (f.Page - 1) * f.PageSize
	args   = append(args, f.PageSize, offset)

	q := fmt.Sprintf(`
		SELECT id, name, description, price, stock, category, is_active, created_at, updated_at
		FROM products %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, sortBy, order, idx, idx+1,
	)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ProductRepository.List query: %w", err)
	}
	defer rows.Close()

	var products []*product.Product
	for rows.Next() {
		p, err := scanProductRows(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *ProductRepository) UpdateStock(ctx context.Context, id uuid.UUID, quantity int32, op product.StockOperation) (int32, error) {
	var q string
	switch op {
	case product.StockOpIncrement:
		q = `UPDATE products SET stock = stock + $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING stock`
	case product.StockOpDecrement:
		q = `UPDATE products SET stock = stock - $2, updated_at = NOW() WHERE id = $1 AND stock >= $2 AND deleted_at IS NULL RETURNING stock`
	case product.StockOpSet:
		q = `UPDATE products SET stock = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING stock`
	default:
		return 0, fmt.Errorf("unknown stock operation: %s", op)
	}

	var newStock int32
	err := r.pool.QueryRow(ctx, q, id, quantity).Scan(&newStock)
	if errors.Is(err, pgx.ErrNoRows) {
		if op == product.StockOpDecrement {
			return 0, product.ErrOutOfStock
		}
		return 0, product.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("ProductRepository.UpdateStock: %w", err)
	}
	return newStock, nil
}

// ── helpers ───────────────────────────────────────────────────

func scanProduct(row pgx.Row) (*product.Product, error) {
	var p product.Product
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Category, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanProduct: %w", err)
	}
	return &p, nil
}

func scanProductRows(rows pgx.Rows) (*product.Product, error) {
	var p product.Product
	err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Category, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanProductRows: %w", err)
	}
	return &p, nil
}
