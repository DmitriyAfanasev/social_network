// Package ports содержит интерфейсы, которыми владеет application-слой.
package ports

import (
	"context"
	"errors"
	"time"
	"uuid"

	"general-project/analytics/internal/domain"
)

var (
	// ErrNotFound означает, что событие отсутствует.
	ErrNotFound = errors.New("event not found")
	// ErrInProgress означает, что событие сейчас обрабатывается другим consumer-ом.
	ErrInProgress = errors.New("event processing in progress")
)

// OutboxRepository предоставляет доступ к очереди исходящих событий.
type OutboxRepository interface {
	Claim(ctx context.Context, limit int, now time.Time) ([]domain.Event, error)
	MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error
}

// EventPublisher отправляет событие в транспорт сообщений.
type EventPublisher interface {
	Publish(ctx context.Context, event domain.Event) error
	Close() error
}

// DeadLetterPublisher отправляет событие, исчерпавшее лимит обработки, в DLQ.
type DeadLetterPublisher interface {
	PublishDeadLetter(ctx context.Context, event domain.Event, reason string, attempts int) error
}

// ProcessedEventRepository обеспечивает идемпотентность consumer-а.
type ProcessedEventRepository interface {
	Claim(ctx context.Context, eventID uuid.UUID, now time.Time) (ClaimResult, error)
	MarkProcessed(ctx context.Context, eventID uuid.UUID, processedAt time.Time) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, reason string) error
}

// ClaimResult содержит результат резервирования события и номер попытки.
type ClaimResult struct {
	Claimed  bool
	Attempts int
}

// EventHandler выполняет прикладную обработку события.
type EventHandler interface {
	Handle(ctx context.Context, event domain.Event) error
}

// SummaryRepository предоставляет application-слою агрегированный analytics summary.
type SummaryRepository interface {
	GetSummary(ctx context.Context) (domain.Summary, error)
}

// ReadinessChecker проверяет доступность обязательных зависимостей сервиса.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}

// VideoAnalyticsRepository читает агрегаты просмотра видео.
type VideoAnalyticsRepository interface {
	GetVideoStats(ctx context.Context, videoID uuid.UUID) (domain.VideoStats, error)
	ListViewerVideoStats(ctx context.Context, userID uuid.UUID, limit int) ([]domain.ViewerVideoStats, error)
}
