// Package postgres содержит PostgreSQL-адаптеры content-сервиса.
package postgres

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

// PostRepository реализует операции текстовых постов через pgx.
type PostRepository struct {
	pool *pgxpool.Pool
}

// NewPostRepository создаёт PostgreSQL-адаптер текстовых постов.
func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{pool: pool}
}

// Create сохраняет новый пост и возвращает его версию из базы данных.
func (r *PostRepository) Create(ctx context.Context, post domain.Post) (domain.Post, error) {
	const query = `
		INSERT INTO content.posts AS post (id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, author_id, body, created_at, updated_at,
			(SELECT COUNT(*) FROM content.comments WHERE post_id = post.id)`
	created, err := r.scanOne(ctx, query, post.ID, post.AuthorID, post.Body)
	if err != nil {
		return domain.Post{}, err
	}
	if err := r.replaceMedia(ctx, created.ID, post.MediaIDs); err != nil {
		_ = r.Delete(ctx, created.ID)
		return domain.Post{}, err
	}
	created.MediaIDs = append([]uuid.UUID(nil), post.MediaIDs...)
	return created, nil
}

// CreateWithOutbox сохраняет пост и событие в одной PostgreSQL-транзакции.
func (r *PostRepository) CreateWithOutbox(ctx context.Context, post domain.Post, event ports.OutboxEvent) (domain.Post, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Post{}, err
	}
	defer tx.Rollback(ctx)
	const query = `
		INSERT INTO content.posts AS post (id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, author_id, body, created_at, updated_at,
			(SELECT COUNT(*) FROM content.comments WHERE post_id = post.id)`
	var created domain.Post
	if err := tx.QueryRow(ctx, query, post.ID, post.AuthorID, post.Body).Scan(
		&created.ID, &created.AuthorID, &created.Body, &created.CreatedAt, &created.UpdatedAt, &created.CommentsCount,
	); err != nil {
		return domain.Post{}, err
	}
	if err := replaceMedia(ctx, tx, created.ID, post.MediaIDs); err != nil {
		return domain.Post{}, err
	}
	const eventQuery = `
		INSERT INTO content.outbox_events (id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, COALESCE($6, now()), COALESCE($7, now()))`
	if _, err := tx.Exec(ctx, eventQuery, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt); err != nil {
		return domain.Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Post{}, err
	}
	created.MediaIDs = append([]uuid.UUID(nil), post.MediaIDs...)
	return created, nil
}

// FindByID загружает пост по UUID.
func (r *PostRepository) FindByID(ctx context.Context, postID uuid.UUID) (domain.Post, error) {
	const query = `
		SELECT id, author_id, body, created_at, updated_at,
			(SELECT COUNT(*) FROM content.comments WHERE post_id = post.id)
		FROM content.posts AS post
		WHERE id = $1`
	post, err := r.scanOne(ctx, query, postID)
	if err != nil {
		return domain.Post{}, err
	}
	post.MediaIDs, err = r.mediaIDs(ctx, post.ID)
	return post, err
}

// ListRecent возвращает последние посты в стабильном порядке.
func (r *PostRepository) ListRecent(ctx context.Context, limit int) ([]domain.Post, error) {
	const query = `
		SELECT id, author_id, body, created_at, updated_at,
			(SELECT COUNT(*) FROM content.comments WHERE post_id = post.id)
		FROM content.posts AS post
		ORDER BY created_at DESC, id DESC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]domain.Post, 0, limit)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		post.MediaIDs, err = r.mediaIDs(ctx, post.ID)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

// Update изменяет текст поста и обновляет время изменения.
func (r *PostRepository) Update(ctx context.Context, postID uuid.UUID, body string, mediaIDs []uuid.UUID) (domain.Post, error) {
	const query = `
		UPDATE content.posts AS post
		SET body = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, author_id, body, created_at, updated_at,
			(SELECT COUNT(*) FROM content.comments WHERE post_id = post.id)`
	post, err := r.scanOne(ctx, query, postID, body)
	if err != nil {
		return domain.Post{}, err
	}
	if err := r.replaceMedia(ctx, postID, mediaIDs); err != nil {
		return domain.Post{}, err
	}
	post.MediaIDs = append([]uuid.UUID(nil), mediaIDs...)
	return post, nil
}

// Delete удаляет пост по UUID.
func (r *PostRepository) Delete(ctx context.Context, postID uuid.UUID) error {
	const query = `DELETE FROM content.posts WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, postID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *PostRepository) scanOne(ctx context.Context, query string, args ...any) (domain.Post, error) {
	var post domain.Post
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&post.ID,
		&post.AuthorID,
		&post.Body,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.CommentsCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Post{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Post{}, err
	}
	return post, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPost(row rowScanner) (domain.Post, error) {
	var post domain.Post
	err := row.Scan(
		&post.ID,
		&post.AuthorID,
		&post.Body,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.CommentsCount,
	)
	return post, err
}

func (r *PostRepository) mediaIDs(ctx context.Context, postID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT media_id FROM content.post_media WHERE post_id = $1 ORDER BY position`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mediaIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var mediaID uuid.UUID
		if err := rows.Scan(&mediaID); err != nil {
			return nil, err
		}
		mediaIDs = append(mediaIDs, mediaID)
	}
	return mediaIDs, rows.Err()
}

func (r *PostRepository) replaceMedia(ctx context.Context, postID uuid.UUID, mediaIDs []uuid.UUID) error {
	return replaceMedia(ctx, r.pool, postID, mediaIDs)
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func replaceMedia(ctx context.Context, executor dbExecutor, postID uuid.UUID, mediaIDs []uuid.UUID) error {
	if _, err := executor.Exec(ctx, `DELETE FROM content.post_media WHERE post_id = $1`, postID); err != nil {
		return err
	}
	for position, mediaID := range mediaIDs {
		if _, err := executor.Exec(ctx, `INSERT INTO content.post_media (post_id, media_id, position) VALUES ($1, $2, $3)`, postID, mediaID, position); err != nil {
			return err
		}
	}
	return nil
}
