// Package events содержит Kafka-адаптеры content-сервиса.
package events

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"github.com/segmentio/kafka-go"

	"general-project/content/internal/ports"
)

// Publisher публикует content-события в Kafka.
type Publisher struct {
	writer *kafka.Writer
}

// NewPublisher создаёт Kafka publisher для общего topic событий.
func NewPublisher(broker string, topic string) *Publisher {
	return &Publisher{writer: &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}}}
}

// Publish отправляет outbox-событие с ключом aggregate ID или event ID.
func (p *Publisher) Publish(ctx context.Context, event ports.OutboxEvent) error {
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

// Close закрывает Kafka writer.
func (p *Publisher) Close() error {
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
