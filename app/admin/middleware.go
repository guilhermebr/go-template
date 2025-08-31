package admin

import (
	"context"
	"go-template/domain/entities"
	gweb "go-template/gateways/web"
	"net/http"

	"github.com/gofrs/uuid/v5"
)

type contextKey string

const userContextKey contextKey = "user"

// AuthMiddleware handles user authentication for protected routes
type AuthMiddleware struct {
	client       *gweb.Client
	cookieSecure bool
	cookieDomain string
	cookieMaxAge int
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(client *gweb.Client, cookieSecure bool, cookieDomain string, cookieMaxAge int) *AuthMiddleware {
	return &AuthMiddleware{
		client:       client,
		cookieMaxAge: cookieMaxAge,
		cookieSecure: cookieSecure,
		cookieDomain: cookieDomain,
	}
}

// RequireAuth middleware that requires user authentication
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getCookieValue(r, CookieToken)
		if token == "" {
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusFound)
			return
		}

		// Set token in client and validate
		m.client.SetAuthToken(token)
		if err := m.client.VerifyToken(); err != nil {
			m.clearAuthCookies(w)
			http.Redirect(w, r, "/login?error=session_expired&redirect="+r.URL.Path, http.StatusFound)
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

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth middleware that adds user to context if authenticated, but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getCookieValue(r, CookieToken)
		if token != "" {
			// Set token in client and try to verify
			m.client.SetAuthToken(token)
			if err := m.client.VerifyToken(); err == nil {
				var user entities.User
				if idStr := getCookieValue(r, CookieUserID); idStr != "" {
					if id, err := uuid.FromString(idStr); err == nil {
						user.ID = id
					}
				}
				user.Email = getCookieValue(r, CookieUserEmail)
				user.AccountType = entities.AccountType(getCookieValue(r, CookieAccountType))
				ctx := context.WithValue(r.Context(), userContextKey, &user)
				r = r.WithContext(ctx)
			} else {
				// Clear invalid token cookies
				m.clearAuthCookies(w)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireSuperAdmin middleware ensures only super admin users can access the route
func (m *AuthMiddleware) RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if user.AccountType != entities.AccountTypeSuperAdmin {
			// For HTMX requests, return error
			if r.Header.Get("HX-Request") == "true" {
				http.Error(w, "Access denied: super admin privileges required", http.StatusForbidden)
				return
			}
			// For regular requests, redirect to dashboard
			http.Redirect(w, r, "/dashboard?error=access_denied", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts the user from the request context
func GetUserFromContext(r *http.Request) *entities.User {
	if user, ok := r.Context().Value(userContextKey).(*entities.User); ok {
		return user
	}
	return nil
}

// IsAuthenticated checks if the current request has an authenticated user
func IsAuthenticated(r *http.Request) bool {
	return GetUserFromContext(r) != nil
}
