// Package config загружает конфигурацию identity-сервиса.
package config

import (
	platformconfig "general-project/libs/platform/config"
	"time"
)

// Config содержит настройки запуска identity-сервиса.
type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	RedisAddr        string
	JWTSecret        string
	SMTPAddr         string
	SMTPFrom         string
	FrontendBaseURL  string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	ConfirmationTTL  time.Duration
	PasswordResetTTL time.Duration
	KafkaBrokers     string
	EventsTopic      string
}

// Load загружает конфигурацию identity-сервиса из переменных окружения.
func Load() Config {
	return Config{
		HTTPAddr:         platformconfig.Env("IDENTITY_HTTP_ADDR", ":8101"),
		DatabaseURL:      platformconfig.Env("IDENTITY_DATABASE_URL", "postgres://admin:password@localhost:5432/database"),
		RedisAddr:        platformconfig.Env("IDENTITY_REDIS_ADDR", "localhost:6379"),
		JWTSecret:        platformconfig.Env("IDENTITY_JWT_SECRET", "local-identity-secret"),
		SMTPAddr:         platformconfig.Env("IDENTITY_SMTP_ADDR", "localhost:1025"),
		SMTPFrom:         platformconfig.Env("IDENTITY_SMTP_FROM", "noreply@localhost"),
		FrontendBaseURL:  platformconfig.Env("IDENTITY_FRONTEND_BASE_URL", "http://localhost:5173"),
		AccessTTL:        platformconfig.SecondsDuration(platformconfig.Env("IDENTITY_ACCESS_TTL", "1800"), 30*time.Minute),
		RefreshTTL:       platformconfig.SecondsDuration(platformconfig.Env("IDENTITY_REFRESH_TTL", "604800"), 7*24*time.Hour),
		ConfirmationTTL:  platformconfig.SecondsDuration(platformconfig.Env("IDENTITY_CONFIRMATION_TTL", "1800"), 30*time.Minute),
		PasswordResetTTL: platformconfig.SecondsDuration(platformconfig.Env("IDENTITY_PASSWORD_RESET_TTL", "600"), 10*time.Minute),
		KafkaBrokers:     platformconfig.Env("IDENTITY_KAFKA_BROKERS", "localhost:9092"),
		EventsTopic:      platformconfig.Env("IDENTITY_EVENTS_TOPIC", "application.events"),
	}
}
