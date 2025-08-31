package main

import (
	"errors"
	"fmt"

	"github.com/ardanlabs/conf/v3"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Environment string `conf:"env:ENVIRONMENT,default:development"`
	Address     string `conf:"env:ADDRESS,default:0.0.0.0:8081"`

	// API service configuration
	ApiBaseURL string `conf:"env:API_BASE_URL,default:http://localhost:3000"`

	// Session configuration
	CookieMaxAge   int    `conf:"env:COOKIE_MAX_AGE,default:86400"` // 24 hours
	CookieDomain   string `conf:"env:COOKIE_DOMAIN,default:localhost"`
	CookieSecure   bool   `conf:"env:COOKIE_SECURE,default:false"`
	SessionTimeout int    `conf:"env:SESSION_TIMEOUT,default:86400"` // 24 hours

	// Static files
	StaticPath string `conf:"env:STATIC_PATH,default:web/static"`
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
