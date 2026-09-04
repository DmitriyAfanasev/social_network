// Package config содержит конфигурацию Go-сервиса WebRTC signaling.
package config

import (
	platformconfig "general-project/libs/platform/config"
	"log/slog"
	"strings"
	"time"
)

// Config содержит настройки запуска call-signaling.
type Config struct {
	// HTTPAddr — адрес HTTP/WebSocket-сервера signaling.
	HTTPAddr string
	// AllowedOrigins — список origin, которым разрешён WebSocket.
	AllowedOrigins []string
	// JWTSecret должен совпадать с секретом, которым identity подписывает access-токены.
	JWTSecret string
	// RedisAddr — адрес Redis для состояния и межпроцессного Pub/Sub.
	RedisAddr string
	// RedisDB — номер Redis database.
	RedisDB int
	// PostgresURL — адрес PostgreSQL с фактами участников и прав.
	PostgresURL string
	// SessionTTL — время жизни заброшенной call-сессии.
	SessionTTL time.Duration
	// MaxMessageSize — максимальный размер одного signaling-сообщения.
	MaxMessageSize int64
	// LogLevel — уровень структурированного логирования.
	LogLevel slog.Level
}

// Load загружает конфигурацию и применяет безопасные локальные значения по умолчанию.
func Load() Config {
	return Config{
		HTTPAddr:       platformconfig.Env("CALL_SIGNALING_ADDR", ":8001"),
		AllowedOrigins: split(platformconfig.Env("CALL_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		JWTSecret:      platformconfig.Env("CALL_JWT_SECRET", "local-identity-secret"),
		RedisAddr:      platformconfig.Env("CALL_REDIS_ADDR", "localhost:6379"),
		RedisDB:        platformconfig.NonNegativeInt(platformconfig.Env("CALL_REDIS_DB", "0"), 0),
		PostgresURL:    platformconfig.Env("CALL_POSTGRES_URL", "postgres://admin:password@localhost:5432/database"),
		SessionTTL:     platformconfig.PositiveDuration(platformconfig.Env("CALL_SESSION_TTL", "2m"), 2*time.Minute),
		MaxMessageSize: int64(platformconfig.NonNegativeInt(platformconfig.Env("CALL_MAX_MESSAGE_BYTES", "262144"), 262144)),
		LogLevel:       platformconfig.LogLevel(platformconfig.Env("LOG_LEVEL", "INFO")),
	}
}

// split разбирает значения окружения через запятую и убирает пробелы.
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
