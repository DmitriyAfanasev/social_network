package postgres

import (
	"context"
	"errors"
	"time"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/identity/internal/ports"
)

// RefreshTokenStore реализует безопасное хранение refresh-токенов в PostgreSQL.
type RefreshTokenStore struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenStore создаёт адаптер refresh-токенов.
func NewRefreshTokenStore(pool *pgxpool.Pool) *RefreshTokenStore {
	return &RefreshTokenStore{pool: pool}
}

// Save сохраняет только хеш refresh-токена и срок его действия.
func (s *RefreshTokenStore) Save(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO identity.refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`
	_, err := s.pool.Exec(ctx, query, uuid.New(), userID, tokenHash, expiresAt)
	return err
}

// Rotate атомарно отзывает текущий токен и сохраняет его замену.
func (s *RefreshTokenStore) Rotate(ctx context.Context, tokenHash string, replacementHash string, replacementExpiresAt time.Time) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil(), err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const revokeQuery = `
		UPDATE identity.refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id`
	var userID uuid.UUID
	if err := tx.QueryRow(ctx, revokeQuery, tokenHash).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil(), ports.ErrNotFound
	} else if err != nil {
		return uuid.Nil(), err
	}

	const saveQuery = `
		INSERT INTO identity.refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`
	if _, err := tx.Exec(ctx, saveQuery, uuid.New(), userID, replacementHash, replacementExpiresAt); err != nil {
		return uuid.Nil(), err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil(), err
	}
	return userID, nil
}

// Revoke отзывает действующий refresh-токен; повторный logout безопасен.
func (s *RefreshTokenStore) Revoke(ctx context.Context, tokenHash string) error {
	const query = `
		UPDATE identity.refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, query, tokenHash)
	return err
}

// RevokeAll отзывает все активные refresh-токены пользователя.
func (s *RefreshTokenStore) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE identity.refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, query, userID)
	return err
}
