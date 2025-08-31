package main

import (
	"errors"
	"fmt"

	"github.com/ardanlabs/conf/v3"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Environment string `conf:"env:ENVIRONMENT,default:development"`
	Address     string `conf:"env:ADDRESS,default:0.0.0.0:8080"`

	// API Configuration
	APIBaseURL string `conf:"env:API_BASE_URL,default:http://localhost:3000"`

	// Cookie Configuration
	CookieMaxAge   int    `conf:"env:COOKIE_MAX_AGE,default:86400"`    // 24 hours in seconds
	CookieSecure   bool   `conf:"env:COOKIE_SECURE,default:false"`     // Set to true in production with HTTPS
	CookieDomain   string `conf:"env:COOKIE_DOMAIN,default:localhost"` // Set to your domain in production
	SessionTimeout int    `conf:"env:SESSION_TIMEOUT,default:1440"`    // Session timeout in minutes (24 hours)
	StaticPath     string `conf:"env:STATIC_PATH,default:web/static"`
}

func (c *Config) Load(prefix string) error {
	if help, err := conf.Parse(prefix, c); err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return err
		}
		return err
	}
	return nil
}
