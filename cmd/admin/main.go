package main

import (
	"fmt"
	"go-template/app/admin"
	"log/slog"
	"os"

	httpPkg "github.com/guilhermebr/gox/http"

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
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
	)

	app := admin.New(admin.Config{
		APIBaseURL:     cfg.ApiBaseURL,
		CookieMaxAge:   cfg.CookieMaxAge,
		CookieSecure:   cfg.CookieSecure,
		CookieDomain:   cfg.CookieDomain,
		SessionTimeout: cfg.SessionTimeout,
		StaticPath:     cfg.StaticPath,
	}, log)

	// Create admin server
	server, err := httpPkg.NewServer("admin", app.Routes(), log)
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
