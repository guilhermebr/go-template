package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	apiMiddleware "go-template/app/api/middleware"
	"go-template/app/api/v1/admin/mocks"
	"go-template/domain/auth"
	"go-template/domain/entities"
	"go-template/internal/jwt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
)

func newTestJWT() jwt.Service {
	return jwt.NewService("test-secret", "test-issuer", "1h")
}

func TestAdminLogin_Success_Admin(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{
				Token: func() string {
					js := newTestJWT()
					t, _ := js.GenerateToken("user-1", "admin@x.com", entities.AccountTypeAdmin.String())
					return t
				}(),
				User: entities.User{Email: "admin@x.com", AccountType: entities.AccountTypeAdmin},
			}, nil
		},
	}
	jh := newTestJWT()
	ah := NewAdminHandler(uc, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	body, _ := json.Marshal(AdminLoginRequest{Email: "admin@x.com", Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	ah.AdminLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp AdminLoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Token == "" || resp.User.Email != "admin@x.com" || resp.AccountType != entities.AccountTypeAdmin.String() {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAdminLogin_Forbidden_NonAdmin(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			js := newTestJWT()
			t, _ := js.GenerateToken("user-2", "user@x.com", entities.AccountTypeUser.String())
			return auth.AuthResponse{
				Token: t,
				User:  entities.User{Email: "user@x.com", AccountType: entities.AccountTypeUser},
			}, nil
		},
	}
	jh := newTestJWT()
	h := NewAdminHandler(uc, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	body, _ := json.Marshal(AdminLoginRequest{Email: "user@x.com", Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.AdminLogin(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAdminLogin_BadJSON(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{}
	jh := newTestJWT()
	h := NewAdminHandler(uc, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("{"))
	w := httptest.NewRecorder()

	h.AdminLogin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminLogin_ValidationFailed(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{}
	jh := newTestJWT()
	h := NewAdminHandler(uc, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	// invalid email and missing password
	body, _ := json.Marshal(AdminLoginRequest{Email: "not-an-email"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.AdminLogin(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminLogin_AuthFailed(t *testing.T) {
	uc := &mocks.AuthUseCaseMock{
		LoginFunc: func(ctx context.Context, req auth.LoginRequest) (auth.AuthResponse, error) {
			return auth.AuthResponse{}, errors.New("auth failed")
		},
	}
	jh := newTestJWT()
	h := NewAdminHandler(uc, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	body, _ := json.Marshal(AdminLoginRequest{Email: "admin@x.com", Password: "pwd"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.AdminLogin(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestVerifyAdminToken_Success(t *testing.T) {
	jh := newTestJWT()
	// Generate a real token and parse claims so ExpiresAt is populated
	tok, _ := jh.GenerateToken("u1", "a@b.com", entities.AccountTypeAdmin.String())
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()

	h.VerifyAdminToken(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestVerifyAdminToken_Unauthorized(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	w := httptest.NewRecorder()

	h.VerifyAdminToken(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodGet, "/users/invalid", nil)
	w := httptest.NewRecorder()

	// add route param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	jh := newTestJWT()
	uc := &mocks.UserUseCaseMock{
		GetUserByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) {
			return entities.User{}, errors.New("not found")
		},
	}
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, uc, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	uid := uuid.Must(uuid.NewV4())
	req := httptest.NewRequest(http.MethodGet, "/users/"+uid.String(), nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uid.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetUser(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetUser_Success(t *testing.T) {
	jh := newTestJWT()
	u := entities.User{ID: uuid.Must(uuid.NewV4()), Email: "a@b.com", AccountType: entities.AccountTypeAdmin}
	uc := &mocks.UserUseCaseMock{
		GetUserByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) {
			return u, nil
		},
	}
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, uc, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodGet, "/users/"+u.ID.String(), nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", u.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got entities.User
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.ID != u.ID || got.Email != u.Email {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestUpdateUser_InvalidID(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodPut, "/users/invalid", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateUser_BadJSON(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	uID := uuid.Must(uuid.NewV4())
	req := httptest.NewRequest(http.MethodPut, "/users/"+uID.String(), bytes.NewBufferString("{"))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateUser_ValidationFailed(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	uID := uuid.Must(uuid.NewV4())
	// missing required account_type
	body, _ := json.Marshal(map[string]string{"email": "invalid-email"})
	req := httptest.NewRequest(http.MethodPut, "/users/"+uID.String(), bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateUser_Success(t *testing.T) {
	jh := newTestJWT()
	existing := entities.User{ID: uuid.Must(uuid.NewV4()), Email: "old@x.com", AccountType: entities.AccountTypeAdmin}
	uc := &mocks.UserUseCaseMock{
		GetUserByIDFunc: func(ctx context.Context, id uuid.UUID) (entities.User, error) {
			return existing, nil
		},
	}
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, uc, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	body, _ := json.Marshal(UpdateUserRequest{Email: "new@x.com", AccountType: entities.AccountTypeSuperAdmin})
	req := httptest.NewRequest(http.MethodPut, "/users/"+existing.ID.String(), bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", existing.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got entities.User
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Email != "new@x.com" || got.AccountType != entities.AccountTypeSuperAdmin {
		t.Fatalf("unexpected updated user: %+v", got)
	}
}

func TestDeleteUser_InvalidID(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	req := httptest.NewRequest(http.MethodDelete, "/users/invalid", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DeleteUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteUser_SelfDelete(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	uID := uuid.Must(uuid.NewV4())
	req := httptest.NewRequest(http.MethodDelete, "/users/"+uID.String(), nil)
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, &jwt.Claims{UserID: uID.String(), Email: "a@b.com"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DeleteUser(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	uID := uuid.Must(uuid.NewV4())
	req := httptest.NewRequest(http.MethodDelete, "/users/"+uID.String(), nil)
	// add admin context claims (different from target user) to authorize deletion
	adminID := uuid.Must(uuid.NewV4())
	ctx := context.WithValue(req.Context(), apiMiddleware.UserContextKey, &jwt.Claims{UserID: adminID.String(), Email: "admin@x.com", AccountType: entities.AccountTypeSuperAdmin.String()})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DeleteUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestMiscEndpoints(t *testing.T) {
	jh := newTestJWT()
	h := NewAdminHandler(&mocks.AuthUseCaseMock{}, &mocks.UserUseCaseMock{}, &mocks.SettingsUseCaseMock{}, jh, apiMiddleware.NewAuthMiddleware(jh))

	t.Run("DashboardStats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
		w := httptest.NewRecorder()
		h.GetDashboardStats(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("ListUsers default pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()
		h.ListUsers(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("GetUserStats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/stats", nil)
		w := httptest.NewRecorder()
		h.GetUserStats(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("GetSettings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		w := httptest.NewRecorder()
		h.GetSettings(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("UpdateSettings bad json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewBufferString("{"))
		w := httptest.NewRecorder()
		h.UpdateSettings(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("UpdateSettings ok", func(t *testing.T) {
		body := bytes.NewBufferString(`{"maintenance_mode":true}`)
		req := httptest.NewRequest(http.MethodPut, "/settings", body)
		w := httptest.NewRecorder()
		h.UpdateSettings(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}
