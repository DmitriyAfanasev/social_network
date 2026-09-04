// Package config содержит настройки gateway/BFF.
package config

import (
	"log/slog"
	"strings"
	"time"

	platformconfig "general-project/libs/platform/config"
)

// Config содержит настройки маршрутизации и edge-слоя.
type Config struct {
	// HTTPAddr — адрес HTTP-сервера gateway.
	HTTPAddr string
	// IdentityURL — адрес identity-сервиса.
	IdentityURL string
	// ProfilesURL — адрес profiles-сервиса.
	ProfilesURL string
	// SocialURL — адрес social-сервиса.
	SocialURL string
	// ContentURL — адрес content-сервиса.
	ContentURL string
	// MediaURL — адрес media-сервиса.
	MediaURL string
	// MessagingURL — адрес messaging-сервиса.
	MessagingURL string
	// CallsURL — адрес call-signaling-сервиса.
	CallsURL string
	// RedisAddr — адрес Redis для edge rate limit.
	RedisAddr string
	// AllowedOrigins — список origin, которым разрешены browser-запросы.
	AllowedOrigins []string
	// RateLimit — число запросов gateway на один IP в окне RateLimitWindow.
	RateLimit int
	// RateLimitWindow — длительность окна edge rate limit.
	RateLimitWindow time.Duration
	// ReadinessTimeout — таймаут проверки одного upstream-сервиса.
	ReadinessTimeout time.Duration
	// LogLevel — уровень структурированного логирования.
	LogLevel slog.Level
}

// Load загружает настройки gateway из окружения с локальными значениями по умолчанию.
func Load() Config {
	return Config{
		HTTPAddr:         platformconfig.Env("GATEWAY_HTTP_ADDR", ":8000"),
		IdentityURL:      platformconfig.Env("GATEWAY_IDENTITY_URL", "http://localhost:8101"),
		ProfilesURL:      platformconfig.Env("GATEWAY_PROFILES_URL", "http://localhost:8102"),
		SocialURL:        platformconfig.Env("GATEWAY_SOCIAL_URL", "http://localhost:8103"),
		ContentURL:       platformconfig.Env("GATEWAY_CONTENT_URL", "http://localhost:8104"),
		MediaURL:         platformconfig.Env("GATEWAY_MEDIA_URL", "http://localhost:8105"),
		MessagingURL:     platformconfig.Env("GATEWAY_MESSAGING_URL", "http://localhost:8106"),
		CallsURL:         platformconfig.Env("GATEWAY_CALLS_URL", "http://localhost:8001"),
		RedisAddr:        platformconfig.Env("GATEWAY_REDIS_ADDR", "localhost:6379"),
		AllowedOrigins:   split(platformconfig.Env("GATEWAY_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		RateLimit:        platformconfig.PositiveInt(platformconfig.Env("GATEWAY_RATE_LIMIT", "600"), 600),
		RateLimitWindow:  platformconfig.PositiveDuration(platformconfig.Env("GATEWAY_RATE_LIMIT_WINDOW", "1m"), time.Minute),
		ReadinessTimeout: platformconfig.PositiveDuration(platformconfig.Env("GATEWAY_READINESS_TIMEOUT", "2s"), 2*time.Second),
		LogLevel:         platformconfig.LogLevel(platformconfig.Env("LOG_LEVEL", "INFO")),
	}
}

// split разбирает список значений окружения через запятую.
func split(value string) []string {
	values := strings.Split(value, ",")
	result := make([]string, 0, len(values))
	for _, item := range values {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
