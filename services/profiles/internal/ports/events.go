package ports

import (
	"context"
	"encoding/json"
	"time"
	"uuid"
)

// IntegrationEvent описывает событие из общего Kafka topic на границе profiles.
type IntegrationEvent struct {
	ID            uuid.UUID
	EventType     string
	AggregateID   *uuid.UUID
	Payload       json.RawMessage
	CorrelationID string
	CreatedAt     time.Time
}

// EventConsumer читает интеграционные события и подтверждает их после обработки.
type EventConsumer interface {
	Run(ctx context.Context, handle func(context.Context, IntegrationEvent) error) error
	Close() error
}
