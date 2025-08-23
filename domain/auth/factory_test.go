package auth

import (
	"testing"
)

func TestNewAuthProvider_Supabase_Success(t *testing.T) {
	cfg := AuthConfig{
		Provider: "supabase",
		Supabase: SupabaseConfig{
			URL:    "http://localhost:54321",
			APIKey: "test-api-key",
		},
	}

	p, err := NewAuthProvider(cfg)
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

func TestNewAuthProvider_Supabase_MissingConfig(t *testing.T) {
	cases := []AuthConfig{
		{Provider: "supabase", Supabase: SupabaseConfig{URL: "", APIKey: "k"}},
		{Provider: "supabase", Supabase: SupabaseConfig{URL: "u", APIKey: ""}},
		{Provider: "supabase", Supabase: SupabaseConfig{}},
	}

	for _, cfg := range cases {
		if _, err := NewAuthProvider(cfg); err == nil {
			t.Fatalf("expected error for missing supabase config, got nil")
		}
	}
}

func TestNewAuthProvider_Unsupported(t *testing.T) {
	cfg := AuthConfig{Provider: "unknown"}
	if _, err := NewAuthProvider(cfg); err == nil {
		t.Fatalf("expected error for unsupported provider, got nil")
	}
}
