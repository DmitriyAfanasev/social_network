// Package application содержит сценарии публикации и обработки analytics-событий.
package application

import (
	"context"
	"errors"
	"time"

	"general-project/analytics/internal/domain"
	"general-project/analytics/internal/ports"
)

const (
	defaultBatchSize = 100
)

// OutboxPublisher публикует незавершённые события с повторными попытками.
type OutboxPublisher struct {
	outbox    ports.OutboxRepository
	broker    ports.EventPublisher
	batchSize int
}

// NewOutboxPublisher создаёт application-сервис publisher-а.
func NewOutboxPublisher(outbox ports.OutboxRepository, broker ports.EventPublisher) *OutboxPublisher {
	return &OutboxPublisher{outbox: outbox, broker: broker, batchSize: defaultBatchSize}
}

// Run запускает периодическую обработку outbox до отмены контекста.
func (p *OutboxPublisher) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := p.RunOnce(ctx); err != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// RunOnce забирает пачку событий и обрабатывает их независимо.
func (p *OutboxPublisher) RunOnce(ctx context.Context) error {
	now := time.Now().UTC()
	events, err := p.outbox.Claim(ctx, p.batchSize, now)
	if err != nil {
		return err
	}
	var firstErr error
	for _, event := range events {
		if err := p.broker.Publish(ctx, event); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if markErr := p.outbox.MarkFailed(ctx, event.ID, nextAttempt(now, event.Attempts), err.Error()); markErr != nil && firstErr == nil {
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

// EventConsumer обрабатывает Kafka-события с защитой от повторной доставки.
type EventConsumer struct {
	processed ports.ProcessedEventRepository
	handler   ports.EventHandler
	dlq       ports.DeadLetterPublisher
	maxTries  int
	retryBase time.Duration
}

// NewEventConsumer создаёт идемпотентный consumer событий.
func NewEventConsumer(processed ports.ProcessedEventRepository, handler ports.EventHandler, dlq ports.DeadLetterPublisher, maxTries int, retryBases ...time.Duration) *EventConsumer {
	if maxTries < 1 {
		maxTries = 3
	}
	retryBase := time.Second
	if len(retryBases) > 0 && retryBases[0] >= 0 {
		retryBase = retryBases[0]
	}
	return &EventConsumer{processed: processed, handler: handler, dlq: dlq, maxTries: maxTries, retryBase: retryBase}
}

// Handle принимает событие после успешного декодирования Kafka-сообщения.
func (c *EventConsumer) Handle(ctx context.Context, event domain.Event) error {
	for {
		claim, err := c.processed.Claim(ctx, event.ID, time.Now().UTC())
		if err != nil || !claim.Claimed {
			return err
		}
		if err := c.handler.Handle(ctx, event); err != nil {
			if claim.Attempts >= c.maxTries {
				if c.dlq == nil {
					return err
				}
				if dlqErr := c.dlq.PublishDeadLetter(ctx, event, err.Error(), claim.Attempts); dlqErr != nil {
					return dlqErr
				}
				return c.processed.MarkProcessed(ctx, event.ID, time.Now().UTC())
			}
			if markErr := c.processed.MarkFailed(ctx, event.ID, err.Error()); markErr != nil {
				return markErr
			}
			if waitErr := waitRetry(ctx, retryDelay(c.retryBase, claim.Attempts)); waitErr != nil {
				return waitErr
			}
			continue
		}
		return c.processed.MarkProcessed(ctx, event.ID, time.Now().UTC())
	}
}

func nextAttempt(now time.Time, attempts int) time.Time {
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Duration(1<<minInt(attempts, 6)) * time.Second
	return now.Add(delay)
}

func retryDelay(base time.Duration, attempts int) time.Duration {
	if base <= 0 {
		return 0
	}
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<attempts) * base
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func minInt(first int, second int) int {
	if first < second {
		return first
	}
	return second
}
