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
}
