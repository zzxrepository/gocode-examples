package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gin-demo/gateway/internal/model"
)

type UserClient struct {
	baseURL string
	client  *http.Client
}

type PostClient struct {
	baseURL string
	client  *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{baseURL: strings.TrimRight(baseURL, "/"), client: defaultHTTPClient()}
}

func NewPostClient(baseURL string) *PostClient {
	return &PostClient{baseURL: strings.TrimRight(baseURL, "/"), client: defaultHTTPClient()}
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func (c *UserClient) Register(ctx context.Context, body io.Reader) (*http.Response, error) {
	return c.forward(ctx, http.MethodPost, "/internal/v1/auth/register", body, nil)
}

func (c *UserClient) Login(ctx context.Context, body io.Reader) (*http.Response, error) {
	return c.forward(ctx, http.MethodPost, "/internal/v1/auth/login", body, nil)
}

func (c *UserClient) ValidateToken(ctx context.Context, authorization string) (*model.TokenClaims, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/auth/validate", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authorization)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var envelope struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    model.TokenClaims `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(envelope.Message)
	}
	return &envelope.Data, nil
}

func (c *UserClient) forward(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.client.Do(req)
}

func (c *PostClient) List(ctx context.Context) (*http.Response, error) {
	return c.forward(ctx, http.MethodGet, "/internal/v1/posts", nil, nil)
}

func (c *PostClient) Get(ctx context.Context, id string) (*http.Response, error) {
	return c.forward(ctx, http.MethodGet, "/internal/v1/posts/"+id, nil, nil)
}

func (c *PostClient) Create(ctx context.Context, userID int64, body []byte) (*http.Response, error) {
	return c.forward(ctx, http.MethodPost, "/internal/v1/posts", bytes.NewReader(body), userHeaders(userID))
}

func (c *PostClient) Update(ctx context.Context, userID int64, id string, body []byte) (*http.Response, error) {
	return c.forward(ctx, http.MethodPut, "/internal/v1/posts/"+id, bytes.NewReader(body), userHeaders(userID))
}

func (c *PostClient) Delete(ctx context.Context, userID int64, id string) (*http.Response, error) {
	return c.forward(ctx, http.MethodDelete, "/internal/v1/posts/"+id, nil, userHeaders(userID))
}

func (c *PostClient) forward(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.client.Do(req)
}

func userHeaders(userID int64) map[string]string {
	return map[string]string{"X-User-ID": fmt.Sprintf("%d", userID)}
}
