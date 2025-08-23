package auth

import (
	"go-template/app/api/middleware"
	"go-template/domain/auth"
	"net/http"

	"github.com/go-chi/render"
	"github.com/gofrs/uuid/v5"
)

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body auth.RegisterRequest true "Registration request"
// @Success      201 {object} auth.AuthResponse
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
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

	response, err := h.authUC.Register(r.Context(), req)
	if err != nil {
		// Check for duplicate key error
		if err.Error() == "duplicate key" {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{
				"error": "user already exists",
			})
			return
		}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "registration failed",
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body auth.LoginRequest true "Login request"
// @Success      200 {object} auth.AuthResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
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

	response, err := h.authUC.Login(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "authentication failed",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// GetMe godoc
// @Summary      Get current user
// @Description  Get current user information
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Success      200 {object} entities.User
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/v1/auth/me [get]
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.userUC.GetMe(r.Context(), uuid.FromStringOrNil(claims.UserID))
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
