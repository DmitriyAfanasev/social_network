// Package authz содержит PostgreSQL-адаптер проверки прав call signaling.
package authz

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrForbidden означает, что пользователь не имеет права на действие.
var ErrForbidden = errors.New("forbidden")

// Repository читает факты, необходимые для авторизации звонка.
type Repository struct{ pool *pgxpool.Pool }

// NewRepository создаёт PostgreSQL-адаптер авторизации.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Participants возвращает пользователей, входящих в диалог.
func (r *Repository) Participants(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id FROM messaging.conversation_participants
		WHERE conversation_id = $1 ORDER BY user_id`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0, 2)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrForbidden
	}
	return ids, nil
}

// CanSubscribe проверяет, что аутентифицированный пользователь входит в диалог.
func (r *Repository) CanSubscribe(ctx context.Context, conversationID, userID uuid.UUID) error {
	ids, err := r.Participants(ctx, conversationID)
	if err != nil {
		return err
	}
	if contains(ids, userID) {
		return nil
	}
	return ErrForbidden
}

// CanStart проверяет прямой диалог и политику взаимодействия с адресатом.
func (r *Repository) CanStart(ctx context.Context, conversationID, callerID, calleeID uuid.UUID) error {
	if callerID == uuid.Nil() || calleeID == uuid.Nil() || callerID == calleeID {
		return ErrForbidden
	}
	ids, err := r.Participants(ctx, conversationID)
	if err != nil {
		return err
	}
	if len(ids) != 2 || !contains(ids, callerID) || !contains(ids, calleeID) {
		return ErrForbidden
	}
	return r.canInteract(ctx, callerID, calleeID)
}

// CanUseCall повторно проверяет права при каждом действии звонка.
func (r *Repository) CanUseCall(ctx context.Context, conversationID, userID, callerID, calleeID uuid.UUID) error {
	if err := r.CanSubscribe(ctx, conversationID, userID); err != nil {
		return err
	}
	ids, err := r.Participants(ctx, conversationID)
	if err != nil {
		return err
	}
	if len(ids) != 2 || !contains(ids, callerID) || !contains(ids, calleeID) {
		return ErrForbidden
	}
	return r.canInteract(ctx, callerID, calleeID)
}

// canInteract применяет блокировки и message_policy профиля адресата.
func (r *Repository) canInteract(ctx context.Context, actorID, targetID uuid.UUID) error {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM social.user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)`, actorID, targetID).Scan(&blocked)
	if err != nil {
		return err
	}
	if blocked {
		return ErrForbidden
	}

	var policy string
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT message_policy FROM profiles.profiles WHERE user_id = $1), 'everyone')`, targetID).Scan(&policy)
	if err != nil {
		return err
	}
	if policy == "everyone" {
		return nil
	}

	friends, err := r.areFriends(ctx, actorID, targetID)
	if err != nil {
		return err
	}
	if policy == "friends" && friends {
		return nil
	}
	if policy == "friends_of_friends" {
		friendsOfFriends, queryErr := r.areFriendsOfFriends(ctx, actorID, targetID)
		if queryErr != nil {
			return queryErr
		}
		if friends || friendsOfFriends {
			return nil
		}
	}
	return ErrForbidden
}

// areFriends проверяет нормализованную пару дружбы из SQL-модели.
func (r *Repository) areFriends(ctx context.Context, left, right uuid.UUID) (bool, error) {
	var result bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM social.friendships
			WHERE user_id = LEAST($1, $2) AND friend_id = GREATEST($1, $2)
		)`, left, right).Scan(&result)
	return result, err
}

// areFriendsOfFriends проверяет связь через общего друга.
func (r *Repository) areFriendsOfFriends(ctx context.Context, actor, target uuid.UUID) (bool, error) {
	var result bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM social.friendships first_edge
			JOIN social.friendships second_edge
			  ON (second_edge.user_id = CASE WHEN first_edge.user_id = $1 THEN first_edge.friend_id ELSE first_edge.user_id END
			      OR second_edge.friend_id = CASE WHEN first_edge.user_id = $1 THEN first_edge.friend_id ELSE first_edge.user_id END)
			WHERE (first_edge.user_id = $1 OR first_edge.friend_id = $1)
			  AND (second_edge.user_id = $2 OR second_edge.friend_id = $2)
		)`, actor, target).Scan(&result)
	return result, err
}

func contains(values []uuid.UUID, wanted uuid.UUID) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
