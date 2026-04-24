package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"

	"github.com/yourorg/microservice-platform/backend/internal/config"
	"github.com/yourorg/microservice-platform/backend/internal/domain/user"
	"github.com/yourorg/microservice-platform/backend/internal/service"
)

// ── Mock Repository ───────────────────────────────────────────

type mockUserRepo struct {
	users map[uuid.UUID]*user.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*user.User)}
}

func (m *mockUserRepo) Create(_ context.Context, u *user.User) error {
	for _, existing := range m.users {
		if existing.Email == u.Email {
			return user.ErrAlreadyExists
		}
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*user.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (m *mockUserRepo) Update(_ context.Context, u *user.User) error {
	if _, ok := m.users[u.ID]; !ok {
		return user.ErrNotFound
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.users[id]; !ok {
		return user.ErrNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) List(_ context.Context, _ user.ListFilter) ([]*user.User, int64, error) {
	var users []*user.User
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, int64(len(users)), nil
}

func (m *mockUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, u := range m.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

// ── Test Helpers ──────────────────────────────────────────────

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-secret-32-characters-long-!!",
			RefreshSecret: "test-refresh-32-characters-long!!",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "test",
		},
	}
}

func newTestSvc(t *testing.T) (*service.UserService, *mockUserRepo) {
	t.Helper()
	repo := newMockUserRepo()
	logger := zaptest.NewLogger(t)
	svc := service.NewUserService(repo, testConfig(), logger)
	return svc, repo
}

// ── Tests ─────────────────────────────────────────────────────

func TestUserService_Create_Success(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	u, err := svc.Create(ctx, user.CreateCommand{
		Email:     "alice@example.com",
		Password:  "SecurePass1!",
		FirstName: "Alice",
		LastName:  "Smith",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", u.Email)
	}
	if u.PasswordHash == "SecurePass1!" {
		t.Error("password must be hashed, not stored as plain text")
	}
	if !u.IsActive {
		t.Error("new user should be active by default")
	}
}

func TestUserService_Create_DuplicateEmail(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	cmd := user.CreateCommand{
		Email:     "dup@example.com",
		Password:  "Pass1234!",
		FirstName: "Bob",
		LastName:  "Jones",
	}

	if _, err := svc.Create(ctx, cmd); err != nil {
		t.Fatalf("first create failed unexpectedly: %v", err)
	}
	_, err := svc.Create(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestUserService_Login_Success(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, user.CreateCommand{
		Email:     "login@example.com",
		Password:  "LoginPass1!",
		FirstName: "Carol",
		LastName:  "White",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	u, access, refresh, err := svc.Login(ctx, "login@example.com", "LoginPass1!")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if u.Email != "login@example.com" {
		t.Errorf("unexpected user email: %s", u.Email)
	}
	if access == "" {
		t.Error("access token should not be empty")
	}
	if refresh == "" {
		t.Error("refresh token should not be empty")
	}
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, user.CreateCommand{
		Email:    "pw@example.com",
		Password: "RightPass1!",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, _, _, err = svc.Login(ctx, "pw@example.com", "WrongPass!")
	if err != user.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestUserService_Update_Success(t *testing.T) {
	svc, _ := newTestSvc(t)
	ctx := context.Background()

	u, _ := svc.Create(ctx, user.CreateCommand{
		Email:     "update@example.com",
		Password:  "Pass1234!",
		FirstName: "Old",
		LastName:  "Name",
	})

	updated, err := svc.Update(ctx, user.UpdateCommand{
		ID:        u.ID,
		FirstName: "New",
		LastName:  "Name",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.FirstName != "New" {
		t.Errorf("expected first name 'New', got %s", updated.FirstName)
	}
}

func TestUserService_Delete_Success(t *testing.T) {
	svc, repo := newTestSvc(t)
	ctx := context.Background()

	u, _ := svc.Create(ctx, user.CreateCommand{
		Email:    "del@example.com",
		Password: "Pass1234!",
	})

	if err := svc.Delete(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, exists := repo.users[u.ID]; exists {
		t.Error("user should have been removed from repository")
	}
}
