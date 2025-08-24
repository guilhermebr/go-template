package admin

import (
	"go-template/app/api/middleware"
	"go-template/domain/auth"
	"go-template/domain/entities"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/gofrs/uuid/v5"
)

// Request/Response types
type AdminLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AdminLoginResponse struct {
	Token       string        `json:"token"`
	User        entities.User `json:"user"`
	AccountType string        `json:"account_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
}

type DashboardStatsResponse struct {
	TotalUsers     int64 `json:"total_users"`
	AdminUsers     int64 `json:"admin_users"`
	ActiveSessions int64 `json:"active_sessions"`
	SystemAlerts   int64 `json:"system_alerts"`
}

type UserListResponse struct {
	Users      []entities.User `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

type UpdateUserRequest struct {
	Email       string               `json:"email" validate:"email"`
	AccountType entities.AccountType `json:"account_type" validate:"required"`
}

// AdminLogin handles admin login with privilege validation
func (h *AdminHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	slog.Error("AdminLogin")
	var req AdminLoginRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "validation failed: " + err.Error(),
		})
		return
	}

	// Authenticate user using the standard login flow
	response, err := h.authUC.Login(r.Context(), auth.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "authentication failed",
		})
		return
	}

	// Check if user has admin or super admin privileges
	if response.User.AccountType != entities.AccountTypeAdmin && response.User.AccountType != entities.AccountTypeSuperAdmin {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{
			"error": "access denied: admin privileges required",
		})
		return
	}

	// Parse token to get expiration
	claims, err := h.jwtService.ValidateToken(response.Token)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to parse token",
		})
		return
	}

	// Return successful admin login response
	adminResponse := AdminLoginResponse{
		Token:       response.Token,
		User:        response.User,
		AccountType: response.User.AccountType.String(),
		ExpiresAt:   claims.ExpiresAt.Time,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, adminResponse)
}

func (h *AdminHandler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	// For now, logout is client-side (remove token from cookies/localStorage)
	// In the future, we could implement token blacklisting
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{
		"message": "logged out successfully",
	})
}

func (h *AdminHandler) VerifyAdminToken(w http.ResponseWriter, r *http.Request) {
	// This endpoint is protected by middleware, so if we reach here, token is valid
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "invalid token",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"valid":        true,
		"user_id":      claims.UserID,
		"email":        claims.Email,
		"account_type": claims.AccountType,
		"expires_at":   claims.ExpiresAt.Time,
	})
}

func (h *AdminHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual stats retrieval from database
	// For now, return mock data
	stats := DashboardStatsResponse{
		TotalUsers:     0, // TODO: Count from user repository
		AdminUsers:     0, // TODO: Count admin users
		ActiveSessions: 0, // TODO: Count active sessions
		SystemAlerts:   0, // TODO: Count system alerts
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, stats)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	pageSize := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// TODO: Implement user listing from repository with pagination
	// For now, return empty list
	response := UserListResponse{
		Users:      []entities.User{},
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid user ID format",
		})
		return
	}

	user, err := h.userUC.GetUserByID(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{
			"error": "user not found",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, user)
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid user ID format",
		})
		return
	}

	var req UpdateUserRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "validation failed: " + err.Error(),
		})
		return
	}

	// Get current user
	user, err := h.userUC.GetUserByID(r.Context(), userID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{
			"error": "user not found",
		})
		return
	}

	// Update user fields
	if req.Email != "" {
		user.Email = req.Email
	}
	user.AccountType = req.AccountType
	user.UpdatedAt = time.Now()

	// TODO: Implement user update in repository
	// if err := h.userUC.UpdateUser(r.Context(), user); err != nil {
	//     render.Status(r, http.StatusInternalServerError)
	//     render.JSON(w, r, map[string]string{
	//         "error": "failed to update user",
	//     })
	//     return
	// }

	render.Status(r, http.StatusOK)
	render.JSON(w, r, user)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid user ID format",
		})
		return
	}

	// Check that admin is not deleting themselves
	claims, ok := middleware.GetUserFromContext(r.Context())
	if ok && claims.UserID == userID.String() {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "cannot delete your own account",
		})
		return
	}

	// TODO: Implement user deletion in repository
	// if err := h.userUC.DeleteUser(r.Context(), userID); err != nil {
	//     render.Status(r, http.StatusInternalServerError)
	//     render.JSON(w, r, map[string]string{
	//         "error": "failed to delete user",
	//     })
	//     return
	// }

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{
		"message": "user deleted successfully",
	})
}

func (h *AdminHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user statistics
	stats := map[string]interface{}{
		"total_users":      0,
		"admin_users":      0,
		"superadmin_users": 0,
		"regular_users":    0,
		"recent_signups":   0,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, stats)
}

func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement system settings retrieval
	settings := map[string]interface{}{
		"maintenance_mode":     false,
		"registration_enabled": true,
		"email_notifications":  true,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, settings)
}

func (h *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]interface{}
	if err := render.DecodeJSON(r.Body, &settings); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	// TODO: Implement system settings update
	// For now, just return the received settings

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"message":  "settings updated successfully",
		"settings": settings,
	})
}
