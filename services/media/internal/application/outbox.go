package application

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"general-project/media/internal/ports"
)

const mediaOutboxBatchSize = 100

// OutboxPublisher публикует media-события с повторными попытками.
type OutboxPublisher struct {
	outbox    ports.OutboxRepository
	broker    ports.EventPublisher
	imageJobs ports.ImageOptimizationPublisher
}

// NewOutboxPublisher создаёт publisher media-событий.
func NewOutboxPublisher(outbox ports.OutboxRepository, broker ports.EventPublisher, imageJobs ...ports.ImageOptimizationPublisher) *OutboxPublisher {
	var imageJobPublisher ports.ImageOptimizationPublisher
	if len(imageJobs) > 0 {
		imageJobPublisher = imageJobs[0]
	}
	return &OutboxPublisher{outbox: outbox, broker: broker, imageJobs: imageJobPublisher}
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
		if p.imageJobs != nil {
			if job, ok := imageOptimizationJob(event); ok {
				if err := p.imageJobs.Publish(ctx, job); err != nil {
					if firstErr == nil {
						firstErr = err
					}
					_ = p.outbox.MarkFailed(ctx, event.ID, now.Add(mediaRetryDelay(event.Attempts)), err.Error())
					continue
				}
			}
		}
		if err := p.outbox.MarkPublished(ctx, event.ID, now); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func imageOptimizationJob(event ports.OutboxEvent) (ports.ImageOptimizationJob, bool) {
	if event.EventType != "media.media.created" {
		return ports.ImageOptimizationJob{}, false
	}
	var payload struct {
		MediaID   string `json:"media_id"`
		MediaType string `json:"media_type"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.MediaType != "image" {
		return ports.ImageOptimizationJob{}, false
	}
	mediaID, err := uuid.Parse(payload.MediaID)
	if err != nil {
		return ports.ImageOptimizationJob{}, false
	}
	return ports.ImageOptimizationJob{MediaID: mediaID}, true
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
