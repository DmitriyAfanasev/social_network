// Package ports содержит интерфейсы, которыми владеет application-слой.
package ports

import (
	"context"
	"errors"
	"time"

	"uuid"

	"general-project/social/internal/domain"
)

var (
	// ErrFriendRequestNotFound сообщает, что заявка не найдена.
	ErrFriendRequestNotFound = errors.New("friend request not found")
	// ErrFriendRequestForbidden сообщает, что пользователь не может изменить заявку.
	ErrFriendRequestForbidden = errors.New("friend request operation forbidden")
	// ErrFriendRequestState сообщает, что переход из текущего состояния недопустим.
	ErrFriendRequestState = errors.New("friend request state conflict")
)

// BlockRepository управляет направленными блокировками пользователей.
type BlockRepository interface {
	IsBlocked(ctx context.Context, first uuid.UUID, second uuid.UUID) (bool, error)
	Block(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error
	BlockWithOutbox(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, event OutboxEvent) error
	Unblock(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID) (bool, error)
}

// FriendshipRepository управляет дружбой и подписками пользователей.
type FriendshipRepository interface {
	IsFriend(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) (bool, error)
	IsSubscribed(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error)
	CreateFriendship(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error
	CreateFriendshipWithOutbox(ctx context.Context, userID uuid.UUID, friendID uuid.UUID, event OutboxEvent) error
	RemoveFriendship(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) (bool, error)
	Subscribe(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID) error
	SubscribeWithOutbox(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID, event OutboxEvent) error
	Unsubscribe(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error)
	ListFriends(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	ListSubscribers(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	ListSubscriptions(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// CreateFriendRequest создаёт заявку и при наличии события пишет его в outbox.
	CreateFriendRequest(ctx context.Context, request domain.FriendRequest, event *OutboxEvent) (domain.FriendRequest, bool, error)
	// GetFriendRequest возвращает заявку по её идентификатору.
	GetFriendRequest(ctx context.Context, requestID uuid.UUID) (domain.FriendRequest, error)
	// ListFriendRequests возвращает pending-заявки отправителя или получателя.
	ListFriendRequests(ctx context.Context, userID uuid.UUID, incoming bool) ([]domain.FriendRequest, error)
	// TransitionFriendRequest выполняет допустимый переход состояния заявки.
	TransitionFriendRequest(ctx context.Context, requestID uuid.UUID, actorID uuid.UUID, status domain.FriendRequestStatus, event *OutboxEvent) (domain.FriendRequest, error)
}

// OutboxEvent описывает social-событие, ожидающее публикации после commit.
type OutboxEvent struct {
	ID            uuid.UUID
	EventType     string
	AggregateID   *uuid.UUID
	Payload       []byte
	CorrelationID string
	Attempts      int
	AvailableAt   time.Time
	CreatedAt     time.Time
}

// OutboxRepository предоставляет application-слою доступ к social outbox.
type OutboxRepository interface {
	Claim(ctx context.Context, limit int, now time.Time) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error
}

// EventPublisher публикует social-события в Kafka.
type EventPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
	Close() error
}

// SocialCache предоставляет application-слою кэш read-моделей social.
type SocialCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// ReadinessChecker проверяет готовность social-сервиса принимать трафик.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}
