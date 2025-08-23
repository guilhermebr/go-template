package user

import (
	"context"
	"go-template/domain/entities"
	"log/slog"

	"github.com/gofrs/uuid/v5"
)

type UseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *UseCase {
	return &UseCase{
		repo: repo,
	}
}

func (uc *UseCase) GetUserByID(ctx context.Context, userID uuid.UUID) (entities.User, error) {
	slog.Info("getting user by ID", "user_id", userID)

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
