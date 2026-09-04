// Package readiness содержит проверки готовности обязательных calls-зависимостей.
package readiness

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Checker проверяет PostgreSQL и Redis перед приёмом signaling-соединений.
type Checker struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

// NewChecker создаёт проверку готовности calls-зависимостей.
func NewChecker(pool *pgxpool.Pool, redisClient *redis.Client) *Checker {
	return &Checker{pool: pool, redis: redisClient}
}

// Check возвращает ошибку, если PostgreSQL или Redis недоступны.
func (c *Checker) Check(ctx context.Context) error {
	if err := c.pool.Ping(ctx); err != nil {
		return err
	}
	return c.redis.Ping(ctx).Err()
}
