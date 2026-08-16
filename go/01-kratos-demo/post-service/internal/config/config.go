package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "configs/local.yaml"

type Config struct {
	HTTPAddr string
	MySQLDSN string
}

type fileConfig struct {
	Server struct {
		HTTPAddr string `yaml:"http_addr"`
	} `yaml:"server"`

	MySQL struct {
		DSN string `yaml:"dsn"`
	} `yaml:"mysql"`
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
		HTTPAddr: raw.Server.HTTPAddr,
		MySQLDSN: raw.MySQL.DSN,
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
	if c.MySQLDSN == "" {
		missing = append(missing, "mysql.dsn")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
