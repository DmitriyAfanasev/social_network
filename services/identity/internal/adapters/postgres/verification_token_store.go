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

// VerificationTokenStore реализует одноразовое хранение verification-токенов в PostgreSQL.
type VerificationTokenStore struct {
	pool *pgxpool.Pool
}

// NewVerificationTokenStore создаёт PostgreSQL-адаптер verification-токенов.
func NewVerificationTokenStore(pool *pgxpool.Pool) *VerificationTokenStore {
	return &VerificationTokenStore{pool: pool}
}

// Save сохраняет только хеш токена и его назначение.
func (s *VerificationTokenStore) Save(ctx context.Context, userID uuid.UUID, purpose string, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO identity.verification_tokens (id, user_id, purpose, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := s.pool.Exec(ctx, query, uuid.New(), userID, purpose, tokenHash, expiresAt)
	return err
}

// Consume атомарно проверяет срок действия и помечает токен использованным.
func (s *VerificationTokenStore) Consume(ctx context.Context, purpose string, tokenHash string) (uuid.UUID, error) {
	const query = `
		UPDATE identity.verification_tokens
		SET consumed_at = now()
		WHERE purpose = $1 AND token_hash = $2 AND consumed_at IS NULL AND expires_at > now()
		RETURNING user_id`
	var userID uuid.UUID
	if err := s.pool.QueryRow(ctx, query, purpose, tokenHash).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil(), ports.ErrNotFound
	} else if err != nil {
		return uuid.Nil(), err
	}
	return userID, nil
}
