package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "configs/local.yaml"

type Config struct {
	HTTPAddr       string
	UserServiceURL string
	PostServiceURL string
	RateLimitRPS   float64
	RateLimitBurst int
}

type fileConfig struct {
	Server struct {
		HTTPAddr string `yaml:"http_addr"`
	} `yaml:"server"`

	Service struct {
		UserServiceURL string `yaml:"user_service_url"`
		PostServiceURL string `yaml:"post_service_url"`
	} `yaml:"service"`

	RateLimit struct {
		RPS   float64 `yaml:"rps"`
		Burst int     `yaml:"burst"`
	} `yaml:"rate_limit"`
}

func Load() (Config, error) {
	path := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if path == "" {
		path = defaultConfigPath
	}
	return LoadFile(path)
}

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var raw fileConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg := Config{
		HTTPAddr:       raw.Server.HTTPAddr,
		UserServiceURL: raw.Service.UserServiceURL,
		PostServiceURL: raw.Service.PostServiceURL,
		RateLimitRPS:   raw.RateLimit.RPS,
		RateLimitBurst: raw.RateLimit.Burst,
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	missing := make([]string, 0)
	if c.HTTPAddr == "" {
		missing = append(missing, "server.http_addr")
	}
	if c.UserServiceURL == "" {
		missing = append(missing, "service.user_service_url")
	}
	if c.PostServiceURL == "" {
		missing = append(missing, "service.post_service_url")
	}
	if c.RateLimitRPS <= 0 {
		missing = append(missing, "rate_limit.rps")
	}
	if c.RateLimitBurst <= 0 {
		missing = append(missing, "rate_limit.burst")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
