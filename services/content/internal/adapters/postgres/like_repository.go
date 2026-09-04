package postgres

import (
	"context"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LikeRepository реализует операции с лайками постов через pgx.
type LikeRepository struct {
	pool *pgxpool.Pool
}

// NewLikeRepository создаёт PostgreSQL-адаптер лайков.
func NewLikeRepository(pool *pgxpool.Pool) *LikeRepository {
	return &LikeRepository{pool: pool}
}

// Add добавляет лайк и повторный вызов оставляет состояние идемпотентным.
func (r *LikeRepository) Add(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
	const query = `
		INSERT INTO content.post_likes (post_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (post_id, user_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, postID, userID)
	return err
}

// Remove удаляет лайк и сообщает, существовал ли он.
func (r *LikeRepository) Remove(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error) {
	const query = `DELETE FROM content.post_likes WHERE post_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, postID, userID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}
