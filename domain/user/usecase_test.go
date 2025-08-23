package user

import (
	"context"
	"errors"
	"go-template/domain"
	mauth "go-template/domain/auth/mocks"
	"go-template/domain/entities"
	muser "go-template/domain/user/mocks"
	"go-template/internal/jwt"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

func newJWT() jwt.Service {
	return jwt.NewService("secret", "test", "1h")
}

func TestUseCase_Register_Success(t *testing.T) {
	repo := &muser.RepositoryMock{
		CreateFunc: func(ctx context.Context, user entities.User) error { return nil },
	}
	provider := &mauth.ProviderMock{
		RegisterUserFunc: func(ctx context.Context, email string, password string) (string, error) { return "prov-123", nil },
		ProviderFunc:     func() string { return "supabase" },
	}
	uc := NewUseCase(repo, provider, newJWT())

	resp, err := uc.Register(context.Background(), RegisterRequest{Email: "a@b.com", Password: "123456"})
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

func TestUseCase_Register_AuthProviderError(t *testing.T) {
	repo := &muser.RepositoryMock{}
	provider := &mauth.ProviderMock{
		RegisterUserFunc: func(ctx context.Context, email string, password string) (string, error) {
			return "", errors.New("fail")
		},
	}
	uc := NewUseCase(repo, provider, newJWT())

	_, err := uc.Register(context.Background(), RegisterRequest{Email: "a@b.com", Password: "123456"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUseCase_Register_DBError(t *testing.T) {
	repo := &muser.RepositoryMock{
		CreateFunc: func(ctx context.Context, user entities.User) error { return errors.New("db") },
	}
	provider := &mauth.ProviderMock{
		RegisterUserFunc: func(ctx context.Context, email string, password string) (string, error) { return "prov-123", nil },
		ProviderFunc:     func() string { return "supabase" },
	}
	uc := NewUseCase(repo, provider, newJWT())

	_, err := uc.Register(context.Background(), RegisterRequest{Email: "a@b.com", Password: "123456"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUseCase_Login_Success_UserExists(t *testing.T) {
	now := time.Now()
	existing := entities.User{ID: uuid.Must(uuid.NewV4()), Email: "a@b.com", AuthProvider: "supabase", AuthProviderID: "id", CreatedAt: now, UpdatedAt: now}
	repo := &muser.RepositoryMock{
		GetByEmailFunc: func(ctx context.Context, email string) (entities.User, error) { return existing, nil },
	}
	provider := &mauth.ProviderMock{
		LoginFunc:    func(ctx context.Context, email string, password string) (string, error) { return "prov-123", nil },
		ProviderFunc: func() string { return "supabase" },
	}
	uc := NewUseCase(repo, provider, newJWT())

	resp, err := uc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.User.Email != existing.Email {
		t.Fatalf("expected existing user email, got %s", resp.User.Email)
	}
}

func TestUseCase_Login_Success_UserCreatedWhenMissing(t *testing.T) {
	var created entities.User
	repo := &muser.RepositoryMock{
		GetByEmailFunc: func(ctx context.Context, email string) (entities.User, error) {
			return entities.User{}, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, user entities.User) error { created = user; return nil },
	}
	provider := &mauth.ProviderMock{
		LoginFunc:    func(ctx context.Context, email string, password string) (string, error) { return "prov-123", nil },
		ProviderFunc: func() string { return "supabase" },
	}
	uc := NewUseCase(repo, provider, newJWT())

	resp, err := uc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Email != "a@b.com" || created.AuthProviderID != "prov-123" {
		t.Fatalf("user not created as expected: %+v", created)
	}
	if resp.Token == "" {
		t.Fatalf("expected token, got empty")
	}
}

func TestUseCase_Login_AuthError(t *testing.T) {
	repo := &muser.RepositoryMock{}
	provider := &mauth.ProviderMock{
		LoginFunc: func(ctx context.Context, email string, password string) (string, error) {
			return "", errors.New("auth")
		},
	}
	uc := NewUseCase(repo, provider, newJWT())

	_, err := uc.Login(context.Background(), LoginRequest{Email: "a@b.com", Password: "x"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUseCase_GetUserByID(t *testing.T) {
	u := entities.User{ID: uuid.Must(uuid.NewV4())}
	repo := &muser.RepositoryMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) { return u, nil },
	}
	uc := NewUseCase(repo, &mauth.ProviderMock{}, newJWT())

	got, err := uc.GetUserByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("expected id %s, got %s", u.ID, got.ID)
	}
}
