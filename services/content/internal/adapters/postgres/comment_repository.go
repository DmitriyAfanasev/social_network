package postgres

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

// CommentRepository реализует операции комментариев через pgx.
type CommentRepository struct {
	pool *pgxpool.Pool
}

// NewCommentRepository создаёт PostgreSQL-адаптер комментариев.
func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

// Create сохраняет комментарий и возвращает его версию из базы данных.
func (r *CommentRepository) Create(ctx context.Context, comment domain.Comment) (domain.Comment, error) {
	const query = `
		INSERT INTO content.comments (id, post_id, author_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, post_id, author_id, body, created_at, updated_at`
	return r.scanOne(ctx, query, comment.ID, comment.PostID, comment.AuthorID, comment.Body)
}

// FindByID загружает комментарий по UUID.
func (r *CommentRepository) FindByID(ctx context.Context, commentID uuid.UUID) (domain.Comment, error) {
	const query = `
		SELECT id, post_id, author_id, body, created_at, updated_at
		FROM content.comments
		WHERE id = $1`
	return r.scanOne(ctx, query, commentID)
}

// ListByPost возвращает комментарии поста в порядке публикации.
func (r *CommentRepository) ListByPost(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Comment, error) {
	const query = `
		SELECT id, post_id, author_id, body, created_at, updated_at
		FROM content.comments
		WHERE post_id = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2`
	rows, err := r.pool.Query(ctx, query, postID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0, limit)
	for rows.Next() {
		var comment domain.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.AuthorID,
			&comment.Body,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

// Delete удаляет комментарий по UUID.
func (r *CommentRepository) Delete(ctx context.Context, commentID uuid.UUID) error {
	const query = `DELETE FROM content.comments WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, commentID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *CommentRepository) scanOne(ctx context.Context, query string, args ...any) (domain.Comment, error) {
	var comment domain.Comment
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.AuthorID,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Comment{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Comment{}, err
	}
	return comment, nil
}
