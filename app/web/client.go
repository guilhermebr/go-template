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

// AuthResponse represents the response from login/register endpoints
type AuthResponse struct {
	Token string           `json:"token"`
	User  entities.User    `json:"user"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Client is an HTTP client for communicating with the API service
type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAuthToken sets the authorization token for API requests
func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

// Register registers a new user
func (c *Client) Register(req RegisterRequest) (*AuthResponse, error) {
	var response AuthResponse
	err := c.makeRequest("POST", "/api/v1/auth/register", req, &response)
	return &response, err
}

// Login authenticates a user
func (c *Client) Login(req LoginRequest) (*AuthResponse, error) {
	var response AuthResponse
	err := c.makeRequest("POST", "/api/v1/auth/login", req, &response)
	return &response, err
}

// GetCurrentUser gets the current authenticated user
func (c *Client) GetCurrentUser() (*entities.User, error) {
	var user entities.User
	err := c.makeAuthenticatedRequest("GET", "/api/v1/auth/me", nil, &user)
	return &user, err
}

// makeRequest makes an HTTP request to the API
func (c *Client) makeRequest(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	fullURL := c.baseURL + path
	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// makeAuthenticatedRequest makes an authenticated HTTP request to the API
func (c *Client) makeAuthenticatedRequest(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	fullURL := c.baseURL + path
	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// ProxyDocsRequest proxies a request to the API service docs endpoints
func (c *Client) ProxyDocsRequest(path string) (*http.Response, error) {
	fullURL := c.baseURL + "/docs" + path
	
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	return resp, nil
}

// ValidateToken validates if the current auth token is still valid
func (c *Client) ValidateToken() error {
	_, err := c.GetCurrentUser()
	return err
}