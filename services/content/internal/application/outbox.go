package application

import (
	"context"
	"time"

	"general-project/content/internal/ports"
)

const outboxBatchSize = 100

// OutboxPublisher публикует content-события с повторными попытками.
type OutboxPublisher struct {
	outbox ports.OutboxRepository
	broker ports.EventPublisher
}

// NewOutboxPublisher создаёт publisher content-событий.
func NewOutboxPublisher(outbox ports.OutboxRepository, broker ports.EventPublisher) *OutboxPublisher {
	return &OutboxPublisher{outbox: outbox, broker: broker}
}

// Run запускает периодическую публикацию событий до отмены контекста.
func (p *OutboxPublisher) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := p.RunOnce(ctx); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// RunOnce публикует одну пачку, не блокируя остальные события при ошибке.
func (p *OutboxPublisher) RunOnce(ctx context.Context) error {
	now := time.Now().UTC()
	events, err := p.outbox.Claim(ctx, outboxBatchSize, now)
	if err != nil {
		return err
	}
	var firstErr error
	for _, event := range events {
		if err := p.broker.Publish(ctx, event); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if markErr := p.outbox.MarkFailed(ctx, event.ID, now.Add(retryDelay(event.Attempts)), err.Error()); markErr != nil && firstErr == nil {
				firstErr = markErr
			}
			continue
		}
		if err := p.outbox.MarkPublished(ctx, event.ID, now); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<attempts) * time.Second
}
