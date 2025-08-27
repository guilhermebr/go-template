package web

import (
	"context"
	"go-template/app/web/templates"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	CookieToken    = "web_token"
	CookieUserID   = "web_user_id"
	CookieUserEmail = "web_user_email"
)

// Handlers contains the HTTP handlers for the web application
type Handlers struct {
	client       *Client
	logger       *slog.Logger
	cookieMaxAge int
	cookieSecure bool
	cookieDomain string
}

// NewHandlers creates a new Handlers instance
func NewHandlers(client *Client, logger *slog.Logger, cookieMaxAge int, cookieSecure bool, cookieDomain string) *Handlers {
	return &Handlers{
		client:       client,
		logger:       logger,
		cookieMaxAge: cookieMaxAge,
		cookieSecure: cookieSecure,
		cookieDomain: cookieDomain,
	}
}

// HomePage renders the home/landing page
func (h *Handlers) HomePage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	
	// If user is authenticated, redirect to dashboard
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title": "Welcome to Go Template",
		"User":  user,
	}

	if err := renderTemplate(w, "home.templ", data); err != nil {
		h.logger.Error("failed to render home template", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// LoginPage renders the login page
func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to dashboard or original destination
	if GetUserFromContext(r) != nil {
		redirectTo := r.URL.Query().Get("redirect")
		if redirectTo == "" {
			redirectTo = "/dashboard"
		}
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title":    "Login",
		"Error":    r.URL.Query().Get("error"),
		"Redirect": r.URL.Query().Get("redirect"),
	}

	if err := renderTemplate(w, "login.templ", data); err != nil {
		h.logger.Error("failed to render login template", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// LoginSubmit handles login form submission
func (h *Handlers) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	redirectTo := r.FormValue("redirect")

	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error=missing_credentials", http.StatusSeeOther)
		return
	}

	loginReq := LoginRequest{
		Email:    email,
		Password: password,
	}

	resp, err := h.client.Login(loginReq)
	if err != nil {
		h.logger.Error("login failed", slog.String("error", err.Error()), slog.String("email", email))
		redirectURL := "/login?error=invalid_credentials"
		if redirectTo != "" {
			redirectURL += "&redirect=" + url.QueryEscape(redirectTo)
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	h.logger.Info("login successful", slog.String("email", email), slog.String("user_id", resp.User.ID.String()))

	// Set auth cookies
	h.setAuthCookies(w, resp)

	// Redirect to original destination or dashboard
	if redirectTo == "" {
		redirectTo = "/dashboard"
	}
	h.logger.Info("redirecting after login", slog.String("redirect_to", redirectTo))
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// RegisterPage renders the registration page
func (h *Handlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to dashboard
	if GetUserFromContext(r) != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title": "Register",
		"Error": r.URL.Query().Get("error"),
	}

	if err := renderTemplate(w, "register.templ", data); err != nil {
		h.logger.Error("failed to render register template", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// RegisterSubmit handles registration form submission
func (h *Handlers) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if email == "" || password == "" {
		http.Redirect(w, r, "/register?error=missing_credentials", http.StatusSeeOther)
		return
	}

	if password != confirmPassword {
		http.Redirect(w, r, "/register?error=password_mismatch", http.StatusSeeOther)
		return
	}

	registerReq := RegisterRequest{
		Email:    email,
		Password: password,
	}

	resp, err := h.client.Register(registerReq)
	if err != nil {
		h.logger.Error("registration failed", slog.String("error", err.Error()))
		errorType := "registration_failed"
		if strings.Contains(err.Error(), "409") {
			errorType = "email_exists"
		}
		http.Redirect(w, r, "/register?error="+errorType, http.StatusSeeOther)
		return
	}

	// Set auth cookies
	h.setAuthCookies(w, resp)

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Dashboard renders the user dashboard
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/dashboard", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title": "Dashboard",
		"User":  user,
	}

	if err := renderTemplate(w, "dashboard.templ", data); err != nil {
		h.logger.Error("failed to render dashboard template", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Profile renders the user profile page
func (h *Handlers) Profile(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/profile", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title": "Profile",
		"User":  user,
	}

	if err := renderTemplate(w, "profile.templ", data); err != nil {
		h.logger.Error("failed to render profile template", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Logout handles user logout
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear auth cookies
	h.clearAuthCookies(w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// DocsProxy proxies requests to the API service documentation
func (h *Handlers) DocsProxy(w http.ResponseWriter, r *http.Request) {
	// Extract the path after /docs
	path := chi.URLParam(r, "*")
	if path == "" {
		path = "/"
	}

	resp, err := h.client.ProxyDocsRequest(path)
	if err != nil {
		h.logger.Error("failed to proxy docs request", slog.String("error", err.Error()))
		http.Error(w, "Documentation temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Error("failed to copy response body", slog.String("error", err.Error()))
	}
}

// Cookie management methods

func (h *Handlers) setAuthCookies(w http.ResponseWriter, resp *AuthResponse) {
	maxAge := h.cookieMaxAge
	
	// Don't set domain for localhost in development
	var domain string
	if h.cookieDomain != "localhost" && h.cookieDomain != "" {
		domain = h.cookieDomain
	}
	
	http.SetCookie(w, &http.Cookie{
		Name:     CookieToken,
		Value:    resp.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Domain:   domain,
	})
	
	http.SetCookie(w, &http.Cookie{
		Name:     CookieUserID,
		Value:    resp.User.ID.String(),
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Domain:   domain,
	})
	
	http.SetCookie(w, &http.Cookie{
		Name:     CookieUserEmail,
		Value:    resp.User.Email,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Domain:   domain,
	})
}

func (h *Handlers) clearAuthCookies(w http.ResponseWriter) {
	cookieNames := []string{CookieToken, CookieUserID, CookieUserEmail}
	
	// Don't set domain for localhost in development
	var domain string
	if h.cookieDomain != "localhost" && h.cookieDomain != "" {
		domain = h.cookieDomain
	}
	
	for _, name := range cookieNames {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == CookieToken,
			Secure:   h.cookieSecure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			Domain:   domain,
		})
	}
}

// Utility functions

func getCookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: name == CookieToken,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func renderTemplate(w http.ResponseWriter, templateName string, data map[string]interface{}) error {
	w.Header().Set("Content-Type", "text/html")

	switch templateName {
	case "home.templ":
		user := data["User"]
		return templates.Home(user).Render(context.Background(), w)
	case "login.templ":
		errorMsg, _ := data["Error"].(string)
		redirect, _ := data["Redirect"].(string)
		return templates.Login(errorMsg, redirect).Render(context.Background(), w)
	case "register.templ":
		errorMsg, _ := data["Error"].(string)
		return templates.Register(errorMsg).Render(context.Background(), w)
	case "dashboard.templ":
		user := data["User"]
		return templates.Dashboard(user).Render(context.Background(), w)
	case "profile.templ":
		user := data["User"]
		return templates.Profile(user).Render(context.Background(), w)
	default:
		http.Error(w, "Template not found", http.StatusNotFound)
		return nil
	}
}