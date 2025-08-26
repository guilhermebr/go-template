package auth

import (
	"context"
	"errors"
	"go-template/domain"
	"go-template/domain/entities"
	"go-template/internal/jwt"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

func newJWT() jwt.Service {
	return jwt.NewService("secret", "test", "1h")
}

// Simple mock for Repository
type mockRepository struct {
	getByEmailFunc func(ctx context.Context, email string) (entities.User, error)
	createFunc     func(ctx context.Context, user entities.User) error
}

func (m *mockRepository) GetByEmail(ctx context.Context, email string) (entities.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return entities.User{}, nil
}

func (m *mockRepository) Create(ctx context.Context, user entities.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (entities.User, error) {
	return entities.User{}, nil
}

func (m *mockRepository) Update(ctx context.Context, user entities.User) error {
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockRepository) ListUsers(ctx context.Context, params entities.ListUsersParams) ([]entities.User, error) {
	return nil, nil
}

func (m *mockRepository) CountUsers(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockRepository) GetUserStats(ctx context.Context) (entities.UserStats, error) {
	return entities.UserStats{}, nil
}

func (m *mockRepository) GetByAuthProviderID(ctx context.Context, provider, providerID string) (entities.User, error) {
	return entities.User{}, nil
}

// Simple mock for Provider
type mockProvider struct {
	loginFunc    func(ctx context.Context, email, password string) (string, error)
	providerFunc func() string
}

func (m *mockProvider) RegisterUser(ctx context.Context, email, password string) (string, error) {
	return "", nil
}

func (m *mockProvider) Login(ctx context.Context, email, password string) (string, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return "prov-123", nil
}

func (m *mockProvider) Provider() string {
	if m.providerFunc != nil {
		return m.providerFunc()
	}
	return "supabase"
}

func (m *mockProvider) ValidateToken(ctx context.Context, token string) (*entities.User, error) {
	return nil, nil
}

func (m *mockProvider) DeleteUser(ctx context.Context, authProviderID string) error {
	return nil
}

func TestUseCase_Login_Success_UserExists(t *testing.T) {
	existingUser := entities.User{
		ID:             uuid.Must(uuid.NewV4()),
		Email:          "a@b.com",
		AuthProvider:   "supabase",
		AuthProviderID: "prov-123",
		AccountType:    entities.AccountTypeUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo := &mockRepository{
		getByEmailFunc: func(ctx context.Context, email string) (entities.User, error) {
			return existingUser, nil
		},
	}
	provider := &mockProvider{
		loginFunc:    func(ctx context.Context, email, password string) (string, error) { return "prov-123", nil },
		providerFunc: func() string { return "supabase" },
	}
	uc := NewUseCase(repo, provider, newJWT())

	resp, err := uc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "123456"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("expected token, got empty")
	}
	if resp.User.Email != "a@b.com" || resp.User.AuthProvider != "supabase" {
		t.Fatalf("unexpected user payload: %+v", resp.User)
	}
}

func TestUseCase_Login_Success_UserCreatedWhenMissing(t *testing.T) {
	repo := &mockRepository{
		getByEmailFunc: func(ctx context.Context, email string) (entities.User, error) {
			return entities.User{}, domain.ErrNotFound
		},
		createFunc: func(ctx context.Context, user entities.User) error { return nil },
	}
	provider := &mockProvider{
		loginFunc:    func(ctx context.Context, email, password string) (string, error) { return "prov-123", nil },
		providerFunc: func() string { return "supabase" },
	}
	uc := NewUseCase(repo, provider, newJWT())

	resp, err := uc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "123456"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("expected token, got empty")
	}
	if resp.User.Email != "a@b.com" || resp.User.AuthProvider != "supabase" || resp.User.AuthProviderID != "prov-123" {
		t.Fatalf("unexpected user payload: %+v", resp.User)
	}
}

func TestUseCase_Login_AuthError(t *testing.T) {
	repo := &mockRepository{}
	provider := &mockProvider{
		loginFunc: func(ctx context.Context, email, password string) (string, error) {
			return "", errors.New("auth failed")
		},
	}
	uc := NewUseCase(repo, provider, newJWT())

	_, err := uc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "123456"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}