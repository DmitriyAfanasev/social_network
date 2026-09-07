// Package config загружает конфигурацию profiles-сервиса.
package config

import platformconfig "general-project/libs/platform/config"

// Config содержит настройки запуска profiles-сервиса.
type Config struct {
	MediaURL      string
	HTTPAddr      string
	DatabaseURL   string
	RedisAddr     string
	JWTSecret     string
	KafkaBrokers  string
	EventsTopic   string
	ConsumerGroup string
	SocialURL     string
}

// Load загружает конфигурацию profiles-сервиса из переменных окружения.
func Load() Config {
	return Config{
		MediaURL:      platformconfig.Env("PROFILES_MEDIA_URL", "http://localhost:8105"),
		HTTPAddr:      platformconfig.Env("PROFILES_HTTP_ADDR", ":8102"),
		DatabaseURL:   platformconfig.Env("PROFILES_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		RedisAddr:     platformconfig.Env("PROFILES_REDIS_ADDR", "localhost:6379"),
		JWTSecret:     platformconfig.Env("PROFILES_JWT_SECRET", "local-identity-secret"),
		KafkaBrokers:  platformconfig.Env("PROFILES_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:   platformconfig.Env("PROFILES_EVENTS_TOPIC", "application.events"),
		ConsumerGroup: platformconfig.Env("PROFILES_KAFKA_GROUP_ID", "profiles"),
		SocialURL:     platformconfig.Env("PROFILES_SOCIAL_URL", "http://localhost:8103"),
	}
}
