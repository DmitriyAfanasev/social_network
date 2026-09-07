// Package config загружает конфигурацию analytics worker-а.
package config

import (
	platformconfig "general-project/libs/platform/config"
	"time"
)

const (
	// EventsTopic — topic интеграционных событий приложения.
	EventsTopic = "application.events"
	// ConsumerGroup — consumer group analytics read model.
	ConsumerGroup = "analytics-read-model"
	// DLQTopic — topic неисправимых analytics-событий.
	DLQTopic = "analytics.dlq"
)

// Config содержит настройки запуска analytics worker-а.
type Config struct {
	// HTTPAddr — адрес HTTP-сервера аналитики.
	HTTPAddr string
	// JWTSecret — секрет проверки access-токенов identity.
	JWTSecret     string
	DatabaseURL   string
	KafkaBrokers  string
	EventsTopic   string
	DLQTopic      string
	ConsumerGroup string
	KafkaMaxBytes int
	PollInterval  time.Duration
	MaxAttempts   int
	RetryBase     time.Duration
}

// Load загружает настройки из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:      platformconfig.Env("ANALYTICS_HTTP_ADDR", ":8108"),
		JWTSecret:     platformconfig.Env("ANALYTICS_JWT_SECRET", "local-identity-secret"),
		DatabaseURL:   platformconfig.Env("ANALYTICS_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		KafkaBrokers:  platformconfig.Env("ANALYTICS_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:   platformconfig.Env("ANALYTICS_EVENTS_TOPIC", EventsTopic),
		DLQTopic:      platformconfig.Env("ANALYTICS_DLQ_TOPIC", DLQTopic),
		ConsumerGroup: platformconfig.Env("ANALYTICS_CONSUMER_GROUP", ConsumerGroup),
		KafkaMaxBytes: platformconfig.PositiveInt(platformconfig.Env("ANALYTICS_KAFKA_MAX_BYTES", "10000000"), 10_000_000),
		PollInterval:  platformconfig.PositiveDuration(platformconfig.Env("ANALYTICS_POLL_INTERVAL", "1s"), time.Second),
		MaxAttempts:   platformconfig.PositiveInt(platformconfig.Env("ANALYTICS_MAX_ATTEMPTS", "3"), 3),
		RetryBase:     platformconfig.NonNegativeDuration(platformconfig.Env("ANALYTICS_RETRY_BASE", "1s"), time.Second),
	}
}
