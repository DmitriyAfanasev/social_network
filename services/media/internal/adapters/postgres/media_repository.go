// Package postgres содержит PostgreSQL-адаптеры media-сервиса.
package postgres

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

// MediaRepository реализует операции метаданных медиа через pgx.
type MediaRepository struct {
	pool *pgxpool.Pool
}

// NewMediaRepository создаёт PostgreSQL-адаптер метаданных медиа.
func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{pool: pool}
}

// Create сохраняет метаданные медиаобъекта.
func (r *MediaRepository) Create(ctx context.Context, media domain.Media) (domain.Media, error) {
	const query = `
		INSERT INTO media.media (
			id, object_key, bucket, original_filename, content_type, media_type,
			size, checksum, width, height, duration, uploaded_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, object_key, bucket, original_filename, content_type, media_type,
			size, checksum, width, height, duration, uploaded_by, created_at, deleted_at`
	result, err := r.scanOne(ctx, query,
		media.ID, media.ObjectKey, media.Bucket, media.OriginalFilename,
		media.ContentType, media.MediaType, media.Size, media.Checksum,
		media.Width, media.Height, media.Duration, media.UploadedBy,
	)
	if isUniqueViolation(err) {
		return domain.Media{}, ports.ErrAlreadyExists
	}
	return result, err
}

// CreateWithOutbox сохраняет media-метаданные и событие в одной транзакции.
func (r *MediaRepository) CreateWithOutbox(ctx context.Context, media domain.Media, event ports.OutboxEvent) (domain.Media, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Media{}, err
	}
	defer tx.Rollback(ctx)
	const query = `
		INSERT INTO media.media (
			id, object_key, bucket, original_filename, content_type, media_type,
			size, checksum, width, height, duration, uploaded_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, object_key, bucket, original_filename, content_type, media_type,
			size, checksum, width, height, duration, uploaded_by, created_at, deleted_at`
	var created domain.Media
	if err := tx.QueryRow(ctx, query,
		media.ID, media.ObjectKey, media.Bucket, media.OriginalFilename,
		media.ContentType, media.MediaType, media.Size, media.Checksum,
		media.Width, media.Height, media.Duration, media.UploadedBy,
	).Scan(
		&created.ID, &created.ObjectKey, &created.Bucket, &created.OriginalFilename,
		&created.ContentType, &created.MediaType, &created.Size, &created.Checksum,
		&created.Width, &created.Height, &created.Duration, &created.UploadedBy,
		&created.CreatedAt, &created.DeletedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return domain.Media{}, ports.ErrAlreadyExists
		}
		return domain.Media{}, err
	}
	const eventQuery = `
		INSERT INTO media.outbox_events (id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, COALESCE($6, now()), COALESCE($7, now()))`
	if _, err := tx.Exec(ctx, eventQuery, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt); err != nil {
		return domain.Media{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Media{}, err
	}
	return created, nil
}

// FindByID загружает только активные метаданные медиаобъекта.
func (r *MediaRepository) FindByID(ctx context.Context, mediaID uuid.UUID) (domain.Media, error) {
	const query = `
		SELECT id, object_key, bucket, original_filename, content_type, media_type,
			size, checksum, width, height, duration, uploaded_by, created_at, deleted_at
		FROM media.media
		WHERE id = $1 AND deleted_at IS NULL`
	return r.scanOne(ctx, query, mediaID)
}

// UpdateProcessedImage обновляет метаданные заменённого WebP-изображения.
func (r *MediaRepository) UpdateProcessedImage(ctx context.Context, mediaID uuid.UUID, contentType string, size int64, checksum string, width int, height int) error {
	const query = `
		UPDATE media.media
		SET content_type = $2, size = $3, checksum = $4, width = $5, height = $6
		WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.pool.Exec(ctx, query, mediaID, contentType, size, checksum, width, height)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// Delete помечает метаданные медиаобъекта временем удаления.
func (r *MediaRepository) Delete(ctx context.Context, mediaID uuid.UUID, deletedAt time.Time) error {
	const query = `
		UPDATE media.media
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.pool.Exec(ctx, query, mediaID, deletedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *MediaRepository) scanOne(ctx context.Context, query string, args ...any) (domain.Media, error) {
	var media domain.Media
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&media.ID,
		&media.ObjectKey,
		&media.Bucket,
		&media.OriginalFilename,
		&media.ContentType,
		&media.MediaType,
		&media.Size,
		&media.Checksum,
		&media.Width,
		&media.Height,
		&media.Duration,
		&media.UploadedBy,
		&media.CreatedAt,
		&media.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Media{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Media{}, err
	}
	return media, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
