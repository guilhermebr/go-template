package main

import (
	"fmt"
	"go-template/app/web"
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
		slog.String("build_commit", BuildCommit),
		slog.String("build_time", BuildTime),
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

	// Create admin server
	server, err := httpPkg.NewServer("web", router, log)
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
