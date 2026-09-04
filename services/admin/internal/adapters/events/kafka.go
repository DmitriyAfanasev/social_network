// Package events содержит Kafka-адаптеры admin-сервиса.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"uuid"

	"github.com/segmentio/kafka-go"
)

// Event содержит минимальный envelope интеграционного события.
type Event struct {
	ID            uuid.UUID       `json:"event_id"`
	EventType     string          `json:"event_type"`
	AggregateID   *uuid.UUID      `json:"aggregate_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// Consumer читает события для admin read model.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создаёт consumer moderation-событий.
func NewConsumer(broker string, topic string, groupID string) *Consumer {
	return &Consumer{reader: kafka.NewReader(kafka.ReaderConfig{Brokers: []string{broker}, Topic: topic, GroupID: groupID, MinBytes: 1, MaxBytes: 10_000_000})}
}

// Run читает события и подтверждает их только после успешной обработки.
func (c *Consumer) Run(ctx context.Context, handle func(context.Context, Event) error) error {
	defer c.reader.Close()
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		var event Event
		if err := json.Unmarshal(message.Value, &event); err != nil || event.ID == uuid.Nil() || event.EventType == "" || len(event.Payload) == 0 || event.CreatedAt.IsZero() {
			_ = c.reader.CommitMessages(ctx, message)
			continue
		}
		if err := handle(ctx, event); err != nil {
			return err
		}
		if err := c.reader.CommitMessages(ctx, message); err != nil {
			return err
		}
	}
}
