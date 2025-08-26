package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"go-template/app/admin/templates"
	"go-template/domain/entities"
	"go-template/internal/types"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"
)

type contextKey string

const userContextKey contextKey = "user"

const (
	CookieToken       = "admin_token"
	CookieUserID      = "admin_user_id"
	CookieUserEmail   = "admin_user_email"
	CookieAccountType = "admin_account_type"
	CookieExpiresAt   = "admin_expires_at"
)

type Handlers struct {
	client       *Client
	logger       *slog.Logger
	cookieMaxAge int
}

func NewHandlers(client *Client, cookieMaxAge int, logger *slog.Logger) *Handlers {
	return &Handlers{
		client:       client,
		logger:       logger,
		cookieMaxAge: cookieMaxAge,
	}
}

// Auth middleware
func (h *Handlers) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getCookieValue(r, CookieToken)
		if token == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Set token in client and verify
		h.client.SetAuthToken(token)
		if err := h.client.VerifyToken(); err != nil {
			h.clearAuthCookies(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Build user context from cookies (minimal fields)
		var user entities.User
		if idStr := getCookieValue(r, CookieUserID); idStr != "" {
			if id, err := uuid.FromString(idStr); err == nil {
				user.ID = id
			}
		}
		user.Email = getCookieValue(r, CookieUserEmail)
		user.AccountType = entities.AccountType(getCookieValue(r, CookieAccountType))

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handlers) getUserFromContext(r *http.Request) *entities.User {
	if user, ok := r.Context().Value(userContextKey).(entities.User); ok {
		return &user
	}
	return nil
}

// Page handlers
func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to dashboard
	if getCookieValue(r, CookieToken) != "" {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title": "Admin Login",
		"Error": r.URL.Query().Get("error"),
	}

	renderTemplate(w, "login.templ", data)
}

func (h *Handlers) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error=missing_credentials", http.StatusSeeOther)
		return
	}

	resp, err := h.client.AdminLogin(email, password)
	if err != nil {
		h.logger.Error("admin login failed", slog.String("error", err.Error()))
		http.Redirect(w, r, "/login?error=invalid_credentials", http.StatusSeeOther)
		return
	}

	// Set auth cookies
	h.setAuthCookies(w, resp)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear cookies
	h.clearAuthCookies(w)

	// Call API logout
	h.client.AdminLogout()

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stats, err := h.client.GetDashboardStats()
	if err != nil {
		h.logger.Error("failed to get dashboard stats", slog.String("error", err.Error()))
		stats = &types.DashboardStats{} // Use empty stats on error
	}

	data := map[string]interface{}{
		"Title": "Admin Dashboard",
		"User":  user,
		"Stats": stats,
	}

	renderTemplate(w, "dashboard.templ", data)
}

