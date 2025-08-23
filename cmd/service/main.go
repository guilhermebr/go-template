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
	"go-template/domain/user"
	"go-template/gateways/repository/pg"
	"go-template/internal/jwt"
	"log/slog"
	"net/http"
	"runtime"
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
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
		slog.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		slog.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	// Repositories
	conn, err := postgres.New(ctx, "")
	if err != nil {
		log.Error("failed to setup postgres",
			slog.String("error", err.Error()),
		)
		return
	}
	defer conn.Close()

	err = conn.Ping(ctx)
	if err != nil {
		log.Error("failed to reach postgres",
			slog.String("error", err.Error()),
		)
		return
	}
	repo := pg.NewRepository(conn)

	authProvider, err := auth.NewAuthProvider(auth.AuthConfig{
		Provider: cfg.AuthProvider,
		Supabase: auth.SupabaseConfig{
			URL:    cfg.SupabaseURL,
			APIKey: cfg.SupabaseAPIKey,
		},
	})
	if err != nil {
		log.Error("failed to setup auth provider",
			slog.String("error", err.Error()),
		)
		return
	}

	jwtService := jwt.NewService(cfg.AuthSecretKey, cfg.AuthProvider, cfg.AuthTokenTTL)
	userUC := user.NewUseCase(repo.UserRepo)
	authUC := auth.NewUseCase(repo.UserRepo, authProvider, jwtService)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)
	exampleUC := example.New(repo.ExampleRepo)

	// Handlers V1 and their dependencies
	// ------------------------------------------
	apiV1 := v1.ApiHandlers{
		ExampleUseCase: exampleUC,
		AuthUseCase:    authUC,
		UserUseCase:    userUC,
		AuthMiddleware: authMiddleware,
		JWTService:     jwtService,
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
	log.Info("server started",
		slog.String("address", server.Addr),
	)

	if serverErr := server.ListenAndServe(); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		log.Error("failed to listen and serve server",
			slog.String("error", serverErr.Error()),
		)
	}
}
