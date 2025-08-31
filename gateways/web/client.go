package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-template/domain/entities"
	"io"
	"net/http"
	"time"
)

// Client provides HTTP methods for both public web and admin endpoints.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) SetAuthToken(token string) { c.authToken = token }

// doRequest performs a generic HTTP request with optional auth and JSON (un)marshal.
func (c *Client) doRequest(method, endpoint string, body any, requireAuth bool, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if requireAuth && c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to surface structured error messages if present
		var errorResp map[string]any
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if msg, ok := errorResp["error"].(string); ok {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
			}
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// =========================
// Public Web API
// =========================

type AuthResponse struct {
	Token string        `json:"token"`
	User  entities.User `json:"user"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c *Client) Register(req RegisterRequest) (*AuthResponse, error) {
	var response AuthResponse
	if err := c.doRequest(http.MethodPost, "/api/v1/auth/register", req, false, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) Login(req LoginRequest) (*AuthResponse, error) {
	var response AuthResponse
	if err := c.doRequest(http.MethodPost, "/api/v1/auth/login", req, false, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetCurrentUser() (*entities.User, error) {
	var user entities.User
	if err := c.doRequest(http.MethodGet, "/api/v1/auth/me", nil, true, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) ProxyDocsRequest(path string) (*http.Response, error) {
	fullURL := c.baseURL + "/docs" + path
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	return resp, nil
}

func (c *Client) ValidateToken() error {
	_, err := c.GetCurrentUser()
	return err
}

// =========================
// Admin API
// =========================

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	Token       string        `json:"token"`
	User        entities.User `json:"user"`
	AccountType string        `json:"account_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
}

func (c *Client) AdminLogin(email, password string) (*AdminLoginResponse, error) {
	req := AdminLoginRequest{Email: email, Password: password}
	var resp AdminLoginResponse
	if err := c.doRequest(http.MethodPost, "/admin/v1/login", req, false, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AdminLogout() error {
	return c.doRequest(http.MethodPost, "/admin/v1/logout", nil, true, nil)
}

func (c *Client) VerifyToken() error {
	return c.doRequest(http.MethodGet, "/admin/v1/verify", nil, true, nil)
}

func (c *Client) GetDashboardStats() (*entities.DashboardStats, error) {
	var stats entities.DashboardStats
	if err := c.doRequest(http.MethodGet, "/admin/v1/dashboard/stats", nil, true, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) ListUsers(page, pageSize int) (*entities.UserListResponse, error) {
	endpoint := fmt.Sprintf("/admin/v1/users?page=%d&page_size=%d", page, pageSize)
	var resp entities.UserListResponse
	if err := c.doRequest(http.MethodGet, endpoint, nil, true, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListUsersWithFilter(page, pageSize int, search, accountType string) (*entities.UserListResponse, error) {
	endpoint := fmt.Sprintf("/admin/v1/users?page=%d&page_size=%d", page, pageSize)
	if search != "" {
		endpoint += fmt.Sprintf("&search=%s", search)
	}
	if accountType != "" {
		endpoint += fmt.Sprintf("&account_type=%s", accountType)
	}
	var resp entities.UserListResponse
	if err := c.doRequest(http.MethodGet, endpoint, nil, true, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetUser(userID string) (*entities.User, error) {
	var user entities.User
	endpoint := fmt.Sprintf("/admin/v1/users/%s", userID)
	if err := c.doRequest(http.MethodGet, endpoint, nil, true, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

type UpdateUserRequest struct {
	Email       string               `json:"email,omitempty"`
	AccountType entities.AccountType `json:"account_type"`
}

type CreateUserRequest struct {
	Email        string               `json:"email"`
	Password     string               `json:"password"`
	AccountType  entities.AccountType `json:"account_type"`
	AuthProvider string               `json:"auth_provider"`
}

func (c *Client) CreateUser(req CreateUserRequest) (*entities.User, error) {
	var user entities.User
	if err := c.doRequest(http.MethodPost, "/admin/v1/users", req, true, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) UpdateUser(userID string, req UpdateUserRequest) (*entities.User, error) {
	var user entities.User
	endpoint := fmt.Sprintf("/admin/v1/users/%s", userID)
	if err := c.doRequest(http.MethodPut, endpoint, req, true, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) DeleteUser(userID string) error {
	endpoint := fmt.Sprintf("/admin/v1/users/%s", userID)
	return c.doRequest(http.MethodDelete, endpoint, nil, true, nil)
}

func (c *Client) GetSettings() (*entities.SystemSettings, error) {
	var settings entities.SystemSettings
	if err := c.doRequest(http.MethodGet, "/admin/v1/settings", nil, true, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (c *Client) UpdateSettings(settings entities.SystemSettings) error {
	return c.doRequest(http.MethodPut, "/admin/v1/settings", settings, true, nil)
}

func (c *Client) GetAuthProviders() (map[string]any, error) {
	var response map[string]any
	if err := c.doRequest(http.MethodGet, "/admin/v1/settings/auth-providers", nil, true, &response); err != nil {
		return nil, err
	}
	return response, nil
}
