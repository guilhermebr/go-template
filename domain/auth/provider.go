package auth

import (
	"context"
	"go-template/domain/entities"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/provider.go . Provider
type Provider interface {
	Provider() string
	RegisterUser(ctx context.Context, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	ValidateToken(ctx context.Context, token string) (*entities.User, error)
}

type AuthConfig struct {
	Provider string
	Supabase SupabaseConfig
}

type SupabaseConfig struct {
	URL    string `conf:"required"`
	APIKey string `conf:"required"`
}
