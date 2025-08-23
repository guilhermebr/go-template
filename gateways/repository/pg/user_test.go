package pg

import (
	"context"
	"database/sql"
	"go-template/domain/entities"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	// Create
	user := entities.User{
		ID:             uuid.Must(uuid.NewV4()),
		Email:          "john.doe@example.com",
		AuthProvider:   "supabase",
		AuthProviderID: "prov-123",
		AccountType:    entities.AccountTypeUser,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, user))

	// GetByID
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, got.Email)

	// GetByEmail
	got2, err := repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, user.ID, got2.ID)

	// GetByAuthProviderID
	got3, err := repo.GetByAuthProviderID(ctx, user.AuthProvider, user.AuthProviderID)
	require.NoError(t, err)
	require.Equal(t, user.Email, got3.Email)

	// Update
	got2.Email = "johnny.doe@example.com"
	require.NoError(t, repo.Update(ctx, got2))
	got4, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "johnny.doe@example.com", got4.Email)

	// Duplicate email should error with duplicate key
	user2 := entities.User{
		ID:             uuid.Must(uuid.NewV4()),
		Email:          "johnny.doe@example.com",
		AuthProvider:   "supabase",
		AuthProviderID: "prov-456",
		AccountType:    entities.AccountTypeUser,
	}
	err = repo.Create(ctx, user2)
	require.Error(t, err)

	// Delete
	require.NoError(t, repo.Delete(ctx, user.ID))
	_, err = repo.GetByID(ctx, user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
