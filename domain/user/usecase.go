package user

import (
	"context"
	"fmt"
	"go-template/domain/auth"
	"go-template/domain/entities"
	"log/slog"
	"time"

	"github.com/gofrs/uuid/v5"
)

type UseCase struct {
	repo           Repository
	authFactory    auth.AuthProviderFactory
	defaultProvider string
}

func NewUseCase(repo Repository, authFactory auth.AuthProviderFactory, defaultProvider string) *UseCase {
	return &UseCase{
		repo:           repo,
		authFactory:    authFactory,
		defaultProvider: defaultProvider,
	}
}

func (uc *UseCase) GetUserByID(ctx context.Context, userID uuid.UUID) (entities.User, error) {
	user, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("failed to get user by ID", "error", err)
		return entities.User{}, err
	}

	return user, nil
}

func (uc *UseCase) GetMe(ctx context.Context, userID uuid.UUID) (entities.User, error) {
	return uc.GetUserByID(ctx, userID)
}

// Admin use cases
func (uc *UseCase) ListUsers(ctx context.Context, page, pageSize int) ([]entities.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := int32((page - 1) * pageSize)
	limit := int32(pageSize)

	users, err := uc.repo.ListUsers(ctx, entities.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		slog.Error("failed to list users", "error", err)
		return nil, 0, err
	}

	total, err := uc.repo.CountUsers(ctx)
	if err != nil {
		slog.Error("failed to count users", "error", err)
		return nil, 0, err
	}

	return users, total, nil
}

func (uc *UseCase) UpdateUser(ctx context.Context, user entities.User) error {
	err := uc.repo.Update(ctx, user)
	if err != nil {
		slog.Error("failed to update user", "error", err)
		return err
	}

	return nil
}

func (uc *UseCase) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// First get the user to obtain auth provider information
	user, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("failed to get user for deletion", "error", err)
		return err
	}

	// Delete from external auth provider if we have provider info
	if user.AuthProvider != "" && user.AuthProviderID != "" {
		provider, err := uc.authFactory.CreateProvider(user.AuthProvider)
		if err != nil {
			slog.Error("failed to create auth provider for deletion", "provider", user.AuthProvider, "error", err)
			// Continue with local deletion even if auth provider fails
		} else {
			if err := provider.DeleteUser(ctx, user.AuthProviderID); err != nil {
				slog.Error("failed to delete user from auth provider", "provider", user.AuthProvider, "auth_provider_id", user.AuthProviderID, "error", err)
				// Continue with local deletion even if auth provider deletion fails
			} else {
				slog.Info("successfully deleted user from auth provider", "provider", user.AuthProvider, "auth_provider_id", user.AuthProviderID)
			}
		}
	}

	// Delete from local database
	err = uc.repo.Delete(ctx, userID)
	if err != nil {
		slog.Error("failed to delete user from local database", "error", err)
		return err
	}

	slog.Info("user deleted successfully", "user_id", userID, "email", user.Email)
	return nil
}

func (uc *UseCase) GetUserStats(ctx context.Context) (entities.UserStats, error) {
	stats, err := uc.repo.GetUserStats(ctx)
	if err != nil {
		slog.Error("failed to get user stats", "error", err)
		return entities.UserStats{}, err
	}

	return stats, nil
}

func (uc *UseCase) CreateUser(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
	// Use default provider if none specified
	if authProvider == "" {
		authProvider = uc.defaultProvider
	}
	
	// Use default account type if none specified (for API registration)
	if accountType == "" {
		accountType = entities.AccountTypeUser
	}

	slog.Info("starting user creation", "email", email, "auth_provider", authProvider, "account_type", accountType)

	// Create auth provider instance
	provider, err := uc.authFactory.CreateProvider(authProvider)
	if err != nil {
		slog.Error("failed to create auth provider", "provider", authProvider, "error", err)
		return entities.User{}, fmt.Errorf("unsupported auth provider %s: %w", authProvider, err)
	}

	// Register with external auth provider
	authProviderID, err := provider.RegisterUser(ctx, email, password)
	if err != nil {
		slog.Error("failed to register with auth provider", "provider", authProvider, "error", err)
		return entities.User{}, fmt.Errorf("failed to register with %s: %w", authProvider, err)
	}

	// Create user with external auth provider ID
	now := time.Now()
	user := entities.User{
		ID:             uuid.Must(uuid.NewV4()),
		Email:          email,
		AuthProvider:   authProvider,
		AuthProviderID: authProviderID,
		AccountType:    accountType,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Store user in local database
	if err := uc.repo.Create(ctx, user); err != nil {
		slog.Error("failed to create user locally after external registration", "error", err, "auth_provider_id", authProviderID)
		// TODO: Consider rollback from external provider if supported
		return entities.User{}, fmt.Errorf("failed to create user locally: %w", err)
	}

	slog.Info("user created successfully", "email", email, "account_type", accountType, "auth_provider", authProvider, "auth_provider_id", authProviderID)
	return user, nil
}

func (uc *UseCase) SearchUsers(ctx context.Context, page, pageSize int, search, accountType string) ([]entities.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// For now, use the same ListUsers method
	// In a real implementation, you would add search functionality to the repository
	users, _, err := uc.ListUsers(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Simple client-side filtering for demo purposes
	// In production, this should be done at the database level
	var filteredUsers []entities.User
	for _, user := range users {
		// Filter by search term (email)
		if search != "" && !contains(user.Email, search) {
			continue
		}

		// Filter by account type
		if accountType != "" && string(user.AccountType) != accountType {
			continue
		}

		filteredUsers = append(filteredUsers, user)
	}

	return filteredUsers, int64(len(filteredUsers)), nil
}

// Helper function for case-insensitive string contains
func contains(s, substr string) bool {
	// Simple case-insensitive contains check
	// In production, you might want to use a proper text search
	return len(s) >= len(substr) && (s == substr ||
		// Simple implementation - could be improved
		s[0:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr)))
}
