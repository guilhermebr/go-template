package middleware

import (
	"context"
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

func GetUserFromContext(ctx context.Context) (*jwt.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*jwt.Claims)
	return claims, ok
}
