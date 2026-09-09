// Package events содержит Kafka-адаптеры profiles-сервиса.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"uuid"

	"github.com/segmentio/kafka-go"

	"general-project/profiles/internal/ports"
)

// Consumer читает identity-события для создания профилей.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создаёт consumer с отдельной durable-группой profiles.
func NewConsumer(broker string, topic string, groupID string) *Consumer {
	return &Consumer{reader: kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker}, Topic: topic, GroupID: groupID,
		MinBytes: 1, MaxBytes: 10_000_000, StartOffset: kafka.FirstOffset,
	})}
}

// Run читает события и подтверждает их только после успешной обработки.
func (c *Consumer) Run(ctx context.Context, handle func(context.Context, ports.IntegrationEvent) error) (runErr error) {
	defer func() {
		if err := c.reader.Close(); err != nil && runErr == nil {
			runErr = err
		}
	}()
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		event, err := decodeEvent(message.Value)
		if err != nil {
			// Некорректное сообщение не должно навсегда блокировать consumer group.
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				return err
			}
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

// Close закрывает Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

type eventEnvelope struct {
	EventID       uuid.UUID       `json:"event_id"`
	EventType     string          `json:"event_type"`
	AggregateID   *uuid.UUID      `json:"aggregate_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

func decodeEvent(value []byte) (ports.IntegrationEvent, error) {
	var envelope eventEnvelope
	if err := json.Unmarshal(value, &envelope); err != nil {
		return ports.IntegrationEvent{}, err
	}
	if envelope.EventID == uuid.Nil() || envelope.EventType == "" || len(envelope.Payload) == 0 || envelope.CreatedAt.IsZero() {
		return ports.IntegrationEvent{}, errors.New("invalid event envelope")
	}
	return ports.IntegrationEvent{
		ID: envelope.EventID, EventType: envelope.EventType, AggregateID: envelope.AggregateID,
		Payload: envelope.Payload, CorrelationID: envelope.CorrelationID, CreatedAt: envelope.CreatedAt,
	}, nil
}
