package admin

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"go-template/app/admin/templates"
)

func NewRouter(handlers *Handlers, staticPath string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.NoCache)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Static files
	fileServer := http.FileServer(http.Dir(staticPath))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public routes (no auth required)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginSubmit)

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(handlers.RequireAuth)

		r.Get("/dashboard", handlers.Dashboard)
		r.Post("/logout", handlers.Logout)

		// User management
		r.Get("/users", handlers.UsersPage)
		r.Get("/users/{id}", handlers.UserDetail)
		r.Post("/users/create", handlers.CreateUser)
		r.Post("/users/update", handlers.UpdateUser)
		r.Post("/users/delete", handlers.DeleteUser)

		// Settings
		r.Get("/settings", handlers.SettingsPage)
		r.Post("/settings", handlers.UpdateSettings)
		r.Get("/settings/auth-providers", handlers.GetAuthProviders)

		// HTMX/API endpoints for dynamic updates
		r.Route("/api", func(r chi.Router) {
			r.Get("/stats", handlers.GetStatsAPI)
			r.Get("/users", handlers.GetUsersAPI)
			r.Post("/users/{id}/toggle", handlers.ToggleUserAPI)
		})
	})

	return r
}

// Additional API endpoints for HTMX responses
func (h *Handlers) GetStatsAPI(w http.ResponseWriter, r *http.Request) {
	stats, err := h.client.GetDashboardStats()
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	// Return stats as HTML fragment using templ component
	w.Header().Set("Content-Type", "text/html")
	_ = templates.StatsCards(stats).Render(context.Background(), w)
}

func (h *Handlers) GetUsersAPI(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	} else if v := r.URL.Query().Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Get search and filter parameters
	search := r.URL.Query().Get("search")
	accountType := r.URL.Query().Get("account_type")

	users, err := h.client.ListUsersWithFilter(page, pageSize, search, accountType)
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	// Check if this is a request for recent users (dashboard)
	if r.URL.Query().Get("limit") == "5" {
		w.Header().Set("Content-Type", "text/html")
		_ = templates.RecentUsers(users.Users).Render(context.Background(), w)
		return
	}

	// Return users table as HTML fragment using templ component
	w.Header().Set("Content-Type", "text/html")
	_ = templates.UsersTable(users).Render(context.Background(), w)
}

func (h *Handlers) ToggleUserAPI(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id") // userID for future implementation

	// This is a placeholder for user status toggle functionality
	// You would implement the actual toggle logic here

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-green-600">Active</span>`))
}
