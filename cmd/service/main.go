// Package main provides the Go Template API service
//
//	@title						Go Template API
//	@version					1.0.0
//	@description				A Go template API built with Domain-Driven Design principles
//	@termsOfService				http://swagger.io/terms/
//	@contact.name				API Support
//	@contact.url				https://github.com/guilhermebr/go-template
//	@contact.email				support@example.com
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//	@host						localhost:8080
//	@BasePath					/
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.
//	@schemes					http https
package main

import (
	"context"
	"fmt"
	"go-template/app/api"
	appMiddleware "go-template/app/api/middleware"
	v1 "go-template/app/api/v1"
	"go-template/domain/auth"
	"go-template/domain/example"
	"go-template/domain/settings"
	"go-template/domain/user"
	"go-template/gateways/repository/pg"
	"go-template/internal/jwt"
	"log/slog"
	"os"

	"github.com/go-playground/validator/v10"

	httpPkg "github.com/guilhermebr/gox/http"

	"github.com/guilhermebr/gox/logger"
	"github.com/guilhermebr/gox/postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	// Import generated docs for swagger integration
	_ "go-template/docs"
)

// Injected on build time by ldflags.
var (
	BuildCommit = "undefined"
	BuildTime   = "undefined"
)

// Dependencies holds all application dependencies
type Dependencies struct {
	// Database
	DB   *pgxpool.Pool
	Repo *pg.Repository

	// Use Cases
	UserUseCase     *user.UseCase
	AuthUseCase     *auth.UseCase
	ExampleUseCase  example.UseCase
	SettingsUseCase *settings.UseCase

	// Services
	JWTService jwt.Service
	Validator  *validator.Validate

	// Middleware
	AuthMiddleware *appMiddleware.AuthMiddleware

	// Server
	Server *httpPkg.Server
}

// setupDependencies initializes all application dependencies
func setupDependencies(ctx context.Context, cfg Config, log *slog.Logger) (*Dependencies, error) {
	// Database
	conn, err := postgres.New(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("setting up database: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	repo := pg.NewRepository(conn)

	// Services
	jwtService := jwt.NewService(cfg.AuthSecretKey, cfg.AuthProvider, cfg.AuthTokenTTL)
	validator := validator.New()

	// Auth setup
	authConfigs := map[string]auth.AuthConfig{
		"supabase": {
			Provider: "supabase",
			Supabase: auth.SupabaseConfig{
				URL:    cfg.SupabaseURL,
				APIKey: cfg.SupabaseAPIKey,
			},
		},
	}

	authFactory := auth.NewProviderFactory(authConfigs)
	authProvider, err := authFactory.CreateProvider(cfg.AuthProvider)
	if err != nil {
		return nil, fmt.Errorf("creating auth provider: %w", err)
	}

	// Use Cases
	userUC := user.NewUseCase(repo.UserRepo, authFactory, cfg.AuthProvider)
	authUC := auth.NewUseCase(repo.UserRepo, authProvider, jwtService)
	exampleUC := example.New(repo.ExampleRepo)
	settingsUC := settings.NewUseCase(repo.SettingsRepo, log)

	// Middleware
	authMiddleware := appMiddleware.NewAuthMiddleware(jwtService)

	return &Dependencies{
		DB:              conn,
		Repo:            repo,
		UserUseCase:     userUC,
		AuthUseCase:     authUC,
		ExampleUseCase:  exampleUC,
		SettingsUseCase: settingsUC,
		JWTService:      jwtService,
		Validator:       validator,
		AuthMiddleware:  authMiddleware,
	}, nil
}

func main() {
	ctx := context.Background()

	var cfg Config
	if err := cfg.Load(""); err != nil {
		panic(fmt.Errorf("loading config: %w", err))
	}

	// Logger
	log, err := logger.NewLogger("")
	if err != nil {
		panic(fmt.Errorf("creating logger: %w", err))
	}

	log = log.With(
		slog.String("environment", cfg.Environment),
		slog.String("app", "service"),
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
	)

	// Setup dependencies
	deps, err := setupDependencies(ctx, cfg, log)
	if err != nil {
		log.Error("failed to setup dependencies",
			slog.String("error", err.Error()),
		)
		return
	}
	defer deps.DB.Close()

	// Handlers V1 and their dependencies
	apiV1 := v1.ApiHandlers{
		ExampleUseCase:  deps.ExampleUseCase,
		AuthUseCase:     deps.AuthUseCase,
		UserUseCase:     deps.UserUseCase,
		SettingsUseCase: deps.SettingsUseCase,
		AuthMiddleware:  deps.AuthMiddleware,
		JWTService:      deps.JWTService,
	}

	// Setup router with middleware
	router := api.Router()
	apiV1.Routes(router)

	server, err := httpPkg.NewServer("api", router, log)
	if err != nil {
		log.Error("failed to create server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Start server with graceful shutdown
	if err := server.StartWithGracefulShutdown(); err != nil {
		log.Error("server error",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
