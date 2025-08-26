package auth

import (
	"context"
	"go-template/domain/auth"
	"go-template/domain/entities"
	"go-template/internal/jwt"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid/v5"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/auth_uc.go . AuthUseCase
type AuthUseCase interface {
	Login(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error)
}

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/user_uc.go . UserUseCase
type UserUseCase interface {
	GetMe(ctx context.Context, userID uuid.UUID) (entities.User, error)
	CreateUser(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error)
}

type AuthHandler struct {
	authUC     AuthUseCase
	userUC     UserUseCase
	jwtService jwt.Service
	validator  *validator.Validate
}

func NewAuthHandler(authUC AuthUseCase, userUC UserUseCase, jwtService jwt.Service) *AuthHandler {
	return &AuthHandler{
		authUC:     authUC,
		userUC:     userUC,
		jwtService: jwtService,
		validator:  validator.New(),
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/me", h.GetMe)

	return r
}
