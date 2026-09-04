package events

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"github.com/segmentio/kafka-go"

	"general-project/media/internal/ports"
)

// EventPublisher публикует media outbox-события в общий Kafka topic.
type EventPublisher struct {
	writer *kafka.Writer
}

// NewEventPublisher создаёт publisher media-событий.
func NewEventPublisher(broker string, topic string) *EventPublisher {
	return &EventPublisher{writer: &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}}}
}

// Publish отправляет событие с ключом aggregate ID или event ID.
func (p *EventPublisher) Publish(ctx context.Context, event ports.OutboxEvent) error {
	payload, err := json.Marshal(eventEnvelope{EventID: event.ID, EventType: event.EventType, AggregateID: event.AggregateID, Payload: json.RawMessage(event.Payload), CorrelationID: event.CorrelationID, CreatedAt: event.CreatedAt})
	if err != nil {
		return err
	}
	key := event.ID.String()
	if event.AggregateID != nil {
		key = event.AggregateID.String()
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: payload})
}

// Close закрывает publisher media-событий.
func (p *EventPublisher) Close() error {
	return p.writer.Close()
}

type eventEnvelope struct {
	EventID       uuid.UUID       `json:"event_id"`
	EventType     string          `json:"event_type"`
	AggregateID   *uuid.UUID      `json:"aggregate_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}
