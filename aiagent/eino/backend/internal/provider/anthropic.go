package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mmzhang/aiagent-eino-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-backend/internal/domain"
)

type anthropicProvider struct{ httpProvider }

func NewAnthropic(cfg config.ProviderConfig) (ChatProvider, error) {
	return &anthropicProvider{httpProvider: newHTTPProvider(cfg)}, nil
}

func (p *anthropicProvider) ID() string { return p.config.ID }

func (p *anthropicProvider) Stream(ctx context.Context, model string, messages []domain.Message, temperature float64, onDelta func(string) error) error {
	body, err := json.Marshal(struct {
		Model       string           `json:"model"`
		Messages    []domain.Message `json:"messages"`
		System      string           `json:"system,omitempty"`
		MaxTokens   int              `json:"max_tokens"`
		Temperature float64          `json:"temperature,omitempty"`
		Stream      bool             `json:"stream"`
	}{model, messages, strings.TrimSpace(p.config.SystemPrompt), p.config.MaxTokens, temperature, true})
	if err != nil {
		return fmt.Errorf("序列化 Anthropic 请求失败：%w", err)
	}
	endpoint, err := endpoint(p.config.BaseURL, "/v1/messages")
	if err != nil {
		return err
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "text/event-stream")
	headers.Set("x-api-key", p.config.APIToken)
	headers.Set("anthropic-version", "2023-06-01")
	response, err := p.post(ctx, endpoint, body, headers)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 16*1024), 2<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			if err := onDelta(event.Delta.Text); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
