// Package config содержит общие функции чтения конфигурации из окружения.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env возвращает значение переменной окружения или fallback, если переменная пуста.
func Env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// PositiveInt разбирает положительное целое или возвращает fallback.
func PositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

// NonNegativeInt разбирает неотрицательное целое или возвращает fallback.
func NonNegativeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

// PositiveDuration разбирает положительную duration или возвращает fallback.
func PositiveDuration(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// NonNegativeDuration разбирает duration, допускающую нулевое значение.
func NonNegativeDuration(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

// SecondsDuration разбирает TTL в целых секундах и возвращает fallback при ошибке.
func SecondsDuration(value string, fallback time.Duration) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

// LogLevel преобразует текстовый уровень логирования в уровень slog.
func LogLevel(value string) slog.Level {
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
