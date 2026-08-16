package provider

import (
	"context"
	"fmt"

	"github.com/mmzhang/aiagent-eino-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-backend/internal/domain"
)

// ChatProvider 是不同厂商模型的统一策略接口。
// 新增厂商时只需实现该接口并注册一个 Factory，无须改动 handler 或 service。
type ChatProvider interface {
	ID() string
	DefaultTemperature() float64
	Stream(ctx context.Context, model string, messages []domain.Message, temperature float64, onDelta func(string) error) error
}

type Factory func(config.ProviderConfig) (ChatProvider, error)

type Target struct {
	Info     domain.ModelInfo
	Provider ChatProvider
}

type Registry struct {
	factories map[string]Factory
	targets   map[string]Target
}

func NewRegistry(configs []config.ProviderConfig) (*Registry, error) {
	registry := &Registry{
		factories: map[string]Factory{
			"anthropic": NewAnthropic,
			"openai":    NewOpenAI,
		},
		targets: make(map[string]Target),
	}
	for _, providerConfig := range configs {
		factory, ok := registry.factories[providerConfig.Type]
		if !ok {
			return nil, fmt.Errorf("没有注册 %q 类型的模型 provider", providerConfig.Type)
		}
		instance, err := factory(providerConfig)
		if err != nil {
			return nil, fmt.Errorf("初始化 provider %q 失败：%w", providerConfig.ID, err)
		}
		for _, model := range providerConfig.Models {
			id := providerConfig.ID + "/" + model.ID
			registry.targets[id] = Target{
				Info:     domain.ModelInfo{ID: id, Provider: providerConfig.ID, Model: model.ID, Label: model.Label},
				Provider: instance,
			}
		}
	}
	return registry, nil
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
