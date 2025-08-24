package main

import (
	"errors"
	"fmt"

	"github.com/ardanlabs/conf/v3"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Environment    string `conf:"env:ENVIRONMENT,default:development"`
	AdminAddress   string `conf:"env:ADMIN_ADDRESS,default:0.0.0.0:8080"`
	ServiceBaseURL string `conf:"env:SERVICE_BASE_URL,default:http://localhost:3000"`
	AuthSecretKey  string `conf:"env:AUTH_SECRET_KEY,default:dev-secret-change-me"`
	AuthTokenTTL   string `conf:"env:AUTH_TOKEN_TTL,default:24h"`
	AuthProvider   string `conf:"env:AUTH_PROVIDER,default:supabase"`
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
