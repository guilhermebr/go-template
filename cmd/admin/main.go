package main

import (
	"context"
	"errors"
	"fmt"
	"go-template/app/admin"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
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
	if err := cfg.Load("ADMIN"); err != nil {
		panic(fmt.Errorf("loading config: %w", err))
	}

	// Logger
	log, err := logger.NewLogger("")
	if err != nil {
		panic(fmt.Errorf("creating logger: %w", err))
	}

	log = log.With(
		slog.String("environment", cfg.Environment),
		slog.String("app", "admin"),
	)

	mainlog := log.With(
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
		slog.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		slog.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	// Create admin client
	apiClient := admin.NewClient(cfg.ApiBaseURL)

	// Create admin handlers (use cookie-based auth)
	adminHandlers := admin.NewHandlers(apiClient, cfg.SessionMaxAge, log)

	// Create router
	router := admin.NewRouter(adminHandlers, cfg.StaticPath)

	// Create server
	server := http.Server{
		Handler:           router,
		Addr:              cfg.Address,
		ReadHeaderTimeout: 60 * time.Second,
	}
	// Start server
	go func() {
		mainlog.Info("admin server started",
			slog.String("address", server.Addr),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mainlog.Error("failed to listen and serve admin server",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	mainlog.Info("shutting down admin server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		mainlog.Error("failed to shutdown admin server gracefully",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	mainlog.Info("admin server stopped")
}
