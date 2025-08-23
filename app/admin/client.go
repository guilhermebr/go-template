package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-template/domain/entities"
	"io"
	"net/http"
	"time"
)

type AdminClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewAdminClient(baseURL string) *AdminClient {
	return &AdminClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AdminClient) SetToken(token string) {
	c.token = token
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

func (c *AdminClient) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/admin/v1/auth/login", bytes.NewReader(body), false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var response LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &response, nil
}

func (c *AdminClient) Logout(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "POST", "/admin/v1/auth/logout", nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// Dashboard stats
type DashboardStats struct {
	TotalUsers     int64 `json:"total_users"`
	AdminUsers     int64 `json:"admin_users"`
	ActiveSessions int64 `json:"active_sessions"`
	SystemAlerts   int64 `json:"system_alerts"`
}

func (c *AdminClient) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	resp, err := c.makeRequest(ctx, "GET", "/admin/v1/dashboard/stats", nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var stats DashboardStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &stats, nil
}

// User management
type UserListResponse struct {
	Users      []entities.User `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

func (c *AdminClient) ListUsers(ctx context.Context, page, pageSize int) (*UserListResponse, error) {
	url := fmt.Sprintf("/admin/v1/users?page=%d&page_size=%d", page, pageSize)
	resp, err := c.makeRequest(ctx, "GET", url, nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var response UserListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &response, nil
}

func (c *AdminClient) GetUser(ctx context.Context, userID string) (*entities.User, error) {
	url := fmt.Sprintf("/admin/v1/users/%s", userID)
	resp, err := c.makeRequest(ctx, "GET", url, nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var user entities.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &user, nil
}

type UpdateUserRequest struct {
	Email       string                `json:"email"`
	AccountType entities.AccountType `json:"account_type"`
}

func (c *AdminClient) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*entities.User, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("/admin/v1/users/%s", userID)
	resp, err := c.makeRequest(ctx, "PUT", url, bytes.NewReader(body), true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var user entities.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &user, nil
}

func (c *AdminClient) DeleteUser(ctx context.Context, userID string) error {
	url := fmt.Sprintf("/admin/v1/users/%s", userID)
	resp, err := c.makeRequest(ctx, "DELETE", url, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// System settings (super admin only)
func (c *AdminClient) GetSettings(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.makeRequest(ctx, "GET", "/admin/v1/settings", nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var settings map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return settings, nil
}

func (c *AdminClient) UpdateSettings(ctx context.Context, settings map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.makeRequest(ctx, "PUT", "/admin/v1/settings", bytes.NewReader(body), true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return response, nil
}

// Helper methods
func (c *AdminClient) makeRequest(ctx context.Context, method, path string, body io.Reader, requireAuth bool) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if requireAuth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	return resp, nil
}

func (c *AdminClient) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error body", resp.StatusCode)
	}

	var errorResp map[string]string
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if msg, ok := errorResp["error"]; ok {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}