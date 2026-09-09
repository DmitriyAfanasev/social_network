// Package postgres содержит PostgreSQL-адаптеры analytics-сервиса.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/analytics/internal/domain"
	"general-project/analytics/internal/ports"
)

// OutboxRepository реализует работу с очередью исходящих событий.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository создаёт адаптер outbox для PostgreSQL.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// Claim атомарно резервирует пачку событий для публикации.
func (r *OutboxRepository) Claim(ctx context.Context, limit int, now time.Time) ([]domain.Event, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			return
		}
	}()
	const query = `
		WITH candidates AS (
			SELECT id
			FROM analytics.outbox_events
			WHERE published_at IS NULL
			  AND available_at <= $1
			  AND (locked_at IS NULL OR locked_at < $1 - interval '2 minutes')
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE analytics.outbox_events AS events
		SET locked_at = $1, attempts = events.attempts + 1
		FROM candidates
		WHERE events.id = candidates.id
		RETURNING events.id, events.event_type, events.aggregate_id, events.payload,
			events.correlation_id, events.attempts, events.available_at, events.created_at`
	rows, err := tx.Query(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Event, 0, limit)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
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

// MarkPublished помечает событие успешно опубликованным.
func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error {
	const query = `
		UPDATE analytics.outbox_events
		SET published_at = $2, locked_at = NULL, last_error = NULL
		WHERE id = $1 AND published_at IS NULL`
	_, err := r.pool.Exec(ctx, query, eventID, publishedAt)
	return err
}

// MarkFailed возвращает событие в очередь после неудачной публикации.
func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error {
	const query = `
		UPDATE analytics.outbox_events
		SET available_at = $2, locked_at = NULL, last_error = $3
		WHERE id = $1 AND published_at IS NULL`
	_, err := r.pool.Exec(ctx, query, eventID, nextAttempt, reason)
	return err
}

// ProcessedEventRepository обеспечивает хранение состояния consumer-а.
type ProcessedEventRepository struct {
	pool *pgxpool.Pool
}

// NewProcessedEventRepository создаёт адаптер идемпотентности.
func NewProcessedEventRepository(pool *pgxpool.Pool) *ProcessedEventRepository {
	return &ProcessedEventRepository{pool: pool}
}

// Claim резервирует событие или возвращает false для уже обработанного.
func (r *ProcessedEventRepository) Claim(ctx context.Context, eventID uuid.UUID, now time.Time) (ports.ClaimResult, error) {
	const query = `
		INSERT INTO analytics.processed_events (event_id, status, attempts, claimed_at)
		VALUES ($1, 'processing', 1, $2)
		ON CONFLICT (event_id) DO UPDATE
		SET status = 'processing', attempts = analytics.processed_events.attempts + 1,
			claimed_at = EXCLUDED.claimed_at, processed_at = NULL, last_error = NULL
		WHERE analytics.processed_events.status = 'processing'
		  AND analytics.processed_events.claimed_at < $2 - interval '2 minutes'
		RETURNING event_id, attempts`
	var claimedID uuid.UUID
	var attempts int
	err := r.pool.QueryRow(ctx, query, eventID, now).Scan(&claimedID, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		const statusQuery = `SELECT status, attempts FROM analytics.processed_events WHERE event_id = $1`
		var status string
		if statusErr := r.pool.QueryRow(ctx, statusQuery, eventID).Scan(&status, &attempts); statusErr != nil {
			if errors.Is(statusErr, pgx.ErrNoRows) {
				return ports.ClaimResult{}, nil
			}
			return ports.ClaimResult{}, statusErr
		}
		if status == "processing" {
			return ports.ClaimResult{Attempts: attempts}, ports.ErrInProgress
		}
		return ports.ClaimResult{Attempts: attempts}, nil
	}
	return ports.ClaimResult{Claimed: err == nil, Attempts: attempts}, err
}

// MarkProcessed завершает обработку события.
func (r *ProcessedEventRepository) MarkProcessed(ctx context.Context, eventID uuid.UUID, processedAt time.Time) error {
	const query = `
		UPDATE analytics.processed_events
		SET status = 'processed', processed_at = $2
		WHERE event_id = $1`
	_, err := r.pool.Exec(ctx, query, eventID, processedAt)
	return err
}

// MarkFailed сохраняет ошибку и делает событие доступным для следующей попытки.
func (r *ProcessedEventRepository) MarkFailed(ctx context.Context, eventID uuid.UUID, reason string) error {
	const query = `
		UPDATE analytics.processed_events
		SET claimed_at = now() - interval '2 minutes', last_error = $2
		WHERE event_id = $1 AND status = 'processing'`
	_, err := r.pool.Exec(ctx, query, eventID, reason)
	return err
}

// EventRecordHandler сохраняет успешно принятые события в analytics read model.
type EventRecordHandler struct {
	pool *pgxpool.Pool
}

// NewEventRecordHandler создаёт идемпотентный обработчик журнала событий.
func NewEventRecordHandler(pool *pgxpool.Pool) *EventRecordHandler {
	return &EventRecordHandler{pool: pool}
}

// Handle записывает событие, не создавая дубль при повторной доставке.
func (h *EventRecordHandler) Handle(ctx context.Context, event domain.Event) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			return
		}
	}()

	const query = `
		INSERT INTO analytics.event_records (event_id, event_type, aggregate_id, payload, correlation_id, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		ON CONFLICT (event_id) DO NOTHING`
	result, err := tx.Exec(ctx, query, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.CreatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return nil
	}
	if isVideoEvent(event.EventType) {
		if err := insertVideoEvent(ctx, tx, event); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func isVideoEvent(eventType string) bool {
	switch eventType {
	case "media.video.viewed", "media.video.playback", "media.video.like.created",
		"media.video.like.deleted", "media.video.bookmark.created", "media.video.bookmark.deleted",
		"media.video.favorite.created", "media.video.favorite.deleted":
		return true
	default:
		return false
	}
}

type videoEventPayload struct {
	VideoID         uuid.UUID  `json:"video_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	SessionID       string     `json:"session_id,omitempty"`
	WatchSeconds    float64    `json:"watch_seconds,omitempty"`
	ProgressSeconds float64    `json:"progress_seconds,omitempty"`
	DurationSeconds float64    `json:"duration_seconds,omitempty"`
	Completed       bool       `json:"completed,omitempty"`
}

func insertVideoEvent(ctx context.Context, tx pgx.Tx, event domain.Event) error {
	var payload videoEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.VideoID == uuid.Nil() || payload.WatchSeconds < 0 || payload.ProgressSeconds < 0 || payload.DurationSeconds < 0 {
		return errors.New("invalid video analytics payload")
	}
	const query = `
		INSERT INTO analytics.video_events (
			event_id, event_type, video_id, user_id, session_id,
			watch_seconds, progress_seconds, duration_seconds, completed, occurred_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10)
		ON CONFLICT (event_id) DO NOTHING`
	_, err := tx.Exec(ctx, query, event.ID, event.EventType, payload.VideoID, payload.UserID, payload.SessionID,
		payload.WatchSeconds, payload.ProgressSeconds, payload.DurationSeconds, payload.Completed, event.CreatedAt)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row rowScanner) (domain.Event, error) {
	var event domain.Event
	var aggregateID *uuid.UUID
	if err := row.Scan(&event.ID, &event.EventType, &aggregateID, &event.Payload, &event.CorrelationID, &event.Attempts, &event.AvailableAt, &event.CreatedAt); err != nil {
		return domain.Event{}, err
	}
	if aggregateID != nil {
		event.AggregateID = aggregateID
	}
	return event, nil
}
