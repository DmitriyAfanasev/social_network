// Package postgres содержит PostgreSQL-адаптеры messaging-сервиса.
package postgres

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/messaging/internal/domain"
	"general-project/messaging/internal/ports"
)

// Repository реализует операции диалогов и сообщений через pgx.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создаёт PostgreSQL-репозиторий messaging-сервиса.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetOrCreateDirect возвращает существующий или создаёт новый прямой диалог.
func (r *Repository) GetOrCreateDirect(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (domain.Conversation, error) {
	key := domain.PairKey(userID, otherUserID)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Conversation{}, err
	}
	defer tx.Rollback(ctx)

	const conversationQuery = `
		INSERT INTO messaging.conversations (id, direct_key)
		VALUES ($1, $2)
		ON CONFLICT (direct_key) DO UPDATE SET updated_at = messaging.conversations.updated_at
		RETURNING id, direct_key, created_at, updated_at`
	var conversation domain.Conversation
	err = tx.QueryRow(ctx, conversationQuery, uuid.New(), key).Scan(&conversation.ID, &conversation.DirectKey, &conversation.CreatedAt, &conversation.UpdatedAt)
	if err != nil {
		return domain.Conversation{}, err
	}
	const participantQuery = `
		INSERT INTO messaging.conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING`
	if _, err := tx.Exec(ctx, participantQuery, conversation.ID, userID, otherUserID); err != nil {
		return domain.Conversation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Conversation{}, err
	}
	conversation.ParticipantIDs = []uuid.UUID{userID, otherUserID}
	return conversation, nil
}

