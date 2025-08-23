package auth

import (
	"context"
	"go-template/domain/entities"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/repository.go . Repository

type Repository interface {
	Create(ctx context.Context, user entities.User) error
	GetByEmail(ctx context.Context, email string) (entities.User, error)
}
