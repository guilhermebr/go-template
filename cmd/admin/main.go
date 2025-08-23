package main

import (
	"errors"
	"fmt"
	"go-template/app/admin"
	"go-template/internal/jwt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/guilhermebr/gox/logger"
)

// Injected on build time by ldflags.
var (
	BuildCommit = "undefined"
	BuildTime   = "undefined"
)

func main() {
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
		slog.String("app", "admin"),
		slog.String("environment", cfg.Environment),
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
		slog.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		slog.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	// Services (only JWT for token validation in middleware)
	jwtService := jwt.NewService(cfg.AuthSecretKey, cfg.AuthProvider, cfg.AuthTokenTTL)

	// Admin Handlers (connects to service API)
	adminHandlers := admin.NewHandlers(cfg.ServiceBaseURL, jwtService)
	router := admin.Router()
	router.Mount("/admin", adminHandlers.Routes())

	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Admin Panel OK"))
	})

	// SERVER
	server := http.Server{
		Handler:           router,
		Addr:              cfg.AdminAddress,
		ReadHeaderTimeout: 60 * time.Second,
	}
	log.Info("admin server started",
		slog.String("address", server.Addr),
	)

	if serverErr := server.ListenAndServe(); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		log.Error("failed to listen and serve admin server",
			slog.String("error", serverErr.Error()),
		)
	}
}
