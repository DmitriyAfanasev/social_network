// Package ports содержит интерфейсы, которыми владеет application-слой.
package ports

import (
	"context"
	"errors"
	"time"

	"uuid"

	"general-project/content/internal/domain"
)

var (
	// ErrNotFound означает, что пост отсутствует.
	ErrNotFound = errors.New("post not found")
	// ErrMediaNotFound означает, что вложение отсутствует или удалено.
	ErrMediaNotFound = errors.New("media reference not found")
	// ErrMediaForbidden означает, что вложение принадлежит другому пользователю.
	ErrMediaForbidden = errors.New("media reference forbidden")
)

// OutboxEvent описывает событие, которое должно быть опубликовано после commit.
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

// PostRepository предоставляет application-слою доступ к постам.
type PostRepository interface {
	Create(ctx context.Context, post domain.Post) (domain.Post, error)
	CreateWithOutbox(ctx context.Context, post domain.Post, event OutboxEvent) (domain.Post, error)
	FindByID(ctx context.Context, postID uuid.UUID) (domain.Post, error)
	ListRecent(ctx context.Context, limit int) ([]domain.Post, error)
	Update(ctx context.Context, postID uuid.UUID, body string, mediaIDs []uuid.UUID) (domain.Post, error)
	Delete(ctx context.Context, postID uuid.UUID) error
}

// OutboxRepository предоставляет application-слою доступ к очереди событий content.
type OutboxRepository interface {
	Claim(ctx context.Context, limit int, now time.Time) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error
}

// EventPublisher публикует событие в Kafka после его commit в outbox.
type EventPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
	Close() error
}

// PostCache предоставляет application-слою абстракцию cache read-моделей.
type PostCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// MediaReferenceChecker проверяет доступность и ownership вложений поста.
type MediaReferenceChecker interface {
	ValidateOwned(ctx context.Context, userID uuid.UUID, mediaIDs []uuid.UUID) error
}

// ReadinessChecker проверяет готовность content-сервиса принимать трафик.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}
