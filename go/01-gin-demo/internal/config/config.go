package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	MySQLDSN      string
	HTTPAddr      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	JWTExpire     time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		MySQLDSN:      getEnv("MYSQL_DSN", "blog:blog123@tcp(127.0.0.1:3306)/blog_demo?parseTime=true&loc=Local"),
		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-local-dev"),
		JWTExpire:     time.Duration(getEnvInt("JWT_EXPIRE_HOURS", 24)) * time.Hour,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
