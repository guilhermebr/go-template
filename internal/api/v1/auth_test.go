package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-template/domain"
	mauth "go-template/domain/auth/mocks"
	"go-template/domain/entities"
	"go-template/domain/user"
	muser "go-template/domain/user/mocks"
	apiMiddleware "go-template/internal/api/middleware"
	"go-template/internal/jwt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid/v5"
)

func newJWT() jwt.Service { return jwt.NewService("secret", "test", "1h") }

func TestAuthHandler_Register_Success(t *testing.T) {
	repo := &muser.RepositoryMock{CreateFunc: func(ctx context.Context, u entities.User) error { return nil }}
	provider := &mauth.ProviderMock{
		RegisterUserFunc: func(ctx context.Context, email string, password string) (string, error) { return "prov-123", nil },
		ProviderFunc:     func() string { return "supabase" },
	}
	uc := user.NewUseCase(repo, provider, newJWT())
	h := NewAuthHandler(uc)

	body, _ := json.Marshal(user.RegisterRequest{Email: "a@b.com", Password: "123456"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp user.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" || resp.User.Email != "a@b.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	uc := user.NewUseCase(&muser.RepositoryMock{}, &mauth.ProviderMock{}, newJWT())
	h := NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{"))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ValidationFailed(t *testing.T) {
	uc := user.NewUseCase(&muser.RepositoryMock{}, &mauth.ProviderMock{}, newJWT())
	h := NewAuthHandler(uc)

	body, _ := json.Marshal(map[string]string{"email": "not-an-email"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	existing := entities.User{ID: uuid.Must(uuid.NewV4()), Email: "a@b.com", AuthProvider: "supabase", AuthProviderID: "id"}
	repo := &muser.RepositoryMock{GetByEmailFunc: func(ctx context.Context, email string) (entities.User, error) { return existing, nil }}
	provider := &mauth.ProviderMock{LoginFunc: func(ctx context.Context, email string, password string) (string, error) { return "prov-123", nil }, ProviderFunc: func() string { return "supabase" }}
	uc := user.NewUseCase(repo, provider, newJWT())
	h := NewAuthHandler(uc)

	body, _ := json.Marshal(user.LoginRequest{Email: existing.Email, Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_Login_AuthFailed(t *testing.T) {
	provider := &mauth.ProviderMock{LoginFunc: func(ctx context.Context, email string, password string) (string, error) {
		return "", errors.New("auth")
	}}
	uc := user.NewUseCase(&muser.RepositoryMock{}, provider, newJWT())
	h := NewAuthHandler(uc)

	body, _ := json.Marshal(user.LoginRequest{Email: "a@b.com", Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_Success(t *testing.T) {
	u := entities.User{ID: uuid.Must(uuid.NewV4()), Email: "me@example.com"}
	repo := &muser.RepositoryMock{GetByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) { return u, nil }}
	uc := user.NewUseCase(repo, &mauth.ProviderMock{}, newJWT())
	h := NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	claims := &jwt.Claims{UserID: u.ID.String(), Email: u.Email}
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_Unauthorized(t *testing.T) {
	uc := user.NewUseCase(&muser.RepositoryMock{}, &mauth.ProviderMock{}, newJWT())
	h := NewAuthHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_NotFound(t *testing.T) {
	uc := user.NewUseCase(&muser.RepositoryMock{GetByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) {
		return entities.User{}, domain.ErrNotFound
	}}, &mauth.ProviderMock{}, newJWT())
	h := NewAuthHandler(uc)
	userID := uuid.Must(uuid.NewV4())
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	claims := &jwt.Claims{UserID: userID.String(), Email: "x@y.com"}
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
