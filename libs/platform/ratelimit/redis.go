// Package ratelimit содержит Redis-backed ограничители частоты запросов.
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"general-project/libs/platform/httpx"
)

// RedisFixedWindow реализует ограничение запросов по фиксированным окнам.
type RedisFixedWindow struct {
	client *redis.Client
}

var fixedWindowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`)

// NewRedisFixedWindow создаёт ограничитель запросов на основе Redis.
func NewRedisFixedWindow(client *redis.Client) *RedisFixedWindow {
	return &RedisFixedWindow{client: client}
}

// Allow проверяет, не превышен ли лимит в текущем временном окне.
func (l *RedisFixedWindow) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit < 1 || window <= 0 {
		return false, fmt.Errorf("invalid rate limit configuration")
	}
	windowID := time.Now().UTC().UnixNano() / window.Nanoseconds()
	redisKey := fmt.Sprintf("rate:%s:%d", key, windowID)
	ttlMilliseconds := window.Milliseconds()
	if ttlMilliseconds < 1 {
		ttlMilliseconds = 1
	}
	count, err := fixedWindowScript.Run(ctx, l.client, []string{redisKey}, ttlMilliseconds).Int64()
	if err != nil {
		return false, err
	}
	return count <= int64(limit), nil
}

// Middleware ограничивает частоту вызовов HTTP-обработчика по вычисляемому ключу.
func Middleware(limiter *RedisFixedWindow, limit int, window time.Duration, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := limiter.Allow(r.Context(), key(r), limit, window)
			if err != nil {
				httpx.WriteError(w, http.StatusServiceUnavailable, "rate_limiter_unavailable", "rate limiter unavailable")
				return
			}
			if !allowed {
				retryAfter := int64(math.Ceil(window.Seconds()))
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
				httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
