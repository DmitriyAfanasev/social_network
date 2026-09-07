// Package postgres содержит PostgreSQL-адаптеры analytics-сервиса.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker проверяет доступность PostgreSQL через пул соединений.
type HealthChecker struct {
	pool *pgxpool.Pool
}

// NewHealthChecker создаёт адаптер проверки доступности PostgreSQL.
func NewHealthChecker(pool *pgxpool.Pool) *HealthChecker {
	return &HealthChecker{pool: pool}
}

// Check проверяет доступность базы данных в переданном контексте.
func (h *HealthChecker) Check(ctx context.Context) error {
	return h.pool.Ping(ctx)
}
