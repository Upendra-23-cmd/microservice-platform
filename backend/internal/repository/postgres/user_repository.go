// Package postgres contains PostgreSQL implementations of domain repository interfaces.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/yourorg/microservice-platform/backend/internal/domain/user"
)

// UserRepository implements user.Repository using PostgreSQL via pgx/v5.
type UserRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewUserRepository(pool *pgxpool.Pool, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		pool:   pool,
		logger: logger.Named("user-repo-postgres"),
	}
}

// ============================================================
// Write Operations
// ============================================================

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, q,
		u.ID, u.Email, u.PasswordHash,
		u.FirstName, u.LastName, string(u.Role),
		u.IsActive, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return user.ErrAlreadyExists
		}
		return fmt.Errorf("UserRepository.Create: %w", err)
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	const q = `
		UPDATE users
		SET first_name = $2, last_name = $3, role = $4, is_active = $5,
		    last_login_at = $6, updated_at = $7
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		u.ID, u.FirstName, u.LastName, string(u.Role),
		u.IsActive, u.LastLoginAt, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("UserRepository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM users WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("UserRepository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

// ============================================================
// Read Operations
// ============================================================

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	const q = `
		SELECT id, email, password_hash, first_name, last_name, role,
		       is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, q, id)
	return scanUser(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	const q = `
		SELECT id, email, password_hash, first_name, last_name, role,
		       is_active, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	row := r.pool.QueryRow(ctx, q, email)
	return scanUser(row)
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("UserRepository.ExistsByEmail: %w", err)
	}
	return exists, nil
}

func (r *UserRepository) List(ctx context.Context, f user.ListFilter) ([]*user.User, int64, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	argIdx := 1

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}
	if f.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *f.IsActive)
		argIdx++
	}
	if f.Role != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, string(*f.Role))
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List count: %w", err)
	}

	// Sanitize sort
	sortBy := sanitizeSortColumn(f.SortBy, []string{"created_at", "email", "first_name", "last_name"}, "created_at")
	order := "DESC"
	if strings.ToUpper(f.Order) == "ASC" {
		order = "ASC"
	}

	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)

	listQ := fmt.Sprintf(`
		SELECT id, email, password_hash, first_name, last_name, role,
		       is_active, last_login_at, created_at, updated_at
		FROM users %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, sortBy, order, argIdx, argIdx+1,
	)

	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List query: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u, err := scanUserFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, rows.Err()
}

// ============================================================
// Private helpers
// ============================================================

func scanUser(row pgx.Row) (*user.User, error) {
	var u user.User
	var role string
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash,
		&u.FirstName, &u.LastName, &role,
		&u.IsActive, &u.LastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("scanUser: %w", err)
	}
	u.Role = user.Role(role)
	return &u, nil
}

func scanUserFromRows(rows pgx.Rows) (*user.User, error) {
	var u user.User
	var role string
	err := rows.Scan(
		&u.ID, &u.Email, &u.PasswordHash,
		&u.FirstName, &u.LastName, &role,
		&u.IsActive, &u.LastLoginAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanUserFromRows: %w", err)
	}
	u.Role = user.Role(role)
	return &u, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

func sanitizeSortColumn(col string, allowed []string, fallback string) string {
	for _, a := range allowed {
		if strings.EqualFold(col, a) {
			return a
		}
	}
	return fallback
}
