package user

import (
	"context"
	"go-template/domain/entities"

	"github.com/gofrs/uuid/v5"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/repository.go . Repository

type Repository interface {
	Create(ctx context.Context, user entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (entities.User, error)
	GetByEmail(ctx context.Context, email string) (entities.User, error)
	Update(ctx context.Context, user entities.User) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Admin-specific methods
	ListUsers(ctx context.Context, params entities.ListUsersParams) ([]entities.User, error)
	CountUsers(ctx context.Context) (int64, error)
	CountUsersByAccountType(ctx context.Context, accountType entities.AccountType) (int64, error)
	GetUserStats(ctx context.Context) (entities.UserStats, error)
}
