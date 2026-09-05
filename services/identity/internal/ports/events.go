package ports

import (
	"context"
	"time"
	"uuid"
)

// OutboxEvent описывает identity-событие, которое публикуется после commit.
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

// OutboxRepository предоставляет application-слою доступ к identity outbox.
type OutboxRepository interface {
	Claim(ctx context.Context, limit int, now time.Time) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error
}

// EventPublisher публикует identity-события в Kafka.
type EventPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
	Close() error
}
