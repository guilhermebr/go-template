package web

import (
	"context"
	"go-template/domain/entities"
	"net/http"
	"time"
)

type contextKey string

const userContextKey contextKey = "user"

// AuthMiddleware handles user authentication for protected routes
type AuthMiddleware struct {
	client       *Client
	cookieSecure bool
	cookieDomain string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(client *Client, cookieSecure bool, cookieDomain string) *AuthMiddleware {
	return &AuthMiddleware{
		client:       client,
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
		user, err := m.client.GetCurrentUser()
		if err != nil {
			// Clear invalid token cookies
			m.clearAuthCookies(w)
			
			http.Redirect(w, r, "/login?error=session_expired&redirect="+r.URL.Path, http.StatusFound)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth middleware that adds user to context if authenticated, but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := getCookieValue(r, CookieToken)
		if token != "" {
			// Set token in client and try to get user
			m.client.SetAuthToken(token)
			user, err := m.client.GetCurrentUser()
			if err == nil && user != nil {
				// Add user to context if valid
				ctx := context.WithValue(r.Context(), userContextKey, user)
				r = r.WithContext(ctx)
			} else {
				// Clear invalid token cookies
				m.clearAuthCookies(w)
			}
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

// clearAuthCookies clears authentication cookies
func (m *AuthMiddleware) clearAuthCookies(w http.ResponseWriter) {
	cookieNames := []string{CookieToken, CookieUserID, CookieUserEmail}
	
	// Don't set domain for localhost in development
	var domain string
	if m.cookieDomain != "localhost" && m.cookieDomain != "" {
		domain = m.cookieDomain
	}
	
	for _, name := range cookieNames {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == CookieToken,
			Secure:   m.cookieSecure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			Domain:   domain,
		})
	}
}