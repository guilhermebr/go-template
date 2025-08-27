package admin

import (
	"context"
	"go-template/app/api/middleware"
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
	GetUserByID(ctx context.Context, id uuid.UUID) (entities.User, error)

	// Admin methods
	CreateUser(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error)
	ListUsers(ctx context.Context, page, pageSize int) ([]entities.User, int64, error)
	SearchUsers(ctx context.Context, page, pageSize int, search, accountType string) ([]entities.User, int64, error)
	UpdateUser(ctx context.Context, user entities.User) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	GetUserStats(ctx context.Context) (entities.UserStats, error)
}

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/settings_uc.go . SettingsUseCase
type SettingsUseCase interface {
	GetSettings(ctx context.Context) (*entities.SystemSettings, error)
	UpdateSettings(ctx context.Context, settings *entities.SystemSettings) error
}

type AdminHandler struct {
	authUC     AuthUseCase
	userUC     UserUseCase
	settingsUC SettingsUseCase
	jwtService jwt.Service
	authMw     *middleware.AuthMiddleware
	validator  *validator.Validate
}

func NewAdminHandler(authUC AuthUseCase, userUC UserUseCase, settingsUC SettingsUseCase, jwtService jwt.Service, authMw *middleware.AuthMiddleware) *AdminHandler {
	return &AdminHandler{
		authUC:     authUC,
		userUC:     userUC,
		settingsUC: settingsUC,
		jwtService: jwtService,
		authMw:     authMw,
		validator:  validator.New(),
	}
}

func (h *AdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Admin authentication endpoints (public)
	r.Post("/login", h.AdminLogin)
	r.Post("/logout", h.AdminLogout)
	r.Get("/verify", h.VerifyAdminToken)

	// Protected admin endpoints
	r.Group(func(r chi.Router) {
		r.Use(h.authMw.RequireAdmin)

		// Dashboard stats
		r.Get("/dashboard/stats", h.GetDashboardStats)

		// User management (all admins - validation handled in handlers)
		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Get("/{id}", h.GetUser)
			r.Put("/{id}", h.UpdateUser)
			r.Post("/", h.CreateUser)
			r.Delete("/{id}", h.DeleteUser)
			r.Get("/stats", h.GetUserStats)
		})

		// System settings (admin read-only)
		r.Get("/settings", h.GetSettings)
		r.Get("/settings/auth-providers", h.GetAvailableAuthProviders)

		// System settings (super admin only)
		r.Group(func(r chi.Router) {
			r.Use(h.authMw.RequireSuperAdmin)
			r.Put("/settings", h.UpdateSettings)
		})
	})

	return r
}
