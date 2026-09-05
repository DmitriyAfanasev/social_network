// Package config загружает конфигурацию messaging-сервиса.
package config

import (
	platformconfig "general-project/libs/platform/config"
)

const (
	// EventsTopic — общий topic интеграционных событий приложения.
	EventsTopic = "application.events"
	// NotificationConsumerGroup — базовая группа transient notification consumer-а.
	NotificationConsumerGroup = "messaging-notifications"
)

// Config содержит настройки запуска messaging-сервиса.
type Config struct {
	HTTPAddr              string
	DatabaseURL           string
	RedisAddr             string
	JWTSecret             string
	ProfilesURL           string
	SocialURL             string
	RealtimeInflightLimit int
	KafkaBrokers          string
	EventsTopic           string
	KafkaGroupID          string
	KafkaMaxBytes         int
}

// Load загружает конфигурацию messaging-сервиса из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:              platformconfig.Env("MESSAGING_HTTP_ADDR", ":8106"),
		DatabaseURL:           platformconfig.Env("MESSAGING_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		RedisAddr:             platformconfig.Env("MESSAGING_REDIS_ADDR", "localhost:6379"),
		JWTSecret:             platformconfig.Env("MESSAGING_JWT_SECRET", "local-identity-secret"),
		ProfilesURL:           platformconfig.Env("MESSAGING_PROFILES_URL", "http://localhost:8102"),
		SocialURL:             platformconfig.Env("MESSAGING_SOCIAL_URL", "http://localhost:8103"),
		RealtimeInflightLimit: platformconfig.PositiveInt(platformconfig.Env("MESSAGING_REALTIME_INFLIGHT_LIMIT", "1024"), 1024),
		KafkaBrokers:          platformconfig.Env("MESSAGING_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:           platformconfig.Env("MESSAGING_EVENTS_TOPIC", EventsTopic),
		KafkaGroupID:          platformconfig.Env("MESSAGING_NOTIFICATION_CONSUMER_GROUP", NotificationConsumerGroup),
		KafkaMaxBytes:         platformconfig.PositiveInt(platformconfig.Env("MESSAGING_KAFKA_MAX_BYTES", "10000000"), 10_000_000),
	}
}
