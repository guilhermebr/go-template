package admin

import (
	"go-template/app/admin/templates"
	"go-template/app/api/middleware"
	"go-template/domain/entities"
	"go-template/internal/jwt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Handlers struct {
	client     *AdminClient
	jwtService jwt.Service
	authMw     *middleware.AuthMiddleware
	validator  *validator.Validate
}

func NewHandlers(apiBaseURL string, jwtService jwt.Service) *Handlers {
	return &Handlers{
		client:     NewAdminClient(apiBaseURL),
		jwtService: jwtService,
		authMw:     middleware.NewAuthMiddleware(jwtService),
		validator:  validator.New(),
	}
}

func (h *Handlers) Routes() chi.Router {
	r := chi.NewRouter()

	// Public admin routes (login)
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)

	// Static assets
	r.Handle("/static/*", http.StripPrefix("/admin/static", http.FileServer(http.Dir("web/static"))))

	// Protected admin routes
	r.Group(func(r chi.Router) {
		r.Use(h.authMw.RequireAdmin)

		r.Get("/", h.Dashboard)
		r.Get("/dashboard", h.Dashboard)

		// Users management
		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Get("/{id}", h.ViewUser)
			r.Put("/{id}", h.UpdateUser)
			r.Delete("/{id}", h.DeleteUser)
		})

		// Super admin only routes
		r.Group(func(r chi.Router) {
			r.Use(h.authMw.RequireSuperAdmin)
			r.Get("/settings", h.Settings)
		})
	})

	return r
}

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	if cookie, err := r.Cookie("admin_token"); err == nil && cookie.Value != "" {
		if _, err := h.jwtService.ValidateToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
			return
		}
	}

	component := templates.LoginPage()
	component.Render(r.Context(), w)
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		component := templates.LoginError("Invalid request format")
		component.Render(r.Context(), w)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		component := templates.LoginError("Please provide valid email and password")
		component.Render(r.Context(), w)
		return
	}

	// Authenticate user using admin API
	authResp, err := h.client.Login(r.Context(), LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		component := templates.LoginError("Invalid credentials")
		component.Render(r.Context(), w)
		return
	}

	// Set token for subsequent requests
	h.client.SetToken(authResp.Token)

	// Set cookie with JWT token
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    authResp.Token,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// Return success response for HTMX
	w.Header().Set("HX-Redirect", "/admin/dashboard")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the admin token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	// Redirect to login page
	w.Header().Set("HX-Redirect", "/admin/login")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUserFromContext(r.Context())

	// Set token from cookie for API calls
	if cookie, err := r.Cookie("admin_token"); err == nil {
		h.client.SetToken(cookie.Value)
	}

	// Get dashboard stats from API
	stats, err := h.client.GetDashboardStats(r.Context())
	templateStats := templates.DashboardStats{}
	if err == nil {
		templateStats = templates.DashboardStats{
			TotalUsers:     stats.TotalUsers,
			AdminUsers:     stats.AdminUsers,
			ActiveSessions: stats.ActiveSessions,
			SystemAlerts:   stats.SystemAlerts,
		}
	}

	component := templates.Dashboard(templates.DashboardData{
		UserEmail:   claims.Email,
		AccountType: claims.AccountType,
		Stats:       templateStats,
	})
	component.Render(r.Context(), w)
}

func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Set token from cookie for API calls
	if cookie, err := r.Cookie("admin_token"); err == nil {
		h.client.SetToken(cookie.Value)
	}

	// Get users from API
	users, err := h.client.ListUsers(r.Context(), 1, 20)
	if err != nil {
		// Log error and show empty list
		users = &UserListResponse{Users: []entities.User{}}
	}

	component := templates.UsersList(users.Users)
	component.Render(r.Context(), w)
}

func (h *Handlers) ViewUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Set token from cookie for API calls
	if cookie, err := r.Cookie("admin_token"); err == nil {
		h.client.SetToken(cookie.Value)
	}

	// Get user from API
	user, err := h.client.GetUser(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("User not found"))
		return
	}

	component := templates.UserView(*user)
	component.Render(r.Context(), w)
}

func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var req UpdateUserRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request"))
		return
	}

	// Set token from cookie for API calls
	if cookie, err := r.Cookie("admin_token"); err == nil {
		h.client.SetToken(cookie.Value)
	}

	// Update user via API
	user, err := h.client.UpdateUser(r.Context(), userID, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update user"))
		return
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, user)
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Set token from cookie for API calls
	if cookie, err := r.Cookie("admin_token"); err == nil {
		h.client.SetToken(cookie.Value)
	}

	// Delete user via API
	err := h.client.DeleteUser(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete user"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User deleted successfully"))
}

func (h *Handlers) Settings(w http.ResponseWriter, r *http.Request) {
	// Set token from cookie for API calls
	if cookie, err := r.Cookie("admin_token"); err == nil {
		h.client.SetToken(cookie.Value)
	}

	// Get settings from API
	settings, err := h.client.GetSettings(r.Context())
	if err != nil {
		// Log error and show empty settings
		settings = make(map[string]interface{})
	}

	component := templates.Settings(settings)
	component.Render(r.Context(), w)
}
