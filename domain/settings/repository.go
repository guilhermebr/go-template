package settings

import (
	"context"
	"go-template/domain/entities"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/repository.go . Repository

type Repository interface {
	GetSettings(ctx context.Context) (*entities.SystemSettings, error)
	UpdateSettings(ctx context.Context, settings *entities.SystemSettings) error
	GetSetting(ctx context.Context, key string) (any, error)
	SetSetting(ctx context.Context, key string, value any) error
}