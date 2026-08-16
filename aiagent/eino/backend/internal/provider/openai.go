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

type openAIProvider struct{ httpProvider }

func NewOpenAI(cfg config.ProviderConfig) (ChatProvider, error) {
	return &openAIProvider{httpProvider: newHTTPProvider(cfg)}, nil
}

func (p *openAIProvider) ID() string { return p.config.ID }

func (p *openAIProvider) Stream(ctx context.Context, model string, messages []domain.Message, temperature float64, onDelta func(string) error) error {
	allMessages := make([]domain.Message, 0, len(messages)+1)
	if prompt := strings.TrimSpace(p.config.SystemPrompt); prompt != "" {
		allMessages = append(allMessages, domain.Message{Role: "system", Content: prompt})
	}
	allMessages = append(allMessages, messages...)
	body, err := json.Marshal(struct {
		Model       string           `json:"model"`
		Messages    []domain.Message `json:"messages"`
		Temperature float64          `json:"temperature,omitempty"`
		Stream      bool             `json:"stream"`
	}{model, allMessages, temperature, true})
	if err != nil {
		return fmt.Errorf("序列化 OpenAI 请求失败：%w", err)
	}
	endpoint, err := endpoint(p.config.BaseURL, "/chat/completions")
	if err != nil {
		return err
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "text/event-stream")
	headers.Set("Authorization", "Bearer "+p.config.APIToken)
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
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil || len(event.Choices) == 0 {
			continue
		}
		if delta := event.Choices[0].Delta.Content; delta != "" {
			if err := onDelta(delta); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}
