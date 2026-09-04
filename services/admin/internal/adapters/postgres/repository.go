// Package postgres содержит PostgreSQL-адаптеры admin-сервиса.
package postgres

import (
	"context"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/admin/internal/domain"
)

// AuditRepository реализует moderation audit через pgx.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository создаёт PostgreSQL-адаптер moderation audit.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Record сохраняет audit-событие и безопасно переносит повторную доставку.
func (r *AuditRepository) Record(ctx context.Context, log domain.AuditLog) error {
	const query = `
		INSERT INTO admin.moderation_audit_logs
			(id, event_id, actor_id, action, target_type, target_id, details, correlation_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, COALESCE($9, now()))
		ON CONFLICT (event_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, log.ID, log.EventID, log.ActorID, log.Action, log.TargetType, log.TargetID, log.Details, log.CorrelationID, log.CreatedAt)
	return err
}

// List возвращает последние audit-записи в стабильном порядке.
func (r *AuditRepository) List(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	const query = `
		SELECT id, event_id, actor_id, action, target_type, target_id, details, correlation_id, created_at
		FROM admin.moderation_audit_logs
		ORDER BY created_at DESC, id DESC
		LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.AuditLog, 0, limit)
	for rows.Next() {
		var log domain.AuditLog
		var actorID *uuid.UUID
		var targetID *uuid.UUID
		if err := rows.Scan(&log.ID, &log.EventID, &actorID, &log.Action, &log.TargetType, &targetID, &log.Details, &log.CorrelationID, &log.CreatedAt); err != nil {
			return nil, err
		}
		log.ActorID = actorID
		log.TargetID = targetID
		result = append(result, log)
	}
	return result, rows.Err()
}
