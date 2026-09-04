// Package config загружает настройки admin consumer-а.
package config

import platformconfig "general-project/libs/platform/config"

const (
	// EventsTopic — общий topic интеграционных событий приложения.
	EventsTopic = "application.events"
	// ConsumerGroup — consumer group moderation audit.
	ConsumerGroup = "admin-moderation-audit"
)

// Config содержит настройки запуска admin consumer-а.
type Config struct {
	DatabaseURL   string
	KafkaBrokers  string
	EventsTopic   string
	ConsumerGroup string
}

// Load загружает настройки из переменных окружения.
func Load() Config {
	return Config{
		DatabaseURL:   platformconfig.Env("ADMIN_DATABASE_URL", "postgres://admin:password@localhost:5432/database?sslmode=disable"),
		KafkaBrokers:  platformconfig.Env("ADMIN_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:   platformconfig.Env("ADMIN_EVENTS_TOPIC", EventsTopic),
		ConsumerGroup: platformconfig.Env("ADMIN_CONSUMER_GROUP", ConsumerGroup),
	}
}
