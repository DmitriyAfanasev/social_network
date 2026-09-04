// Package postgres содержит PostgreSQL-адаптеры social-сервиса.
package postgres

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/social/internal/domain"
	"general-project/social/internal/ports"
)

// BlockRepository реализует операции блокировок через pgx.
type BlockRepository struct {
	pool *pgxpool.Pool
}

// NewBlockRepository создаёт адаптер блокировок.
func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{pool: pool}
}

// IsBlocked проверяет блокировку в любом из двух направлений.
func (r *BlockRepository) IsBlocked(ctx context.Context, first uuid.UUID, second uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM social.user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)`
	var blocked bool
	err := r.pool.QueryRow(ctx, query, first, second).Scan(&blocked)
	return blocked, err
}

// Block создаёт блокировку или оставляет существующую без изменений.
func (r *BlockRepository) Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	const query = `INSERT INTO social.user_blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, blockerID, blockedID)
	return err
}

// BlockWithOutbox атомарно создаёт блокировку, очищает отношения и пишет событие.
func (r *BlockRepository) BlockWithOutbox(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, event ports.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	const query = `
		WITH inserted AS (
			INSERT INTO social.user_blocks (blocker_id, blocked_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
			RETURNING blocker_id
		), removed_friendship AS (
			DELETE FROM social.friendships
			WHERE (user_id = LEAST($1, $2) AND friend_id = GREATEST($1, $2))
		), removed_subscriptions AS (
			DELETE FROM social.subscriptions
			WHERE (subscriber_id = $1 AND target_id = $2)
			   OR (subscriber_id = $2 AND target_id = $1)
		), cancelled_requests AS (
			UPDATE social.friend_requests
			SET status = 'cancelled', responded_at = COALESCE(responded_at, $10)
			WHERE status = 'pending'
			  AND ((sender_id = $1 AND recipient_id = $2)
			    OR (sender_id = $2 AND recipient_id = $1))
		)
		INSERT INTO social.outbox_events (id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
		SELECT $3, $4, $5, $6::jsonb, $7, COALESCE($8, now()), COALESCE($9, now())
		FROM inserted`
	if _, err := tx.Exec(ctx, query, blockerID, blockedID, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Unblock удаляет направленную блокировку и сообщает, была ли она найдена.
func (r *BlockRepository) Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error) {
	const query = `DELETE FROM social.user_blocks WHERE blocker_id = $1 AND blocked_id = $2`
	result, err := r.pool.Exec(ctx, query, blockerID, blockedID)
	return result.RowsAffected() > 0, err
}

// FriendshipRepository реализует дружбу и подписки через pgx.
type FriendshipRepository struct {
	pool *pgxpool.Pool
}

// NewFriendshipRepository создаёт адаптер дружбы и подписок.
func NewFriendshipRepository(pool *pgxpool.Pool) *FriendshipRepository {
	return &FriendshipRepository{pool: pool}
}

// IsFriend проверяет наличие нормализованной friendship-записи.
func (r *FriendshipRepository) IsFriend(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) (bool, error) {
	first, second := domain.Pair(userID, friendID)
	const query = `SELECT EXISTS (SELECT 1 FROM social.friendships WHERE user_id = $1 AND friend_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, first, second).Scan(&exists)
	return exists, err
}

// IsSubscribed проверяет направленную подписку пользователя.
func (r *FriendshipRepository) IsSubscribed(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM social.subscriptions WHERE subscriber_id = $1 AND target_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, subscriberID, targetID).Scan(&exists)
	return exists, err
}

// CreateFriendship создаёт friendship-запись или оставляет существующую.
func (r *FriendshipRepository) CreateFriendship(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error {
	const query = `INSERT INTO social.friendships (user_id, friend_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, userID, friendID)
	return err
}

// CreateFriendshipWithOutbox атомарно создаёт friendship и событие после commit.
func (r *FriendshipRepository) CreateFriendshipWithOutbox(ctx context.Context, userID uuid.UUID, friendID uuid.UUID, event ports.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	const query = `
		WITH inserted AS (
			INSERT INTO social.friendships (user_id, friend_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
			RETURNING user_id
		)
		INSERT INTO social.outbox_events (id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
		SELECT $3, $4, $5, $6::jsonb, $7, COALESCE($8, now()), COALESCE($9, now())
		FROM inserted`
	if _, err := tx.Exec(ctx, query, userID, friendID, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RemoveFriendship удаляет нормализованную friendship-запись.
func (r *FriendshipRepository) RemoveFriendship(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) (bool, error) {
	const query = `DELETE FROM social.friendships WHERE user_id = $1 AND friend_id = $2`
	result, err := r.pool.Exec(ctx, query, userID, friendID)
	return result.RowsAffected() > 0, err
}

// Subscribe создаёт направленную подписку.
func (r *FriendshipRepository) Subscribe(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID) error {
	const query = `INSERT INTO social.subscriptions (subscriber_id, target_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, subscriberID, targetID)
	return err
}

// SubscribeWithOutbox атомарно создаёт подписку и событие после commit.
func (r *FriendshipRepository) SubscribeWithOutbox(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID, event ports.OutboxEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	const query = `
		WITH inserted AS (
			INSERT INTO social.subscriptions (subscriber_id, target_id)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
			RETURNING subscriber_id
		)
		INSERT INTO social.outbox_events (id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
		SELECT $3, $4, $5, $6::jsonb, $7, COALESCE($8, now()), COALESCE($9, now())
		FROM inserted`
	if _, err := tx.Exec(ctx, query, subscriberID, targetID, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Unsubscribe удаляет направленную подписку.
func (r *FriendshipRepository) Unsubscribe(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error) {
	const query = `DELETE FROM social.subscriptions WHERE subscriber_id = $1 AND target_id = $2`
	result, err := r.pool.Exec(ctx, query, subscriberID, targetID)
	return result.RowsAffected() > 0, err
}

// ListFriends возвращает UUID друзей пользователя.
func (r *FriendshipRepository) ListFriends(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
		SELECT CASE WHEN user_id = $1 THEN friend_id ELSE user_id END
		FROM social.friendships
		WHERE user_id = $1 OR friend_id = $1
		ORDER BY 1`
	return r.listIDs(ctx, query, userID)
}

// ListSubscribers возвращает UUID пользователей, подписанных на пользователя.
func (r *FriendshipRepository) ListSubscribers(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
		SELECT subscriber_id FROM social.subscriptions
		WHERE target_id = $1 ORDER BY subscriber_id`
	return r.listIDs(ctx, query, userID)
}

// ListSubscriptions возвращает UUID пользователей, на которых подписан пользователь.
func (r *FriendshipRepository) ListSubscriptions(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
		SELECT target_id FROM social.subscriptions
		WHERE subscriber_id = $1 ORDER BY target_id`
	return r.listIDs(ctx, query, userID)
}

// CreateFriendRequest создаёт pending-заявку или возвращает существующую.
func (r *FriendshipRepository) CreateFriendRequest(ctx context.Context, request domain.FriendRequest, event *ports.OutboxEvent) (domain.FriendRequest, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.FriendRequest{}, false, err
	}
	defer tx.Rollback(ctx)

	const insertQuery = `
		INSERT INTO social.friend_requests (id, sender_id, recipient_id, status, created_at)
		VALUES ($1, $2, $3, 'pending', $4)
		ON CONFLICT DO NOTHING
		RETURNING id, sender_id, recipient_id, status, created_at, responded_at`
	created, scanErr := scanFriendRequest(tx.QueryRow(ctx, insertQuery, request.ID, request.SenderID, request.RecipientID, request.CreatedAt))
	if scanErr == nil {
		if event != nil {
			if err := insertOutbox(ctx, tx, *event); err != nil {
				return domain.FriendRequest{}, false, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.FriendRequest{}, false, err
		}
		return created, true, nil
	}
	if !errors.Is(scanErr, pgx.ErrNoRows) {
		return domain.FriendRequest{}, false, scanErr
	}

	const existingQuery = `
		SELECT id, sender_id, recipient_id, status, created_at, responded_at
		FROM social.friend_requests
		WHERE status = 'pending'
		  AND ((sender_id = $1 AND recipient_id = $2)
		    OR (sender_id = $2 AND recipient_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT 1`
	existing, existingErr := scanFriendRequest(tx.QueryRow(ctx, existingQuery, request.SenderID, request.RecipientID))
	if existingErr != nil {
		return domain.FriendRequest{}, false, existingErr
	}
	if existing.SenderID != request.SenderID {
		return domain.FriendRequest{}, false, ports.ErrFriendRequestState
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.FriendRequest{}, false, err
	}
	return existing, false, nil
}

// GetFriendRequest возвращает заявку по UUID.
func (r *FriendshipRepository) GetFriendRequest(ctx context.Context, requestID uuid.UUID) (domain.FriendRequest, error) {
	const query = `
		SELECT id, sender_id, recipient_id, status, created_at, responded_at
		FROM social.friend_requests WHERE id = $1`
	request, err := scanFriendRequest(r.pool.QueryRow(ctx, query, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.FriendRequest{}, ports.ErrFriendRequestNotFound
	}
	return request, err
}

// ListFriendRequests возвращает входящие или исходящие pending-заявки пользователя.
func (r *FriendshipRepository) ListFriendRequests(ctx context.Context, userID uuid.UUID, incoming bool) ([]domain.FriendRequest, error) {
	query := `
		SELECT id, sender_id, recipient_id, status, created_at, responded_at
		FROM social.friend_requests
		WHERE status = 'pending' AND recipient_id = $1
		ORDER BY created_at DESC, id DESC`
	if !incoming {
		query = `
			SELECT id, sender_id, recipient_id, status, created_at, responded_at
			FROM social.friend_requests
			WHERE status = 'pending' AND sender_id = $1
			ORDER BY created_at DESC, id DESC`
	}
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]domain.FriendRequest, 0)
	for rows.Next() {
		request, scanErr := scanFriendRequest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

// TransitionFriendRequest меняет pending-заявку и при необходимости пишет outbox-событие.
func (r *FriendshipRepository) TransitionFriendRequest(ctx context.Context, requestID uuid.UUID, actorID uuid.UUID, status domain.FriendRequestStatus, event *ports.OutboxEvent) (domain.FriendRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.FriendRequest{}, err
	}
	defer tx.Rollback(ctx)

	const lockQuery = `
		SELECT id, sender_id, recipient_id, status, created_at, responded_at
		FROM social.friend_requests WHERE id = $1 FOR UPDATE`
	request, err := scanFriendRequest(tx.QueryRow(ctx, lockQuery, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.FriendRequest{}, ports.ErrFriendRequestNotFound
	}
	if err != nil {
		return domain.FriendRequest{}, err
	}
	if !request.IsPending() {
		return domain.FriendRequest{}, ports.ErrFriendRequestState
	}
	if status == domain.FriendRequestAccepted || status == domain.FriendRequestDeclined {
		if request.RecipientID != actorID {
			return domain.FriendRequest{}, ports.ErrFriendRequestForbidden
		}
	} else if status == domain.FriendRequestCancelled {
		if request.SenderID != actorID {
			return domain.FriendRequest{}, ports.ErrFriendRequestForbidden
		}
	} else {
		return domain.FriendRequest{}, ports.ErrFriendRequestState
	}

	respondedAt := time.Now().UTC()
	const updateQuery = `
		UPDATE social.friend_requests
		SET status = $2, responded_at = $3
		WHERE id = $1
		RETURNING id, sender_id, recipient_id, status, created_at, responded_at`
	updated, err := scanFriendRequest(tx.QueryRow(ctx, updateQuery, requestID, status, respondedAt))
	if err != nil {
		return domain.FriendRequest{}, err
	}
	if status == domain.FriendRequestAccepted {
		first, second := domain.Pair(request.SenderID, request.RecipientID)
		if _, err := tx.Exec(ctx, `INSERT INTO social.friendships (user_id, friend_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, first, second); err != nil {
			return domain.FriendRequest{}, err
		}
	}
	if event != nil {
		if err := insertOutbox(ctx, tx, *event); err != nil {
			return domain.FriendRequest{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.FriendRequest{}, err
	}
	return updated, nil
}

func scanFriendRequest(row interface{ Scan(...any) error }) (domain.FriendRequest, error) {
	var request domain.FriendRequest
	err := row.Scan(&request.ID, &request.SenderID, &request.RecipientID, &request.Status, &request.CreatedAt, &request.RespondedAt)
	return request, err
}

func insertOutbox(ctx context.Context, tx pgx.Tx, event ports.OutboxEvent) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO social.outbox_events (id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, COALESCE($6, now()), COALESCE($7, now()))`,
		event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt)
	return err
}

func (r *FriendshipRepository) listIDs(ctx context.Context, query string, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
