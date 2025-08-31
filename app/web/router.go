package web

import (
	"go-template/app/web/docs"
	"log/slog"
	"net/http"
	"time"

	gweb "go-template/gateways/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Config holds the configuration for the web application
type Config struct {
	APIBaseURL     string
	CookieMaxAge   int
	CookieSecure   bool
	CookieDomain   string
	SessionTimeout int
	StaticPath     string
}

// WebApp represents the web application
type WebApp struct {
	config   Config
	client   *gweb.Client
	handlers *Handlers
	auth     *AuthMiddleware
	logger   *slog.Logger
}

// New creates a new web application instance
func New(config Config, logger *slog.Logger) *WebApp {
	client := gweb.NewClient(config.APIBaseURL)
	auth := NewAuthMiddleware(client, config.CookieSecure, config.CookieDomain, config.CookieMaxAge)
	handlers := NewHandlers(client, logger, auth, config.StaticPath)

	return &WebApp{
		config:   config,
		client:   client,
		handlers: handlers,
		auth:     auth,
		logger:   logger,
	}
}

// Routes sets up and returns the router for the web application
func (app *WebApp) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware stack
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

	// Optional auth middleware for all routes (adds user to context if authenticated)
	r.Use(app.auth.OptionalAuth)
	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", app.handlers.fileServer))

	// Home page
	r.Get("/", app.handlers.HomePage)

	// Authentication routes
	r.Get("/login", app.handlers.LoginPage)
	r.Post("/login", app.handlers.LoginSubmit)
	r.Get("/register", app.handlers.RegisterPage)
	r.Post("/register", app.handlers.RegisterSubmit)
	r.Post("/logout", app.handlers.Logout)

	// Documentation routes (moved from service API)
	docsHandler := docs.NewHandler()
	r.Mount("/docs", docsHandler.Routes())

	// Generated Swagger UI (from code annotations) - now served locally
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/openapi-generated.json"),
	))

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(app.auth.RequireAuth)

		// User dashboard and profile
		r.Get("/dashboard", app.handlers.Dashboard)
		r.Get("/profile", app.handlers.Profile)

		// Additional protected routes can be added here
		// r.Get("/settings", app.handlers.Settings)
		// r.Get("/help", app.handlers.Help)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok","service":"web"}`))
	})

	return r
}
