package user

import (
	"context"
	"go-template/domain/auth"
	"go-template/domain/entities"
	muser "go-template/domain/user/mocks"
	"testing"

	"github.com/gofrs/uuid/v5"
)

// Simple mock auth factory for testing
type mockAuthFactory struct{}

func (m *mockAuthFactory) CreateProvider(providerName string) (auth.Provider, error) {
	return nil, nil // Not used in this test
}

func (m *mockAuthFactory) GetSupportedProviders() []string {
	return []string{"supabase"}
}

func TestUseCase_GetUserByID(t *testing.T) {
	u := entities.User{ID: uuid.Must(uuid.NewV4())}
	repo := &muser.RepositoryMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) { return u, nil },
	}
	authFactory := &mockAuthFactory{}
	uc := NewUseCase(repo, authFactory, "supabase")

	got, err := uc.GetUserByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("expected id %s, got %s", u.ID, got.ID)
	}
}
