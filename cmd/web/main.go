package main

import (
	"context"
	"errors"
	"fmt"
	"go-template/app/web"
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
	if err := cfg.Load("WEB"); err != nil {
		panic(fmt.Errorf("loading config: %w", err))
	}

	// Logger
	log, err := logger.NewLogger("WEB")
	if err != nil {
		panic(fmt.Errorf("creating logger: %w", err))
	}

	log = log.With(
		slog.String("environment", cfg.Environment),
		slog.String("app", "web"),
	)

	mainlog := log.With(
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
		slog.Int("go_max_procs", runtime.GOMAXPROCS(0)),
		slog.Int("runtime_num_cpu", runtime.NumCPU()),
	)

	// Web Application Setup
	// ------------------------------------------
	webApp := web.New(web.Config{
		APIBaseURL:     cfg.APIBaseURL,
		CookieMaxAge:   cfg.CookieMaxAge,
		CookieSecure:   cfg.CookieSecure,
		CookieDomain:   cfg.CookieDomain,
		SessionTimeout: cfg.SessionTimeout,
	}, log)

	router := webApp.Routes()

	// SERVER
	// ------------------------------------------
	server := http.Server{
		Handler:           router,
		Addr:              cfg.Address,
		ReadHeaderTimeout: 60 * time.Second,
	}

	// Start server
	go func() {
		mainlog.Info("web server started",
			slog.String("address", server.Addr),
			slog.String("api_base_url", cfg.APIBaseURL),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			mainlog.Error("failed to listen and serve web server",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	mainlog.Info("shutting down web server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		mainlog.Error("failed to shutdown web server gracefully",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	mainlog.Info("web server stopped")
}
