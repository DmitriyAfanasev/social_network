package ports

import (
	"context"
	"errors"
	"time"
	"uuid"

	"general-project/messaging/internal/domain"
)

var (
	// ErrNotFound означает, что диалог или сообщение недоступны.
	ErrNotFound = errors.New("messaging resource not found")
	// ErrForbidden означает, что пользователь не имеет доступа к ресурсу.
	ErrForbidden = errors.New("messaging access forbidden")
)

// MessagingRepository предоставляет application-слою доступ к диалогам и сообщениям.
type MessagingRepository interface {
	GetOrCreateDirect(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (domain.Conversation, error)
	ListConversations(ctx context.Context, userID uuid.UUID, archived bool) ([]domain.Conversation, error)
	ListMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, offset int, limit int) ([]domain.Message, bool, error)
	CreateMessage(ctx context.Context, message domain.Message) (domain.Message, error)
	UpdateMessage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID, body string) (domain.Message, error)
	DeleteMessage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID) error
	RemoveMessageMedia(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) (domain.Message, error)
	MarkRead(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) error
	ArchiveConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, archived bool) error
	SetConversationPinned(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, pinned bool) error
	SetConversationMuted(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, muted bool) error
	MarkConversationUnread(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	HideConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	ClearConversationHistory(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	ParticipantIDs(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]uuid.UUID, error)
}

// MessagePolicyReader проверяет, может ли пользователь начать диалог с целью.
type MessagePolicyReader interface {
	CanMessage(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (bool, error)
}

// MessageCache хранит горячие read-модели сообщений и диалогов.
type MessageCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// ReadinessChecker проверяет готовность messaging-сервиса.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}

// AccessTokenVerifier извлекает UUID пользователя из access-токена.
type AccessTokenVerifier interface {
	UserID(token string) (uuid.UUID, error)
}
