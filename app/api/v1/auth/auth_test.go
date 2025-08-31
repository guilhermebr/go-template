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
	"time"

	"github.com/gofrs/uuid/v5"
)

// Create a real JWT service for testing
func createTestJWTService() jwt.Service {
	return jwt.NewService("test-secret", "test-issuer", "1h")
}

func TestAuthHandler_Register_Success(t *testing.T) {
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{
				ID:          uuid.Must(uuid.NewV4()),
				Email:       email,
				AccountType: entities.AccountTypeUser,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	body, _ := json.Marshal(RegisterRequest{Email: "a@b.com", Password: "123456"})
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
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{}, nil
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer([]byte("invalid json")))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ValidationFailed(t *testing.T) {
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{}, nil
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	// Invalid email and short password
	body, _ := json.Marshal(RegisterRequest{Email: "invalid-email", Password: "123"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Register_CreateUserFailed(t *testing.T) {
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{}, errors.New("creation failed")
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	body, _ := json.Marshal(RegisterRequest{Email: "a@b.com", Password: "123456"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{}, nil
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: "token",
				User:  entities.User{Email: "a@b.com"},
			}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	body, _ := json.Marshal(auth.LoginRequest{Email: "a@b.com", Password: "123456"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp auth.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" || resp.User.Email != "a@b.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthHandler_GetMe_Success(t *testing.T) {
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{}, nil
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{Email: "a@b.com"}, nil
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)

	// Add mock claims to context
	claims := &jwt.Claims{UserID: uuid.Must(uuid.NewV4()).String()}
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	h.GetMe(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe_NotFound(t *testing.T) {
	userUC := &mocks.UserUseCaseMock{
		CreateUserFunc: func(ctx context.Context, email, password, authProvider string, accountType entities.AccountType) (entities.User, error) {
			return entities.User{}, nil
		},
		GetMeFunc: func(ctx context.Context, userID uuid.UUID) (entities.User, error) {
			return entities.User{}, domain.ErrNotFound
		},
	}

	authUC := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, nil
		},
	}

	jwtService := createTestJWTService()

	h := NewAuthHandler(authUC, userUC, jwtService, apiMiddleware.NewAuthMiddleware(jwtService))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)

	// Add mock claims to context
	claims := &jwt.Claims{UserID: uuid.Must(uuid.NewV4()).String()}
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	h.GetMe(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
