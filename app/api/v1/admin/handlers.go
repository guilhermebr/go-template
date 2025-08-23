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
}

type AdminHandler struct {
	authUC     AuthUseCase
	userUC     UserUseCase
	jwtService jwt.Service
	authMw     *middleware.AuthMiddleware
	validator  *validator.Validate
}

func NewAdminHandler(authUC AuthUseCase, userUC UserUseCase, jwtService jwt.Service, authMw *middleware.AuthMiddleware) *AdminHandler {
	return &AdminHandler{
		authUC:     authUC,
		userUC:     userUC,
		jwtService: jwtService,
		authMw:     authMw,
		validator:  validator.New(),
	}
}

func (h *AdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Admin authentication endpoints (public)
	r.Post("/auth/login", h.AdminLogin)
	r.Post("/auth/logout", h.AdminLogout)
	r.Get("/auth/verify", h.VerifyAdminToken)

	// Protected admin endpoints
	r.Group(func(r chi.Router) {
		r.Use(h.authMw.RequireAdmin)

		// Dashboard stats
		r.Get("/dashboard/stats", h.GetDashboardStats)

		// User management
		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Get("/{id}", h.GetUser)
			r.Put("/{id}", h.UpdateUser)
			r.Delete("/{id}", h.DeleteUser)
			r.Get("/stats", h.GetUserStats)
		})

		// System settings (super admin only)
		r.Group(func(r chi.Router) {
			r.Use(h.authMw.RequireSuperAdmin)
			r.Get("/settings", h.GetSettings)
			r.Put("/settings", h.UpdateSettings)
		})
	})

	return r
}
