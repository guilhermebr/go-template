package admin

import (
	"go-template/app/api/middleware"
	"go-template/domain/auth"
	"go-template/domain/entities"
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

type CreateUserRequest struct {
	Email        string               `json:"email" validate:"required,email"`
	Password     string               `json:"password" validate:"required,min=8"`
	AccountType  entities.AccountType `json:"account_type" validate:"required"`
	AuthProvider string               `json:"auth_provider" validate:"required"`
}

type UpdateUserRequest struct {
	Email       string               `json:"email" validate:"email"`
	AccountType entities.AccountType `json:"account_type" validate:"required"`
}

// AdminLogin handles admin login with privilege validation
func (h *AdminHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
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
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{
		"message": "logged out successfully",
	})
}

func (h *AdminHandler) VerifyAdminToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "missing authorization header",
		})
		return
	}

	// Expected format: "Bearer <token>"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "invalid authorization header format",
		})
		return
	}

	token := authHeader[7:] // Remove "Bearer " prefix

	// Validate token using JWT service
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "invalid token",
		})
		return
	}

	// Check if user has admin privileges
	accountType := entities.AccountType(claims.AccountType)
	if accountType != entities.AccountTypeAdmin && accountType != entities.AccountTypeSuperAdmin {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{
			"error": "insufficient privileges",
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
	userStats, err := h.userUC.GetUserStats(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to get user stats",
		})
		return
	}

	stats := DashboardStatsResponse{
		TotalUsers:     userStats.TotalUsers,
		AdminUsers:     userStats.AdminUsers + userStats.SuperAdminUsers,
		ActiveSessions: 0, // TODO: Implement session tracking
		SystemAlerts:   0, // TODO: Implement system alerts
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, stats)
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
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

	user, err := h.userUC.CreateUser(r.Context(), req.Email, req.Password, req.AuthProvider, req.AccountType)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to create user",
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, user)
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

	// Parse search and filter parameters
	search := r.URL.Query().Get("search")
	accountType := r.URL.Query().Get("account_type")

	var users []entities.User
	var total int64
	var err error

	// Use search if provided, otherwise regular listing
	if search != "" || accountType != "" {
		users, total, err = h.userUC.SearchUsers(r.Context(), page, pageSize, search, accountType)
	} else {
		users, total, err = h.userUC.ListUsers(r.Context(), page, pageSize)
	}

	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to list users",
		})
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := UserListResponse{
		Users:      users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
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

	if err := h.userUC.UpdateUser(r.Context(), user); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to update user",
		})
		return
	}

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

	if err := h.userUC.DeleteUser(r.Context(), userID); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to delete user",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{
		"message": "user deleted successfully",
	})
}

func (h *AdminHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userStats, err := h.userUC.GetUserStats(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to get user stats",
		})
		return
	}

	stats := map[string]interface{}{
		"total_users":      userStats.TotalUsers,
		"admin_users":      userStats.AdminUsers,
		"superadmin_users": userStats.SuperAdminUsers,
		"regular_users":    userStats.RegularUsers,
		"recent_signups":   userStats.RecentSignups,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, stats)
}

func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsUC.GetSettings(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to get settings",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, settings)
}

func (h *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settingsRequest entities.SystemSettings
	if err := render.DecodeJSON(r.Body, &settingsRequest); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.settingsUC.UpdateSettings(r.Context(), &settingsRequest); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to update settings",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{
		"message": "settings updated successfully",
	})
}

func (h *AdminHandler) GetAvailableAuthProviders(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsUC.GetSettings(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "failed to get settings",
		})
		return
	}

	response := map[string]any{
		"available_providers": settings.AvailableAuthProviders,
		"default_provider":   settings.DefaultAuthProvider,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}
