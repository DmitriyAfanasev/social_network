// Package httpx содержит общие HTTP middleware и формат ответов.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"general-project/libs/platform/correlation"
)

// ClientIP возвращает адрес клиента из TCP-соединения без доверия к
// подменяемым заголовкам прокси.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

// RequestID добавляет запросу идентификатор из заголовка или генерирует новый.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := correlation.WithID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext возвращает идентификатор запроса из контекста.
func RequestIDFromContext(ctx context.Context) string {
	return correlation.ID(ctx)
}

// Recovery преобразует панику обработчика в контролируемый ответ HTTP 500.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(ctx, "http panic recovered", "panic", recovered, "stack", string(debug.Stack()))
					WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}(r.Context())
			next.ServeHTTP(w, r)
		})
	}
}

// AccessLog пишет структурированный журнал завершения HTTP-запроса.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			next.ServeHTTP(w, r)
			logger.InfoContext(r.Context(), "http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(startedAt).String(), "request_id", RequestIDFromContext(r.Context()))
		})
	}
}

// ErrorResponse содержит единый код и сообщение HTTP-ошибки.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteError записывает единый JSON-ответ об ошибке.
func WriteError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Code: code, Message: message}); err != nil {
		return
	}
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(bytes[:])
}
