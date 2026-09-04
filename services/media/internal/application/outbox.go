package application

import (
	"context"
	"time"

	"general-project/media/internal/ports"
)

const mediaOutboxBatchSize = 100

// OutboxPublisher публикует media-события с повторными попытками.
type OutboxPublisher struct {
	outbox ports.OutboxRepository
	broker ports.EventPublisher
}

// NewOutboxPublisher создаёт publisher media-событий.
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
		_ = p.RunOnce(ctx)
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
	events, err := p.outbox.Claim(ctx, mediaOutboxBatchSize, now)
	if err != nil {
		return err
	}
	var firstErr error
	for _, event := range events {
		if err := p.broker.Publish(ctx, event); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			_ = p.outbox.MarkFailed(ctx, event.ID, now.Add(mediaRetryDelay(event.Attempts)), err.Error())
			continue
		}
		if err := p.outbox.MarkPublished(ctx, event.ID, now); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func mediaRetryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<attempts) * time.Second
}
