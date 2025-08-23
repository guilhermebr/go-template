package auth

import (
	"context"
	"fmt"
	"go-template/domain"
	"go-template/domain/entities"
	"go-template/internal/jwt"
	"log/slog"
	"time"

	"github.com/gofrs/uuid/v5"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string        `json:"token"`
	User  entities.User `json:"user"`
}

type UseCase struct {
	repo         Repository
	authProvider Provider
	jwtService   jwt.Service
}

func NewUseCase(repo Repository, authProvider Provider, jwtService jwt.Service) *UseCase {
	return &UseCase{
		repo:         repo,
		authProvider: authProvider,
		jwtService:   jwtService,
	}
}

func (uc *UseCase) Register(ctx context.Context, req RegisterRequest) (AuthResponse, error) {
	slog.Info("starting user registration", "email", req.Email)

	// Register with auth provider (Supabase)
	authProviderID, err := uc.authProvider.RegisterUser(ctx, req.Email, req.Password)
	if err != nil {
		slog.Error("failed to register with auth provider", "error", err)
		return AuthResponse{}, fmt.Errorf("registration failed: %w", err)
	}

	// Create user in our database
	now := time.Now()
	user := entities.User{
		ID:             uuid.Must(uuid.NewV4()),
		Email:          req.Email,
		AuthProvider:   uc.authProvider.Provider(),
		AuthProviderID: authProviderID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.repo.Create(ctx, user); err != nil {
		slog.Error("failed to create user in database", "error", err)
		return AuthResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token
	token, err := uc.jwtService.GenerateToken(user.ID.String(), user.Email, user.AccountType.String())
	if err != nil {
		slog.Error("failed to generate JWT token", "error", err)
		return AuthResponse{}, fmt.Errorf("failed to generate token: %w", err)
	}

	slog.Info("user registered successfully", "user_id", user.ID)

	return AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (uc *UseCase) Login(ctx context.Context, req LoginRequest) (AuthResponse, error) {
	slog.Info("starting user login", "email", req.Email)

	// Authenticate with auth provider (Supabase)
	authProviderID, err := uc.authProvider.Login(ctx, req.Email, req.Password)
	if err != nil {
		slog.Error("authentication failed", "error", err)
		return AuthResponse{}, fmt.Errorf("authentication failed: %w", err)
	}

	// Get user from database
	user, err := uc.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == domain.ErrNotFound {
			// User doesn't exist in our database, create them
			now := time.Now()
			user = entities.User{
				ID:             uuid.Must(uuid.NewV4()),
				Email:          req.Email,
				AuthProvider:   uc.authProvider.Provider(),
				AuthProviderID: authProviderID,
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			if err := uc.repo.Create(ctx, user); err != nil {
				slog.Error("failed to create user during login", "error", err)
				return AuthResponse{}, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			slog.Error("failed to get user from database", "error", err)
			return AuthResponse{}, fmt.Errorf("failed to get user: %w", err)
		}
	}

	// Generate JWT token
	token, err := uc.jwtService.GenerateToken(user.ID.String(), user.Email, user.AccountType.String())
	if err != nil {
		slog.Error("failed to generate JWT token", "error", err)
		return AuthResponse{}, fmt.Errorf("failed to generate token: %w", err)
	}

	slog.Info("user login successful", "user_id", user.ID)

	return AuthResponse{
		Token: token,
		User:  user,
	}, nil
}
