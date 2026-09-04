// Package config загружает конфигурацию content-сервиса.
package config

import platformconfig "general-project/libs/platform/config"

// Config содержит настройки запуска content-сервиса.
type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	RedisAddr    string
	JWTSecret    string
	MediaAddr    string
	KafkaBrokers string
	EventsTopic  string
}

// Load загружает конфигурацию content-сервиса из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:     platformconfig.Env("CONTENT_HTTP_ADDR", ":8104"),
		DatabaseURL:  platformconfig.Env("CONTENT_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		RedisAddr:    platformconfig.Env("CONTENT_REDIS_ADDR", "localhost:6379"),
		JWTSecret:    platformconfig.Env("CONTENT_JWT_SECRET", "local-identity-secret"),
		MediaAddr:    platformconfig.Env("CONTENT_MEDIA_ADDR", "http://localhost:8105"),
		KafkaBrokers: platformconfig.Env("CONTENT_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:  platformconfig.Env("CONTENT_EVENTS_TOPIC", "application.events"),
	}
}
