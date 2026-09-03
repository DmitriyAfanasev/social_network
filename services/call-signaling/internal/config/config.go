/*
Package config contains configuration for the Go WebRTC signaling service.

The service has its own HTTP listener, Redis client and PostgreSQL pool, but
shares the JWT secret with Python so both processes trust the same login cookie.
Environment-based configuration lets every signaling replica use the same
values without recompiling the binary.
*/
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// HTTPAddr is the address of the signaling HTTP/WebSocket server.
	HTTPAddr string
	// AllowedOrigins limits which browser applications may open the WebSocket.
	AllowedOrigins []string
	// JWTSecret and JWTAlgorithm must match the Python API authentication settings.
	JWTSecret    string
	JWTAlgorithm string
	// Redis stores short-lived call state and transports events between replicas.
	RedisAddr string
	RedisDB   int
	// PostgresURL points to the source of truth for users and permissions.
	PostgresURL string
	// SessionTTL limits how long an abandoned call remains addressable.
	SessionTTL time.Duration
	// MaxMessageSize prevents signaling from becoming an unbounded upload channel.
	MaxMessageSize int64
	LogLevel       slog.Level
}

// Load reads configuration and applies safe defaults for the local Compose stack.
func Load() Config {
	return Config{
		HTTPAddr:       env("CALL_SIGNALING_ADDR", ":8001"),
		AllowedOrigins: split(env("CALL_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		JWTSecret:      env("CALL_JWT_SECRET", env("APP_CONFIG__JWT__SECRET_KEY", "sercretkey")),
		JWTAlgorithm:   strings.ToUpper(env("CALL_JWT_ALGORITHM", env("APP_CONFIG__JWT__ALGORITHM", "HS256"))),
		RedisAddr:      env("CALL_REDIS_ADDR", "localhost:6379"),
		RedisDB:        positiveInt(env("CALL_REDIS_DB", "0"), 0),
		PostgresURL:    env("CALL_POSTGRES_URL", "postgres://admin:password@localhost:5432/database"),
		SessionTTL:     duration(env("CALL_SESSION_TTL", "2m"), 2*time.Minute),
		MaxMessageSize: int64(positiveInt(env("CALL_MAX_MESSAGE_BYTES", "262144"), 262144)),
		LogLevel:       parseLevel(env("LOG_LEVEL", "INFO")),
	}
}

// split parses comma-separated environment values while ignoring whitespace.
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

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// positiveInt prevents invalid limits from accidentally disabling protections.
func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

// duration parses values such as "2m" and rejects non-positive TTLs.
func duration(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// parseLevel maps human-readable log levels to slog levels.
func parseLevel(value string) slog.Level {
	switch strings.ToUpper(value) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
