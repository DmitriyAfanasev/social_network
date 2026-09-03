/*
Package authz is the PostgreSQL authorization boundary of signaling.

Redis answers whether a short-lived call session exists, but PostgreSQL remains
the source of truth for users, conversations, blocks and privacy settings.
Sensitive operations therefore check PostgreSQL before any event is published;
otherwise a caller with a guessed conversation_id or call_id could bypass the
message privacy rules already enforced by the Python application.
*/
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrForbidden = errors.New("forbidden")

// Repository reads the relational facts needed to authorize a call.
type Repository struct{ pool *pgxpool.Pool }

// NewRepository creates the PostgreSQL-backed authorization adapter.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Participants returns the users belonging to a conversation. An empty room
// is forbidden rather than treated as a public/unknown conversation.
func (r *Repository) Participants(ctx context.Context, conversationID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id FROM conversation_participants
		WHERE conversation_id = $1 ORDER BY user_id`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
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

// CanSubscribe verifies that the authenticated socket belongs to the room.
func (r *Repository) CanSubscribe(ctx context.Context, conversationID, userID int64) error {
	ids, err := r.Participants(ctx, conversationID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == userID {
			return nil
		}
	}
	return ErrForbidden
}

// CanStart verifies direct-room membership and the target's message policy.
// Calls intentionally support the same one-to-one shape as the current UI.
func (r *Repository) CanStart(ctx context.Context, conversationID, callerID, calleeID int64) error {
	if callerID == calleeID {
		return ErrForbidden
	}
	ids, err := r.Participants(ctx, conversationID)
	if err != nil {
		return err
	}
	if !contains(ids, callerID) || !contains(ids, calleeID) {
		return ErrForbidden
	}
	if len(ids) != 2 {
		return fmt.Errorf("call requires a direct conversation")
	}
	return r.canInteract(ctx, callerID, calleeID)
}

// CanUseCall binds a call_id to its conversation and repeats authorization for
// accept/reject/end/signaling. Rechecking handles a block changed mid-call.
func (r *Repository) CanUseCall(ctx context.Context, conversationID, userID int64, callerID, calleeID int64) error {
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

// canInteract mirrors Python's effective message policy: blocks win, then the
// target's message_policy decides whether the relationship is sufficient.
func (r *Repository) canInteract(ctx context.Context, actorID, targetID int64) error {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_blocks
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
		SELECT COALESCE((SELECT message_policy FROM profiles WHERE user_id = $1), 'everyone')`, targetID).Scan(&policy)
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

// areFriends checks the normalized friendship pair used by the SQL model.
func (r *Repository) areFriends(ctx context.Context, left, right int64) (bool, error) {
	var result bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM friendships
			WHERE user_id = LEAST($1, $2) AND friend_id = GREATEST($1, $2)
		)`, left, right).Scan(&result)
	return result, err
}

// areFriendsOfFriends checks whether a friend of actor is connected to target.
func (r *Repository) areFriendsOfFriends(ctx context.Context, actor, target int64) (bool, error) {
	var result bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM friendships first_edge
			JOIN friendships second_edge
			  ON (second_edge.user_id = CASE WHEN first_edge.user_id = $1 THEN first_edge.friend_id ELSE first_edge.user_id END
			      OR second_edge.friend_id = CASE WHEN first_edge.user_id = $1 THEN first_edge.friend_id ELSE first_edge.user_id END)
			WHERE (first_edge.user_id = $1 OR first_edge.friend_id = $1)
			  AND (second_edge.user_id = $2 OR second_edge.friend_id = $2)
		)`, actor, target).Scan(&result)
	return result, err
}

// contains keeps database ids typed as int64 instead of untyped map keys.
func contains(values []int64, wanted int64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
