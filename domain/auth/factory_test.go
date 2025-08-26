package auth

import (
	"testing"
)

func TestProviderFactory_CreateProvider_Supabase_Success(t *testing.T) {
	configs := map[string]AuthConfig{
		"supabase": {
			Provider: "supabase",
			Supabase: SupabaseConfig{
				URL:    "http://localhost:54321",
				APIKey: "test-api-key",
			},
		},
	}

	factory := NewProviderFactory(configs)
	p, err := factory.CreateProvider("supabase")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p == nil {
		t.Fatalf("expected provider, got nil")
	}
	if got := p.Provider(); got != "supabase" {
		t.Fatalf("expected provider name 'supabase', got %q", got)
	}
}

func TestProviderFactory_CreateProvider_Supabase_MissingConfig(t *testing.T) {
	configs := []map[string]AuthConfig{
		{"supabase": {Provider: "supabase", Supabase: SupabaseConfig{URL: "", APIKey: "k"}}},
		{"supabase": {Provider: "supabase", Supabase: SupabaseConfig{URL: "u", APIKey: ""}}},
		{"supabase": {Provider: "supabase", Supabase: SupabaseConfig{}}},
	}

	for _, configMap := range configs {
		factory := NewProviderFactory(configMap)
		if _, err := factory.CreateProvider("supabase"); err == nil {
			t.Fatalf("expected error for missing supabase config, got nil")
		}
	}
}

func TestProviderFactory_CreateProvider_Unsupported(t *testing.T) {
	configs := map[string]AuthConfig{
		"supabase": {
			Provider: "supabase",
			Supabase: SupabaseConfig{
				URL:    "http://localhost:54321",
				APIKey: "test-api-key",
			},
		},
	}

	factory := NewProviderFactory(configs)
	if _, err := factory.CreateProvider("unknown"); err == nil {
		t.Fatalf("expected error for unsupported provider, got nil")
	}
}

func TestProviderFactory_GetSupportedProviders(t *testing.T) {
	configs := map[string]AuthConfig{
		"supabase": {
			Provider: "supabase",
			Supabase: SupabaseConfig{
				URL:    "http://localhost:54321",
				APIKey: "test-api-key",
			},
		},
	}

	factory := NewProviderFactory(configs)
	providers := factory.GetSupportedProviders()
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0] != "supabase" {
		t.Fatalf("expected 'supabase', got %q", providers[0])
	}
}