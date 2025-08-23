package auth

import (
	"fmt"
	"go-template/gateways/auth/supabase"
)

func NewAuthProvider(config AuthConfig) (Provider, error) {
	switch config.Provider {
	case "supabase":
		if config.Supabase.URL == "" || config.Supabase.APIKey == "" {
			return nil, fmt.Errorf("supabase configuration missing: url and api_key required")
		}
		return supabase.NewSupabaseProvider(config.Supabase.URL, config.Supabase.APIKey), nil
	default:
		return nil, fmt.Errorf("unsupported auth provider: %s (supported: supabase)", config.Provider)
	}
}
