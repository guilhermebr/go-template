package v1

import (
	"go-template/app/api/middleware"
	"go-template/app/api/v1/admin"
	"go-template/app/api/v1/auth"
	"go-template/app/api/v1/example"
	authDomain "go-template/domain/auth"
	"go-template/domain/settings"
	"go-template/domain/user"
	"go-template/internal/jwt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ApiHandlers struct {
	ExampleUseCase  example.ExampleUseCase
	AuthUseCase     *authDomain.UseCase
	UserUseCase     *user.UseCase
	SettingsUseCase *settings.UseCase
	AuthMiddleware  *middleware.AuthMiddleware
	JWTService      jwt.Service
}

func (h *ApiHandlers) Routes(r chi.Router) {
	// Health check
	r.Get("/health", h.Health)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (mixed public/protected)
		authHandler := auth.NewAuthHandler(h.AuthUseCase, h.UserUseCase, h.JWTService, h.AuthMiddleware)
		r.Mount("/auth", authHandler.Routes())

		// Example routes (protected)
		exampleHandler := example.NewExampleHandler(h.ExampleUseCase, h.AuthMiddleware)
		r.Mount("/example", exampleHandler.Routes())
	})

	// Admin routes (protected)
	adminHandler := admin.NewAdminHandler(h.AuthUseCase, h.UserUseCase, h.SettingsUseCase, h.JWTService, h.AuthMiddleware)
	r.Mount("/admin/v1", adminHandler.Routes())

}

func (h *ApiHandlers) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
