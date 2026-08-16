package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "configs/local.yaml"

type Config struct {
	HTTPAddr           string
	AuthRateLimitRPS   float64
	AuthRateLimitBurst int
	MySQLDSN           string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	JWTSecret          string
	JWTExpire          time.Duration
}

type fileConfig struct {
	Server struct {
		HTTPAddr string `yaml:"http_addr"`
	} `yaml:"server"`

	RateLimit struct {
		AuthRPS   float64 `yaml:"auth_rps"`
		AuthBurst int     `yaml:"auth_burst"`
	} `yaml:"rate_limit"`

	MySQL struct {
		DSN string `yaml:"dsn"`
	} `yaml:"mysql"`

	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	JWT struct {
		Secret      string `yaml:"secret"`
		ExpireHours int    `yaml:"expire_hours"`
	} `yaml:"jwt"`
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
		HTTPAddr:           raw.Server.HTTPAddr,
		AuthRateLimitRPS:   raw.RateLimit.AuthRPS,
		AuthRateLimitBurst: raw.RateLimit.AuthBurst,
		MySQLDSN:           raw.MySQL.DSN,
		RedisAddr:          raw.Redis.Addr,
		RedisPassword:      raw.Redis.Password,
		RedisDB:            raw.Redis.DB,
		JWTSecret:          raw.JWT.Secret,
		JWTExpire:          time.Duration(raw.JWT.ExpireHours) * time.Hour,
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
	if c.AuthRateLimitRPS <= 0 {
		missing = append(missing, "rate_limit.auth_rps")
	}
	if c.AuthRateLimitBurst <= 0 {
		missing = append(missing, "rate_limit.auth_burst")
	}
	if c.MySQLDSN == "" {
		missing = append(missing, "mysql.dsn")
	}
	if c.RedisAddr == "" {
		missing = append(missing, "redis.addr")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "jwt.secret")
	}
	if c.JWTExpire <= 0 {
		missing = append(missing, "jwt.expire_hours")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
