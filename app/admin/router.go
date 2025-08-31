package admin

import (
	gweb "go-template/gateways/web"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Config struct {
	APIBaseURL     string
	CookieMaxAge   int
	CookieSecure   bool
	CookieDomain   string
	SessionTimeout int
	StaticPath     string
}

type AdminApp struct {
	handlers *Handlers
	auth     *AuthMiddleware
	logger   *slog.Logger
}

func New(cfg Config, log *slog.Logger) *AdminApp {
	client := gweb.NewClient(cfg.APIBaseURL)
	auth := NewAuthMiddleware(client, cfg.CookieSecure, cfg.CookieDomain, cfg.CookieMaxAge)
	handlers := NewHandlers(client, auth, log, cfg.StaticPath)

	return &AdminApp{
		handlers: handlers,
		auth:     auth,
		logger:   log,
	}
}

func (app *AdminApp) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", app.handlers.fileServer))

	// Public routes (no auth required)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	r.Get("/login", app.handlers.LoginPage)
	r.Post("/login", app.handlers.LoginSubmit)

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(app.auth.RequireAuth)

		r.Get("/dashboard", app.handlers.Dashboard)
		r.Post("/logout", app.handlers.Logout)

		// User management (all admins - validation handled in handlers)
		r.Get("/users", app.handlers.UsersPage)
		r.Get("/users/{id}", app.handlers.UserDetail)
		r.Post("/users/update", app.handlers.UpdateUser)
		r.Post("/users/create", app.handlers.CreateUser)
		r.Post("/users/delete", app.handlers.DeleteUser)

		// Settings (super admin only)
		r.Group(func(r chi.Router) {
			r.Get("/settings", app.handlers.SettingsPage)
			r.Get("/settings/auth-providers", app.handlers.GetAuthProviders)
		})

		r.Group(func(r chi.Router) {
			r.Use(app.auth.RequireSuperAdmin)
			r.Post("/settings", app.handlers.UpdateSettings)
		})

		// HTMX/API endpoints for dynamic updates
		r.Route("/api", func(r chi.Router) {
			r.Get("/stats", app.handlers.GetStatsAPI)
			r.Get("/users", app.handlers.GetUsersAPI)
			r.Post("/users/{id}/toggle", app.handlers.ToggleUserAPI)
		})
	})

	return r
}
