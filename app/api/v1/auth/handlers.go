package auth

import (
	"context"
	"go-template/domain/auth"
	"go-template/domain/entities"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid/v5"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/auth_uc.go . AuthUseCase
type AuthUseCase interface {
	Register(ctx context.Context, req auth.RegisterRequest) (auth.AuthResponse, error)
	Login(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error)
}

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/user_uc.go . UserUseCase
type UserUseCase interface {
	GetMe(ctx context.Context, userID uuid.UUID) (entities.User, error)
}

type AuthHandler struct {
	authUC    AuthUseCase
	userUC    UserUseCase
	validator *validator.Validate
}

func NewAuthHandler(authUC AuthUseCase, userUC UserUseCase) *AuthHandler {
	return &AuthHandler{
		authUC:    authUC,
		userUC:    userUC,
		validator: validator.New(),
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/me", h.GetMe)

	return r
}