func (h *Handlers) UsersPage(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Parse pagination
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if psStr := r.URL.Query().Get("page_size"); psStr != "" {
		if ps, err := strconv.Atoi(psStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Get search and filter parameters
	search := r.URL.Query().Get("search")
	accountType := r.URL.Query().Get("account_type")

	users, err := h.client.ListUsersWithFilter(page, pageSize, search, accountType)
	if err != nil {
		h.logger.Error("failed to get users", slog.String("error", err.Error()))
		users = &types.UserListResponse{} // Use empty response on error
	}

	data := map[string]interface{}{
		"Title": "User Management",
		"User":  user,
		"Users": users,
	}

	renderTemplate(w, "users.templ", data)
}

func (h *Handlers) UserDetail(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	userID := chi.URLParam(r, "id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// If it's an HTMX request for JSON data, return user data
	if r.Header.Get("HX-Request") == "true" {
		userData, err := h.client.GetUser(userID)
		if err != nil {
			h.logger.Error("failed to get user", slog.String("error", err.Error()))
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userData)
		return
	}

	// For non-HTMX requests, redirect to list for now
	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Extract user ID from form
	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	accountTypeStr := r.FormValue("account_type")
	accountType := entities.AccountType(accountTypeStr)

	req := UpdateUserRequest{
		AccountType: accountType,
	}

	if email := r.FormValue("email"); email != "" {
		req.Email = email
	}

	_, err := h.client.UpdateUser(userID, req)
	if err != nil {
		h.logger.Error("failed to update user", slog.String("error", err.Error()))
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// If HX-Request, return refreshed users table fragment (preserve container id)
	if r.Header.Get("HX-Request") == "true" {
		page := 1
		pageSize := 20
		if v := r.URL.Query().Get("page"); v != "" {
			if p, err := strconv.Atoi(v); err == nil && p > 0 {
				page = p
			}
		}
		if v := r.URL.Query().Get("page_size"); v != "" {
			if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
				pageSize = ps
			}
		}

		users, err := h.client.ListUsers(page, pageSize)
		if err != nil {
			users = &types.UserListResponse{}
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div id="users-table">`))
		_ = templates.UsersTable(users).Render(context.Background(), w)
		w.Write([]byte(`</div>`))
		return
	}

	// Redirect back to users page
	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	accountTypeStr := r.FormValue("account_type")
	authProvider := r.FormValue("auth_provider")

	if email == "" || password == "" || accountTypeStr == "" || authProvider == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	accountType := entities.AccountType(accountTypeStr)
	req := CreateUserRequest{
		Email:        email,
		Password:     password,
		AccountType:  accountType,
		AuthProvider: authProvider,
	}

	_, err := h.client.CreateUser(req)
	if err != nil {
		h.logger.Error("failed to create user", slog.String("error", err.Error()))
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// If HX-Request, return refreshed users table fragment
	if r.Header.Get("HX-Request") == "true" {
		page := 1
		pageSize := 20
		users, err := h.client.ListUsers(page, pageSize)
		if err != nil {
			users = &types.UserListResponse{}
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div id="users-table">`))
		_ = templates.UsersTable(users).Render(context.Background(), w)
		w.Write([]byte(`</div>`))
		return
	}

	// Redirect back to users page
	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	userID := r.FormValue("user_id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	if err := h.client.DeleteUser(userID); err != nil {
		h.logger.Error("failed to delete user", slog.String("error", err.Error()))
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// If HX-Request, return refreshed users table fragment (preserve container id)
	if r.Header.Get("HX-Request") == "true" {
		page := 1
		pageSize := 20
		if v := r.URL.Query().Get("page"); v != "" {
			if p, err := strconv.Atoi(v); err == nil && p > 0 {
				page = p
			}
		}
		if v := r.URL.Query().Get("page_size"); v != "" {
			if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
				pageSize = ps
			}
		}

		users, err := h.client.ListUsers(page, pageSize)
		if err != nil {
			users = &types.UserListResponse{}
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div id="users-table">`))
		_ = templates.UsersTable(users).Render(context.Background(), w)
		w.Write([]byte(`</div>`))
		return
	}

	// Redirect back to users page
	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *Handlers) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	settings, err := h.client.GetSettings()
	if err != nil {
		h.logger.Error("failed to get settings", slog.String("error", err.Error()))
		settings = &types.SystemSettings{} // Use empty settings on error
	}

	data := map[string]interface{}{
		"Title":    "System Settings",
		"User":     user,
		"Settings": settings,
	}

	renderTemplate(w, "settings.templ", data)
}

func (h *Handlers) GetAuthProviders(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	providers, err := h.client.GetAuthProviders()
	if err != nil {
		h.logger.Error("failed to get auth providers", slog.String("error", err.Error()))
		// Return default options if API call fails
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<option value="">Select authentication provider</option>
			<option value="supabase">Supabase</option>
		`))
		return
	}

	// Convert response to HTML options
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<option value="">Select authentication provider</option>`))
	
	if availableProviders, ok := providers["available_providers"].([]interface{}); ok {
		defaultProvider, _ := providers["default_provider"].(string)
		
		for _, provider := range availableProviders {
			if providerStr, ok := provider.(string); ok {
				selected := ""
				if providerStr == defaultProvider {
					selected = " selected"
				}
				w.Write([]byte(fmt.Sprintf(`<option value="%s"%s>%s</option>`, providerStr, selected, providerStr)))
			}
		}
	}
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Parse form values with defaults
	sessionTimeout := 1440
	if timeout := r.FormValue("session_timeout"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			sessionTimeout = t
		}
	}

	minPasswordLength := 8
	if minLen := r.FormValue("min_password_length"); minLen != "" {
		if l, err := strconv.Atoi(minLen); err == nil {
			minPasswordLength = l
		}
	}

	backupRetentionDays := 30
	if retention := r.FormValue("backup_retention_days"); retention != "" {
		if d, err := strconv.Atoi(retention); err == nil {
			backupRetentionDays = d
		}
	}

	// Parse auth provider fields
	availableProviders := r.Form["available_auth_providers"] // Gets all checkbox values
	if len(availableProviders) == 0 {
		// Default to supabase if none selected
		availableProviders = []string{"supabase"}
	}
	
	defaultAuthProvider := r.FormValue("default_auth_provider")
	if defaultAuthProvider == "" {
		defaultAuthProvider = "supabase"
	}

	settings := types.SystemSettings{
		MaintenanceMode:        r.FormValue("maintenance_mode") == "on",
		RegistrationEnabled:    r.FormValue("registration_enabled") == "on",
		EmailNotifications:     r.FormValue("email_notifications") == "on",
		SessionTimeout:         sessionTimeout,
		MinPasswordLength:      minPasswordLength,
		Require2FA:             r.FormValue("require_2fa") == "on",
		AutoBackup:             r.FormValue("auto_backup") == "on",
		BackupRetentionDays:    backupRetentionDays,
		AvailableAuthProviders: availableProviders,
		DefaultAuthProvider:    defaultAuthProvider,
	}

	if err := h.client.UpdateSettings(settings); err != nil {
		h.logger.Error("failed to update settings", slog.String("error", err.Error()))
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusFound)
}

// Template rendering using templ templates
func renderTemplate(w http.ResponseWriter, templateName string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html")

	switch templateName {
	case "login.templ":
		errorMsg, _ := data["Error"].(string)
		err := templates.Login(errorMsg).Render(context.Background(), w)
		if err != nil {
			http.Error(w, "Failed to render login template", http.StatusInternalServerError)
		}
	case "dashboard.templ":
		user, _ := data["User"].(*entities.User)
		stats, _ := data["Stats"].(*types.DashboardStats)
		err := templates.Dashboard(user, stats).Render(context.Background(), w)
		if err != nil {
			http.Error(w, "Failed to render dashboard template", http.StatusInternalServerError)
		}
	case "users.templ":
		user, _ := data["User"].(*entities.User)
		users, _ := data["Users"].(*types.UserListResponse)
		err := templates.Users(user, users).Render(context.Background(), w)
		if err != nil {
			http.Error(w, "Failed to render users template", http.StatusInternalServerError)
		}
	case "settings.templ":
		user, _ := data["User"].(*entities.User)
		settings, _ := data["Settings"].(*types.SystemSettings)
		err := templates.Settings(user, settings).Render(context.Background(), w)
		if err != nil {
			http.Error(w, "Failed to render settings template", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Template not found", http.StatusNotFound)
	}
}

// Cookie helpers
func (h *Handlers) setAuthCookies(w http.ResponseWriter, resp *LoginResponse) {
	maxAge := h.cookieMaxAge
	http.SetCookie(w, &http.Cookie{
		Name:     CookieToken,
		Value:    resp.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CookieUserID,
		Value:    resp.User.ID.String(),
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CookieUserEmail,
		Value:    resp.User.Email,
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CookieAccountType,
		Value:    resp.AccountType,
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CookieExpiresAt,
		Value:    resp.ExpiresAt.Format(time.RFC3339),
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
	})
}

func (h *Handlers) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{CookieToken, CookieUserID, CookieUserEmail, CookieAccountType, CookieExpiresAt} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == CookieToken,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}
}

func getCookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
