package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Models ModelsConfig `mapstructure:"models"`
}

type ServerConfig struct {
	Address     string   `mapstructure:"address"`
	CORSOrigins []string `mapstructure:"cors_origins"`
}

type ModelsConfig struct {
	Default   string           `mapstructure:"default"`
	Providers []ProviderConfig `mapstructure:"providers"`
}

type ProviderConfig struct {
	ID             string        `mapstructure:"id"`
	Type           string        `mapstructure:"type"`
	BaseURL        string        `mapstructure:"base_url"`
	APIToken       string        `mapstructure:"api_token"`
	TimeoutSeconds int           `mapstructure:"timeout_seconds"`
	Temperature    float64       `mapstructure:"temperature"`
	MaxTokens      int           `mapstructure:"max_tokens"`
	SystemPrompt   string        `mapstructure:"system_prompt"`
	Models         []ModelConfig `mapstructure:"models"`
}

type ModelConfig struct {
	ID    string `mapstructure:"id"`
	Label string `mapstructure:"label"`
}

func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.SetEnvPrefix("AIAGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.cors_origins", []string{"http://localhost:5173"})
	if err := v.BindEnv("server.address", "AIAGENT_SERVER_ADDRESS"); err != nil {
		return Config{}, fmt.Errorf("绑定服务地址环境变量失败：%w", err)
	}
	if err := v.BindEnv("models.default", "AIAGENT_MODELS_DEFAULT"); err != nil {
		return Config{}, fmt.Errorf("绑定默认模型环境变量失败：%w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("读取 config/config.yaml 失败：%w", err)
	}
	// 本地凭据覆盖默认配置，文件已由 .gitignore 排除。
	v.SetConfigFile("./config/local.yaml")
	if err := v.MergeInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("读取 config/local.yaml 失败：%w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("解析配置失败：%w", err)
	}
	overrideProviderTokens(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func overrideProviderTokens(cfg *Config) {
	for i := range cfg.Models.Providers {
		key := "AIAGENT_PROVIDER_" + strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(cfg.Models.Providers[i].ID)) + "_API_TOKEN"
		if token := os.Getenv(key); token != "" {
			cfg.Models.Providers[i].APIToken = token
		}
	}
}

func (cfg Config) Validate() error {
	if cfg.Server.Address == "" {
		return errors.New("server.address 不能为空")
	}
	if cfg.Models.Default == "" || len(cfg.Models.Providers) == 0 {
		return errors.New("models.default 和 models.providers 不能为空")
	}
	seenProviders := make(map[string]bool, len(cfg.Models.Providers))
	seenModels := make(map[string]bool)
	for _, provider := range cfg.Models.Providers {
		if provider.ID == "" || provider.BaseURL == "" || provider.APIToken == "" {
			return fmt.Errorf("provider 的 id、base_url、api_token 不能为空")
		}
		if seenProviders[provider.ID] {
			return fmt.Errorf("provider %q 重复", provider.ID)
		}
		seenProviders[provider.ID] = true
		if provider.Type != "anthropic" && provider.Type != "openai" {
			return fmt.Errorf("provider %q 的 type 仅支持 anthropic 或 openai", provider.ID)
		}
		if provider.TimeoutSeconds <= 0 || provider.MaxTokens <= 0 || len(provider.Models) == 0 {
			return fmt.Errorf("provider %q 的 timeout_seconds、max_tokens 和 models 必须有效", provider.ID)
		}
		for _, model := range provider.Models {
			ref := provider.ID + "/" + model.ID
			if model.ID == "" || seenModels[ref] {
				return fmt.Errorf("模型引用 %q 无效或重复", ref)
			}
			seenModels[ref] = true
		}
	}
	if !seenModels[cfg.Models.Default] {
		return fmt.Errorf("默认模型 %q 未在 providers.models 中登记", cfg.Models.Default)
	}
	return nil
}
