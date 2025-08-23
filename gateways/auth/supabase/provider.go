package supabase

import (
	"context"
	"fmt"
	"go-template/domain/entities"

	"github.com/gofrs/uuid/v5"
	"github.com/supabase-community/gotrue-go/types"

	"github.com/supabase-community/supabase-go"
)

type SupabaseProvider struct {
	client *supabase.Client
}

func NewSupabaseProvider(url, apiKey string) *SupabaseProvider {
	client, _ := supabase.NewClient(url, apiKey, nil)
	return &SupabaseProvider{
		client: client,
	}
}

func (p *SupabaseProvider) Provider() string {
	return "supabase"
}

func (p *SupabaseProvider) RegisterUser(ctx context.Context, email, password string) (string, error) {
	resp, err := p.client.Auth.Signup(types.SignupRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("failed to register user: %w", err)
	}

	if resp.User.ID.String() == "" {
		return "", fmt.Errorf("no user ID received from Supabase")
	}

	return resp.User.ID.String(), nil
}

func (p *SupabaseProvider) Login(ctx context.Context, email, password string) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("supabase client not initialized")
	}

	// Use the Auth client SignInWithEmailPassword method
	resp, err := p.client.Auth.SignInWithEmailPassword(email, password)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate with Supabase: %w", err)
	}

	if resp.AccessToken == "" {
		return "", fmt.Errorf("no access token received from Supabase")
	}

	return resp.AccessToken, nil
}

func (p *SupabaseProvider) ValidateToken(ctx context.Context, token string) (*entities.User, error) {
	if p.client == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	// Update the auth session with the token
	session := types.Session{
		AccessToken: token,
	}
	p.client.UpdateAuthSession(session)

	user, err := p.client.Auth.GetUser()
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("invalid token: no user found")
	}

	return &entities.User{
		ID:             uuid.Nil,
		Email:          user.Email,
		AuthProvider:   "supabase",
		AuthProviderID: user.ID.String(),
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}
