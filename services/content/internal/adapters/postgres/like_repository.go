package postgres

import (
	"context"
	"uuid"
)

// LikeRepository реализует операции с лайками постов через pgx.
type LikeRepository struct {
	pool dbPool
}

// NewLikeRepository создаёт PostgreSQL-адаптер лайков.
func NewLikeRepository(pool dbPool) *LikeRepository {
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

// Count возвращает количество лайков поста.
func (r *LikeRepository) Count(ctx context.Context, postID uuid.UUID) (int, error) {
	const query = `SELECT COUNT(*) FROM content.post_likes WHERE post_id = $1`
	var count int
	if err := r.pool.QueryRow(ctx, query, postID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Has проверяет, поставил ли пользователь лайк посту.
func (r *LikeRepository) Has(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM content.post_likes WHERE post_id = $1 AND user_id = $2)`
	var liked bool
	if err := r.pool.QueryRow(ctx, query, postID, userID).Scan(&liked); err != nil {
		return false, err
	}
	return liked, nil
}

// ListUserIDs возвращает пользователей, поставивших лайк, начиная с последних.
func (r *LikeRepository) ListUserIDs(ctx context.Context, postID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
		SELECT user_id
		FROM content.post_likes
		WHERE post_id = $1
		ORDER BY created_at DESC, user_id
	`
	rows, err := r.pool.Query(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return userIDs, nil
}
