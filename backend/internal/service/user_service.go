// Package service contains the application-layer business logic.
// Services orchestrate domain objects and call repository ports.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourorg/microservice-platform/backend/internal/config"
	"github.com/yourorg/microservice-platform/backend/internal/domain/user"
	pkgerrors "github.com/yourorg/microservice-platform/backend/pkg/errors"
)

const bcryptCost = 12

// UserService implements user.Service.
type UserService struct {
	repo   user.Repository
	cfg    *config.Config
	logger *zap.Logger
}

// NewUserService constructs a UserService with all dependencies injected.
func NewUserService(repo user.Repository, cfg *config.Config, logger *zap.Logger) *UserService {
	return &UserService{
		repo:   repo,
		cfg:    cfg,
		logger: logger.Named("user-service"),
	}
}

// ============================================================
// Write Operations
// ============================================================

func (s *UserService) Create(ctx context.Context, cmd user.CreateCommand) (*user.User, error) {
	// Check uniqueness
	exists, err := s.repo.ExistsByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, pkgerrors.Internal("check email uniqueness", err)
	}
	if exists {
		return nil, user.ErrAlreadyExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcryptCost)
	if err != nil {
		return nil, pkgerrors.Internal("hash password", err)
	}

	role := cmd.Role
	if !role.IsValid() {
		role = user.RoleMember
	}

	u := &user.User{
		ID:           uuid.New(),
		Email:        cmd.Email,
		PasswordHash: string(hash),
		FirstName:    cmd.FirstName,
		LastName:     cmd.LastName,
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, pkgerrors.Internal("create user", err)
	}

	s.logger.Info("user created", zap.String("id", u.ID.String()), zap.String("email", u.Email))
	return u, nil
}

func (s *UserService) Update(ctx context.Context, cmd user.UpdateCommand) (*user.User, error) {
	u, err := s.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, s.mapRepoError(err)
	}

	if cmd.FirstName != "" {
		u.FirstName = cmd.FirstName
	}
	if cmd.LastName != "" {
		u.LastName = cmd.LastName
	}
	u.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, pkgerrors.Internal("update user", err)
	}
	return u, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return s.mapRepoError(err)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return pkgerrors.Internal("delete user", err)
	}
	s.logger.Info("user deleted", zap.String("id", id.String()))
	return nil
}

// ============================================================
// Read Operations
// ============================================================

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, s.mapRepoError(err)
	}
	return u, nil
}

func (s *UserService) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	users, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, pkgerrors.Internal("list users", err)
	}
	return users, total, nil
}

// ============================================================
// Auth Operations
// ============================================================

func (s *UserService) Login(ctx context.Context, email, password string) (*user.User, string, string, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// Return generic error to prevent email enumeration
		return nil, "", "", user.ErrInvalidCredentials
	}

	if !u.IsActive {
		return nil, "", "", user.ErrInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", user.ErrInvalidCredentials
	}

	accessToken, err := s.generateToken(u, s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessExpiry, "access")
	if err != nil {
		return nil, "", "", pkgerrors.Internal("generate access token", err)
	}

	refreshToken, err := s.generateToken(u, s.cfg.JWT.RefreshSecret, s.cfg.JWT.RefreshExpiry, "refresh")
	if err != nil {
		return nil, "", "", pkgerrors.Internal("generate refresh token", err)
	}

	u.RecordLogin()
	_ = s.repo.Update(ctx, u) // Non-fatal update

	s.logger.Info("user logged in", zap.String("id", u.ID.String()))
	return u, accessToken, refreshToken, nil
}

func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := s.parseToken(refreshToken, s.cfg.JWT.RefreshSecret)
	if err != nil {
		return "", pkgerrors.Unauthenticated("invalid refresh token")
	}

	if claims["type"] != "refresh" {
		return "", pkgerrors.Unauthenticated("token type mismatch")
	}

	userID, err := uuid.Parse(fmt.Sprintf("%v", claims["sub"]))
	if err != nil {
		return "", pkgerrors.Unauthenticated("invalid subject claim")
	}

	u, err := s.repo.GetByID(ctx, userID)
	if err != nil || !u.IsActive {
		return "", pkgerrors.Unauthenticated("user not found or inactive")
	}

	return s.generateToken(u, s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessExpiry, "access")
}

// ============================================================
// Private Helpers
// ============================================================

func (s *UserService) generateToken(u *user.User, secret string, expiry time.Duration, tokenType string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":   u.ID.String(),
		"email": u.Email,
		"role":  string(u.Role),
		"type":  tokenType,
		"iss":   s.cfg.JWT.Issuer,
		"iat":   now.Unix(),
		"exp":   now.Add(expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *UserService) parseToken(tokenStr, secret string) (jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

func (s *UserService) mapRepoError(err error) error {
	if err == user.ErrNotFound {
		return pkgerrors.NotFound("user")
	}
	return pkgerrors.Internal("repository", err)
}
