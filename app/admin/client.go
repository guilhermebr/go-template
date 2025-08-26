package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-template/domain/entities"
	"go-template/internal/types"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

func (c *Client) doRequest(method, endpoint string, body interface{}, result interface{}) error {
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

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
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
		var errorResp map[string]string
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errorMsg, exists := errorResp["error"]; exists {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, errorMsg)
			}
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// Auth endpoints
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token       string        `json:"token"`
	User        entities.User `json:"user"`
	AccountType string        `json:"account_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
}

func (c *Client) AdminLogin(email, password string) (*LoginResponse, error) {
	req := LoginRequest{Email: email, Password: password}
	var resp LoginResponse
	
	if err := c.doRequest("POST", "/admin/v1/login", req, &resp); err != nil {
		return nil, err
	}
	
	return &resp, nil
}

func (c *Client) AdminLogout() error {
	return c.doRequest("POST", "/admin/v1/logout", nil, nil)
}

func (c *Client) VerifyToken() error {
	return c.doRequest("GET", "/admin/v1/verify", nil, nil)
}

// Dashboard endpoints
func (c *Client) GetDashboardStats() (*types.DashboardStats, error) {
	var stats types.DashboardStats
	if err := c.doRequest("GET", "/admin/v1/dashboard/stats", nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// User management endpoints
func (c *Client) ListUsers(page, pageSize int) (*types.UserListResponse, error) {
	endpoint := fmt.Sprintf("/admin/v1/users?page=%d&page_size=%d", page, pageSize)
	var resp types.UserListResponse
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListUsersWithFilter(page, pageSize int, search, accountType string) (*types.UserListResponse, error) {
	endpoint := fmt.Sprintf("/admin/v1/users?page=%d&page_size=%d", page, pageSize)
	
	if search != "" {
		endpoint += fmt.Sprintf("&search=%s", search)
	}
	
	if accountType != "" {
		endpoint += fmt.Sprintf("&account_type=%s", accountType)
	}
	
	var resp types.UserListResponse
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetUser(userID string) (*entities.User, error) {
	var user entities.User
	endpoint := fmt.Sprintf("/admin/v1/users/%s", userID)
	if err := c.doRequest("GET", endpoint, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

type UpdateUserRequest struct {
	Email       string               `json:"email,omitempty"`
	AccountType entities.AccountType `json:"account_type"`
}

type CreateUserRequest struct {
	Email        string               `json:"email" validate:"required,email"`
	Password     string               `json:"password" validate:"required,min=8"`
	AccountType  entities.AccountType `json:"account_type" validate:"required"`
	AuthProvider string               `json:"auth_provider" validate:"required"`
}

func (c *Client) CreateUser(req CreateUserRequest) (*entities.User, error) {
	var user entities.User
	if err := c.doRequest("POST", "/admin/v1/users", req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) UpdateUser(userID string, req UpdateUserRequest) (*entities.User, error) {
	var user entities.User
	endpoint := fmt.Sprintf("/admin/v1/users/%s", userID)
	if err := c.doRequest("PUT", endpoint, req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) DeleteUser(userID string) error {
	endpoint := fmt.Sprintf("/admin/v1/users/%s", userID)
	return c.doRequest("DELETE", endpoint, nil, nil)
}

// System settings endpoints
func (c *Client) GetSettings() (*types.SystemSettings, error) {
	var settings types.SystemSettings
	if err := c.doRequest("GET", "/admin/v1/settings", nil, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (c *Client) UpdateSettings(settings types.SystemSettings) error {
	return c.doRequest("PUT", "/admin/v1/settings", settings, nil)
}

func (c *Client) GetAuthProviders() (map[string]interface{}, error) {
	var response map[string]interface{}
	if err := c.doRequest("GET", "/admin/v1/settings/auth-providers", nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}