package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmzhang/aiagent-eino-backend/internal/config"
)

type httpProvider struct {
	config config.ProviderConfig
	client *http.Client
}

func newHTTPProvider(cfg config.ProviderConfig) httpProvider {
	return httpProvider{config: cfg, client: &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second}}
}

func (p httpProvider) DefaultTemperature() float64 { return p.config.Temperature }

func (p httpProvider) post(ctx context.Context, endpoint string, body []byte, headers http.Header) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建模型请求失败：%w", err)
	}
	request.Header = headers
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("模型服务请求失败：%w", err)
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return response, nil
	}
	defer response.Body.Close()
	content, _ := io.ReadAll(io.LimitReader(response.Body, 32*1024))
	return nil, &RemoteError{StatusCode: response.StatusCode, Message: strings.TrimSpace(string(content))}
}

func endpoint(baseURL, suffix string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return "", fmt.Errorf("模型 base_url 不合法：%w", err)
	}
	if strings.HasSuffix(baseURL, suffix) {
		return baseURL, nil
	}
	return baseURL + suffix, nil
}

type RemoteError struct {
	StatusCode int
	Message    string
}

func (e *RemoteError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("模型服务返回 HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("模型服务返回 HTTP %d：%s", e.StatusCode, e.Message)
}

func IsRemoteError(err error) (*RemoteError, bool) {
	var remote *RemoteError
	if errors.As(err, &remote) {
		return remote, true
	}
	return nil, false
}
