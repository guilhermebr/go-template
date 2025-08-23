package middleware

import (
	"context"
	"go-template/domain/entities"
	"go-template/internal/jwt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthMiddleware struct {
	jwtService jwt.Service
}

func NewAuthMiddleware(jwtService jwt.Service) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{
				"error": "missing authorization header",
			})
			return
		}

		// Check Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{
				"error": "invalid authorization header format",
			})
			return
		}

		token := parts[1]

		// Validate token
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{
				"error": "invalid token",
			})
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header or cookie
		var token string

		// Try Authorization header first
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}

		// Try cookie if no header
		if token == "" {
			if cookie, err := r.Cookie("admin_token"); err == nil {
				token = cookie.Value
			}
		}

		if token == "" {
			// Redirect to admin login page
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		// Validate token
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			// Redirect to admin login page
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		// Check if user is admin or super admin
		accountType := entities.AccountType(claims.AccountType)
		if accountType != entities.AccountTypeAdmin && accountType != entities.AccountTypeSuperAdmin {
			render.Status(r, http.StatusForbidden)
			render.PlainText(w, r, "Access denied: admin privileges required")
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First check if they're an admin
		m.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				render.Status(r, http.StatusUnauthorized)
				render.PlainText(w, r, "Unauthorized")
				return
			}

			// Check if user is super admin specifically
			if entities.AccountType(claims.AccountType) != entities.AccountTypeSuperAdmin {
				render.Status(r, http.StatusForbidden)
				render.PlainText(w, r, "Access denied: super admin privileges required")
				return
			}

			next.ServeHTTP(w, r)
		})).ServeHTTP(w, r)
	})
}

func GetUserFromContext(ctx context.Context) (*jwt.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*jwt.Claims)
	return claims, ok
}
