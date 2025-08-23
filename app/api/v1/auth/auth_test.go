package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	apiMiddleware "go-template/app/api/middleware"
	"go-template/app/api/v1/auth/mocks"
	"go-template/domain"
	"go-template/domain/auth"
	"go-template/domain/entities"
	"go-template/internal/jwt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid/v5"
)

func TestAuthHandler_Register_Success(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		RegisterFunc: func(ctx context.Context, req auth.RegisterRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
	}
	h := NewAuthHandler(uc, nil)

	body, _ := json.Marshal(auth.RegisterRequest{Email: "a@b.com", Password: "123456"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp auth.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" || resp.User.Email != "a@b.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		RegisterFunc: func(ctx context.Context, req auth.RegisterRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
	}
	h := NewAuthHandler(uc, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("{"))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ValidationFailed(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		RegisterFunc: func(ctx context.Context, req auth.RegisterRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
	}
	h := NewAuthHandler(uc, nil)

	body, _ := json.Marshal(map[string]string{"email": "not-an-email"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
	}
	h := NewAuthHandler(uc, nil)

	body, _ := json.Marshal(auth.LoginRequest{Email: "a@b.com", Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_Login_AuthFailed(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, errors.New("auth")
		},
	}
	h := NewAuthHandler(uc, nil)

	body, _ := json.Marshal(auth.LoginRequest{Email: "a@b.com", Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_Success(t *testing.T) {
	uc := &mocks.UserUseCaseMock{
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}
	h := NewAuthHandler(nil, uc)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	claims := &jwt.Claims{UserID: "123", Email: "a@b.com"}
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_Unauthorized(t *testing.T) {
	uc := &mocks.UserUseCaseMock{
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{}, errors.New("auth")
		},
	}
	h := NewAuthHandler(nil, uc)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_NotFound(t *testing.T) {
	uc := &mocks.UserUseCaseMock{
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{}, domain.ErrNotFound
		},
	}
	h := NewAuthHandler(nil, uc)
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
