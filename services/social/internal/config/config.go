// Package config загружает конфигурацию social-сервиса.
package config

import platformconfig "general-project/libs/platform/config"

// Config содержит настройки запуска social-сервиса.
type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	RedisAddr    string
	JWTSecret    string
	KafkaBrokers string
	EventsTopic  string
	ProfilesURL  string
}

// Load загружает конфигурацию social-сервиса из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:     platformconfig.Env("SOCIAL_HTTP_ADDR", ":8103"),
		DatabaseURL:  platformconfig.Env("SOCIAL_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		RedisAddr:    platformconfig.Env("SOCIAL_REDIS_ADDR", "localhost:6379"),
		JWTSecret:    platformconfig.Env("SOCIAL_JWT_SECRET", "local-identity-secret"),
		KafkaBrokers: platformconfig.Env("SOCIAL_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:  platformconfig.Env("SOCIAL_EVENTS_TOPIC", "application.events"),
		ProfilesURL:  platformconfig.Env("SOCIAL_PROFILES_URL", "http://localhost:8102"),
	}
}
