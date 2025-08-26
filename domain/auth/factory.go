package auth

import (
	"fmt"
	"go-template/gateways/auth/supabase"
)

//go:generate moq -skip-ensure -stub -pkg mocks -out mocks/auth_provider_factory.go . AuthProviderFactory

// AuthProviderFactory creates auth provider instances by name
type AuthProviderFactory interface {
	CreateProvider(providerName string) (Provider, error)
	GetSupportedProviders() []string
}

// ProviderFactory implements AuthProviderFactory
type ProviderFactory struct {
	configs map[string]AuthConfig
}

// NewProviderFactory creates a new provider factory with auth configurations
func NewProviderFactory(configs map[string]AuthConfig) *ProviderFactory {
	return &ProviderFactory{
		configs: configs,
	}
}

// CreateProvider creates an auth provider instance by name
func (f *ProviderFactory) CreateProvider(providerName string) (Provider, error) {
	config, exists := f.configs[providerName]
	if !exists {
		return nil, fmt.Errorf("no configuration found for provider: %s", providerName)
	}

	switch providerName {
	case "supabase":
		if config.Supabase.URL == "" || config.Supabase.APIKey == "" {
			return nil, fmt.Errorf("supabase configuration missing: url and api_key required")
		}
		return supabase.NewSupabaseProvider(config.Supabase.URL, config.Supabase.APIKey), nil
	default:
		return nil, fmt.Errorf("unsupported auth provider: %s (supported: supabase)", providerName)
	}
}

// GetSupportedProviders returns list of supported provider names
func (f *ProviderFactory) GetSupportedProviders() []string {
	var providers []string
	for name := range f.configs {
		providers = append(providers, name)
	}
	return providers
}