// ListConversations возвращает диалоги, доступные пользователю.
func (r *Repository) ListConversations(ctx context.Context, userID uuid.UUID, archived bool) ([]domain.Conversation, error) {
	const query = `
		SELECT conversation.id, conversation.direct_key, conversation.created_at, conversation.updated_at,
			participant.archived_at IS NOT NULL,
			participant.pinned_at IS NOT NULL,
			participant.muted_at IS NOT NULL,
			participant.cleared_at,
			ARRAY_AGG(all_participants.user_id ORDER BY all_participants.user_id)
		FROM messaging.conversations AS conversation
		JOIN messaging.conversation_participants AS participant
			ON participant.conversation_id = conversation.id
		JOIN messaging.conversation_participants AS all_participants
			ON all_participants.conversation_id = conversation.id
		WHERE participant.user_id = $1 AND participant.hidden_at IS NULL
			AND (($2 AND participant.archived_at IS NOT NULL) OR (NOT $2 AND participant.archived_at IS NULL))
		GROUP BY conversation.id, conversation.direct_key, conversation.created_at, conversation.updated_at,
			participant.archived_at, participant.pinned_at, participant.muted_at, participant.cleared_at
		ORDER BY participant.pinned_at DESC NULLS LAST, conversation.updated_at DESC, conversation.id DESC`
	rows, err := r.pool.Query(ctx, query, userID, archived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]domain.Conversation, 0)
	for rows.Next() {
		var conversation domain.Conversation
		var clearedAt *time.Time
		if err := rows.Scan(&conversation.ID, &conversation.DirectKey, &conversation.CreatedAt, &conversation.UpdatedAt, &conversation.Archived, &conversation.Pinned, &conversation.Muted, &clearedAt, &conversation.ParticipantIDs); err != nil {
			return nil, err
		}
		lastMessage, err := r.lastMessage(ctx, conversation.ID, clearedAt)
		if err != nil {
			return nil, err
		}
		conversation.LastMessage = lastMessage
		conversations = append(conversations, conversation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conversations, nil
}

func (r *Repository) lastMessage(ctx context.Context, conversationID uuid.UUID, clearedAt *time.Time) (*domain.Message, error) {
	const query = `
		SELECT id, conversation_id, sender_id, body, media_id, created_at, edited_at, deleted_at
		FROM messaging.messages
		WHERE conversation_id = $1 AND ($2::timestamptz IS NULL OR created_at > $2)
		ORDER BY created_at DESC, id DESC
		LIMIT 1`
	var message domain.Message
	if err := r.pool.QueryRow(ctx, query, conversationID, clearedAt).Scan(&message.ID, &message.ConversationID, &message.SenderID, &message.Body, &message.MediaID, &message.CreatedAt, &message.EditedAt, &message.DeletedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &message, nil
}

// ListMessages возвращает страницу сообщений участника диалога.
func (r *Repository) ListMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, offset int, limit int) ([]domain.Message, bool, error) {
	if !r.isParticipant(ctx, userID, conversationID) {
		return nil, false, ports.ErrNotFound
	}
	const query = `
		SELECT id, conversation_id, sender_id, body, media_id, created_at, edited_at, deleted_at
		FROM messaging.messages AS message
		JOIN messaging.conversation_participants AS participant
			ON participant.conversation_id = message.conversation_id AND participant.user_id = $1
		WHERE message.conversation_id = $2
			AND (participant.cleared_at IS NULL OR message.created_at > participant.cleared_at)
		ORDER BY message.created_at DESC, message.id DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, query, userID, conversationID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	messages := make([]domain.Message, 0, limit+1)
	for rows.Next() {
		var message domain.Message
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.SenderID, &message.Body, &message.MediaID, &message.CreatedAt, &message.EditedAt, &message.DeletedAt); err != nil {
			return nil, false, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

// CreateMessage сохраняет сообщение только участнику диалога.
func (r *Repository) CreateMessage(ctx context.Context, message domain.Message) (domain.Message, error) {
	const query = `
		INSERT INTO messaging.messages (id, conversation_id, sender_id, body, media_id)
		SELECT $1, $2, $3, $4, $5
		WHERE EXISTS (
			SELECT 1 FROM messaging.conversation_participants
			WHERE conversation_id = $2 AND user_id = $3
		)
		RETURNING id, conversation_id, sender_id, body, media_id, created_at, edited_at, deleted_at`
	var result domain.Message
	err := r.pool.QueryRow(ctx, query, message.ID, message.ConversationID, message.SenderID, message.Body, message.MediaID).Scan(&result.ID, &result.ConversationID, &result.SenderID, &result.Body, &result.MediaID, &result.CreatedAt, &result.EditedAt, &result.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Message{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Message{}, err
	}
	if _, err := r.pool.Exec(ctx, `UPDATE messaging.conversations SET updated_at = now() WHERE id = $1`, message.ConversationID); err != nil {
		return domain.Message{}, err
	}
	return result, nil
}

// UpdateMessage изменяет сообщение его автором.
func (r *Repository) UpdateMessage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID, body string) (domain.Message, error) {
	const query = `
		UPDATE messaging.messages
		SET body = $3, edited_at = now()
		WHERE id = $1 AND sender_id = $2 AND deleted_at IS NULL
		RETURNING id, conversation_id, sender_id, body, media_id, created_at, edited_at, deleted_at`
	var message domain.Message
	err := r.pool.QueryRow(ctx, query, messageID, userID, body).Scan(&message.ID, &message.ConversationID, &message.SenderID, &message.Body, &message.MediaID, &message.CreatedAt, &message.EditedAt, &message.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Message{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Message{}, err
	}
	return message, nil
}

// DeleteMessage помечает сообщение удалённым, сохраняя запись для истории.
func (r *Repository) DeleteMessage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID) error {
	const query = `UPDATE messaging.messages SET deleted_at = now() WHERE id = $1 AND sender_id = $2 AND deleted_at IS NULL`
	result, err := r.pool.Exec(ctx, query, messageID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// RemoveMessageMedia отсоединяет media от сообщения его автора.
func (r *Repository) RemoveMessageMedia(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) (domain.Message, error) {
	const query = `
		UPDATE messaging.messages AS message
		SET media_id = NULL
		WHERE message.id = $1 AND message.conversation_id = $2 AND message.sender_id = $3
			AND message.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM messaging.conversation_participants
				WHERE conversation_id = $2 AND user_id = $3
			)
		RETURNING message.id, message.conversation_id, message.sender_id, message.body,
			message.media_id, message.created_at, message.edited_at, message.deleted_at`
	var result domain.Message
	err := r.pool.QueryRow(ctx, query, messageID, conversationID, userID).Scan(&result.ID, &result.ConversationID, &result.SenderID, &result.Body, &result.MediaID, &result.CreatedAt, &result.EditedAt, &result.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Message{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Message{}, err
	}
	return result, nil
}

// MarkRead записывает последнее прочитанное сообщение участника.
func (r *Repository) MarkRead(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) error {
	const query = `
		UPDATE messaging.conversation_participants AS participant
		SET last_read_message_id = $3, read_at = now()
		FROM messaging.messages AS message
		WHERE participant.conversation_id = $1 AND participant.user_id = $2
			AND message.id = $3 AND message.conversation_id = $1`
	result, err := r.pool.Exec(ctx, query, conversationID, userID, messageID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ArchiveConversation изменяет состояние архивации диалога для участника.
func (r *Repository) ArchiveConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, archived bool) error {
	return r.setParticipantTimestamp(ctx, userID, conversationID, "archived_at", archived)
}

// SetConversationPinned изменяет закрепление диалога для участника.
func (r *Repository) SetConversationPinned(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, pinned bool) error {
	return r.setParticipantTimestamp(ctx, userID, conversationID, "pinned_at", pinned)
}

// SetConversationMuted изменяет отключение уведомлений диалога для участника.
func (r *Repository) SetConversationMuted(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, muted bool) error {
	return r.setParticipantTimestamp(ctx, userID, conversationID, "muted_at", muted)
}

// MarkConversationUnread сбрасывает позицию прочтения диалога.
func (r *Repository) MarkConversationUnread(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	const query = `
		UPDATE messaging.conversation_participants
		SET last_read_message_id = NULL, read_at = NULL
		WHERE conversation_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// HideConversation скрывает диалог у текущего участника.
func (r *Repository) HideConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	return r.setParticipantTimestamp(ctx, userID, conversationID, "hidden_at", true)
}

