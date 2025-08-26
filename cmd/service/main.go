package main

import (
	"context"
	"errors"
	"fmt"
	"go-template/app/api"
	"go-template/app/api/middleware"
	v1 "go-template/app/api/v1"
	"go-template/domain/auth"
	"go-template/domain/example"
	"go-template/domain/settings"
	"go-template/domain/user"
	"go-template/gateways/repository/pg"
	"go-template/internal/jwt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/guilhermebr/gox/logger"
	"github.com/guilhermebr/gox/postgres"
)

// Injected on build time by ldflags.
var (
	BuildCommit = "undefined"
	BuildTime   = "undefined"
)

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
	)

	mainlog := log.With(
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
		slog.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		slog.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	// Repositories
	conn, err := postgres.New(ctx, "")
	if err != nil {
		mainlog.Error("failed to setup postgres",
			slog.String("error", err.Error()),
		)
		return
	}
	defer conn.Close()

	err = conn.Ping(ctx)
	if err != nil {
		mainlog.Error("failed to reach postgres",
			slog.String("error", err.Error()),
		)
		return
	}
	repo := pg.NewRepository(conn)

	// Create auth provider configurations
	authConfigs := map[string]auth.AuthConfig{
		"supabase": {
			Provider: "supabase",
			Supabase: auth.SupabaseConfig{
				URL:    cfg.SupabaseURL,
				APIKey: cfg.SupabaseAPIKey,
			},
		},
	}

	// Create auth provider factory
	authFactory := auth.NewProviderFactory(authConfigs)

	// Create main auth provider for the auth use case (backwards compatibility)
	authProvider, err := authFactory.CreateProvider(cfg.AuthProvider)
	if err != nil {
		mainlog.Error("failed to setup auth provider",
			slog.String("error", err.Error()),
		)
		return
	}

	jwtService := jwt.NewService(cfg.AuthSecretKey, cfg.AuthProvider, cfg.AuthTokenTTL)
	userUC := user.NewUseCase(repo.UserRepo, authFactory, cfg.AuthProvider)
	authUC := auth.NewUseCase(repo.UserRepo, authProvider, jwtService)
	settingsUC := settings.NewUseCase(repo.SettingsRepo, log)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)
	exampleUC := example.New(repo.ExampleRepo)

	// Handlers V1 and their dependencies
	// ------------------------------------------
	apiV1 := v1.ApiHandlers{
		ExampleUseCase:  exampleUC,
		AuthUseCase:     authUC,
		UserUseCase:     userUC,
		SettingsUseCase: settingsUC,
		AuthMiddleware:  authMiddleware,
		JWTService:      jwtService,
	}
	router := api.Router()
	apiV1.Routes(router)

	// SERVER
	// ------------------------------------------
	server := http.Server{
		Handler:           router,
		Addr:              cfg.ApiAddress,
		ReadHeaderTimeout: 60 * time.Second,
	}

	// Start server
	go func() {
		mainlog.Info("service server started",
			slog.String("address", server.Addr),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mainlog.Error("failed to listen and serve service server",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	mainlog.Info("shutting down service server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		mainlog.Error("failed to shutdown service server gracefully",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	mainlog.Info("service server stopped")
}
