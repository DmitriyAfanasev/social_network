// Package postgres содержит PostgreSQL-адаптеры social-сервиса.
package postgres

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/social/internal/ports"
)

// OutboxRepository реализует публикацию social-событий после commit.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository создаёт адаптер social outbox.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// Claim атомарно резервирует пачку ожидающих публикации событий.
func (r *OutboxRepository) Claim(ctx context.Context, limit int, now time.Time) ([]ports.OutboxEvent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	const query = `
		WITH candidates AS (
			SELECT id FROM social.outbox_events
			WHERE published_at IS NULL AND available_at <= $1
			  AND (locked_at IS NULL OR locked_at < $1 - interval '2 minutes')
			ORDER BY created_at, id FOR UPDATE SKIP LOCKED LIMIT $2
		)
		UPDATE social.outbox_events AS events
		SET locked_at = $1, attempts = events.attempts + 1
		FROM candidates WHERE events.id = candidates.id
		RETURNING events.id, events.event_type, events.aggregate_id, events.payload,
			events.correlation_id, events.attempts, events.available_at, events.created_at`
	rows, err := tx.Query(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ports.OutboxEvent, 0, limit)
	for rows.Next() {
		var event ports.OutboxEvent
		var aggregateID *uuid.UUID
		if err := rows.Scan(&event.ID, &event.EventType, &aggregateID, &event.Payload, &event.CorrelationID, &event.Attempts, &event.AvailableAt, &event.CreatedAt); err != nil {
			return nil, err
		}
		event.AggregateID = aggregateID
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// MarkPublished отмечает событие опубликованным.
func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE social.outbox_events SET published_at = $2, locked_at = NULL, last_error = NULL WHERE id = $1 AND published_at IS NULL`, eventID, publishedAt)
	return err
}

// MarkFailed возвращает событие в очередь с новым временем попытки.
func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error {
	_, err := r.pool.Exec(ctx, `UPDATE social.outbox_events SET available_at = $2, locked_at = NULL, last_error = $3 WHERE id = $1 AND published_at IS NULL`, eventID, nextAttempt, reason)
	return err
}