// ClearConversationHistory скрывает сообщения до момента операции у участника.
func (r *Repository) ClearConversationHistory(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	const query = `
		UPDATE messaging.conversation_participants
		SET cleared_at = now(), last_read_message_id = NULL, read_at = NULL
		WHERE conversation_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *Repository) setParticipantTimestamp(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, column string, enabled bool) error {
	if column != "archived_at" && column != "pinned_at" && column != "muted_at" && column != "hidden_at" {
		return ports.ErrForbidden
	}
	query := `UPDATE messaging.conversation_participants SET ` + column + ` = CASE WHEN $3 THEN now() ELSE NULL END WHERE conversation_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, conversationID, userID, enabled)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ParticipantIDs возвращает участников, если запрашивающий сам состоит в диалоге.
func (r *Repository) ParticipantIDs(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]uuid.UUID, error) {
	if !r.isParticipant(ctx, userID, conversationID) {
		return nil, ports.ErrNotFound
	}
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM messaging.conversation_participants WHERE conversation_id = $1`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]uuid.UUID, 0, 2)
	for rows.Next() {
		var participantID uuid.UUID
		if err := rows.Scan(&participantID); err != nil {
			return nil, err
		}
		users = append(users, participantID)
	}
	return users, rows.Err()
}

func (r *Repository) isParticipant(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) bool {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM messaging.conversation_participants WHERE conversation_id = $1 AND user_id = $2)`, conversationID, userID).Scan(&exists)
	return err == nil && exists
}
