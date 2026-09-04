package ports

import (
	"context"
	"errors"
	"io"
	"time"
	"uuid"

	"general-project/media/internal/domain"
)

var (
	// ErrNotFound означает, что медиаобъект отсутствует.
	ErrNotFound = errors.New("media not found")
	// ErrAlreadyExists означает конфликт уникального object key.
	ErrAlreadyExists = errors.New("media already exists")
)

// MediaRepository предоставляет application-слою доступ к метаданным медиа.
type MediaRepository interface {
	Create(ctx context.Context, media domain.Media) (domain.Media, error)
	CreateWithOutbox(ctx context.Context, media domain.Media, event OutboxEvent) (domain.Media, error)
	FindByID(ctx context.Context, mediaID uuid.UUID) (domain.Media, error)
	Delete(ctx context.Context, mediaID uuid.UUID, deletedAt time.Time) error
}

// OutboxEvent описывает media-событие, ожидающее публикации после commit.
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

// OutboxRepository предоставляет application-слою доступ к media outbox.
type OutboxRepository interface {
	Claim(ctx context.Context, limit int, now time.Time) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error
}

// EventPublisher публикует media-события в Kafka.
type EventPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
	Close() error
}

// MediaCache хранит DTO метаданных для быстрых повторных чтений.
type MediaCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// ObjectStorage хранит бинарное содержимое и не знает о доменной модели media.
type ObjectStorage interface {
	Put(ctx context.Context, objectKey string, contentType string, content io.Reader, size int64) error
	Delete(ctx context.Context, objectKey string) error
}

// ReadinessChecker проверяет готовность media-сервиса принимать трафик.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}
