// Package modelregistry implements the provider Registry + Factory pattern.
// It turns a configured "provider/model" reference into an Eino ChatModel.
package modelregistry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	claude "github.com/cloudwego/eino-ext/components/model/claude"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/domain"
)

// Factory is the adapter boundary for a model provider. Adding a provider only
// requires one Factory and one registration; HTTP and AgentService stay unchanged.
type Factory func(ctx context.Context, provider config.ProviderConfig, modelID string, temperature float64) (model.BaseChatModel, error)

type Target struct {
	Info               domain.ModelInfo
	SystemPrompt       string
	DefaultTemperature float64
	newChatModel       func(context.Context, float64) (model.BaseChatModel, error)
}

func (t Target) NewChatModel(ctx context.Context, temperature float64) (model.BaseChatModel, error) {
	return t.newChatModel(ctx, temperature)
}

type Registry struct {
	factories map[string]Factory
	targets   map[string]Target
}

func New(configs []config.ProviderConfig) (*Registry, error) {
	r := &Registry{
		factories: map[string]Factory{
			"anthropic": newAnthropicChatModel,
			"openai":    newOpenAIChatModel,
		},
		targets: make(map[string]Target),
	}
	for _, provider := range configs {
		factory, ok := r.factories[provider.Type]
		if !ok {
			return nil, fmt.Errorf("没有注册 %q 类型的 Eino ChatModel Factory", provider.Type)
		}
		for _, item := range provider.Models {
			providerCopy, modelID := provider, item.ID
			ref := provider.ID + "/" + modelID
			r.targets[ref] = Target{
				Info:               domain.ModelInfo{ID: ref, Provider: provider.ID, Model: modelID, Label: item.Label},
				SystemPrompt:       provider.SystemPrompt,
				DefaultTemperature: provider.Temperature,
				newChatModel: func(ctx context.Context, temperature float64) (model.BaseChatModel, error) {
					return factory(ctx, providerCopy, modelID, temperature)
				},
			}
		}
	}
	return r, nil
}

func (r *Registry) Resolve(modelRef string) (Target, bool) {
	target, ok := r.targets[modelRef]
	return target, ok
}

func (r *Registry) Models() []domain.ModelInfo {
	models := make([]domain.ModelInfo, 0, len(r.targets))
	for _, target := range r.targets {
		models = append(models, target.Info)
	}
	return models
}

// newAnthropicChatModel also supports providers whose API is Anthropic-compatible,
// such as the configured DashScope endpoint.
func newAnthropicChatModel(ctx context.Context, provider config.ProviderConfig, modelID string, temperature float64) (model.BaseChatModel, error) {
	t := float32(temperature)
	baseURL := provider.BaseURL
	return claude.NewChatModel(ctx, &claude.Config{
		APIKey:         provider.APIToken,
		BaseURL:        &baseURL,
		Model:          modelID,
		MaxTokens:      provider.MaxTokens,
		Temperature:    &t,
		HTTPClient:     &http.Client{Timeout: time.Duration(provider.TimeoutSeconds) * time.Second},
		RequestTimeout: time.Duration(provider.TimeoutSeconds) * time.Second,
	})
}

// newOpenAIChatModel handles OpenAI itself and OpenAI-compatible providers.
func newOpenAIChatModel(ctx context.Context, provider config.ProviderConfig, modelID string, temperature float64) (model.BaseChatModel, error) {
	t := float32(temperature)
	maxTokens := provider.MaxTokens
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      provider.APIToken,
		BaseURL:     provider.BaseURL,
		Model:       modelID,
		MaxTokens:   &maxTokens,
		Temperature: &t,
		Timeout:     time.Duration(provider.TimeoutSeconds) * time.Second,
	})
}
