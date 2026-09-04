// Package config загружает конфигурацию profiles-сервиса.
package config

import platformconfig "general-project/libs/platform/config"

// Config содержит настройки запуска profiles-сервиса.
type Config struct {
	HTTPAddr    string
	DatabaseURL string
	RedisAddr   string
	JWTSecret   string
}

// Load загружает конфигурацию profiles-сервиса из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:    platformconfig.Env("PROFILES_HTTP_ADDR", ":8102"),
		DatabaseURL: platformconfig.Env("PROFILES_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		RedisAddr:   platformconfig.Env("PROFILES_REDIS_ADDR", "localhost:6379"),
		JWTSecret:   platformconfig.Env("PROFILES_JWT_SECRET", "local-identity-secret"),
	}
}
