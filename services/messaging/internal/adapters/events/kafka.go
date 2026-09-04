// Package events содержит Kafka-адаптеры messaging-сервиса.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"uuid"

	"github.com/segmentio/kafka-go"

	"general-project/messaging/internal/ports"
)

// Consumer читает интеграционные события для transient notification stream.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создаёт consumer с переданной Kafka group ID.
func NewConsumer(broker string, topic string, groupID string, maxBytes int) *Consumer {
	return &Consumer{reader: kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: maxBytes,
		// Уведомления transient: новый экземпляр не должен воспроизводить
		// исторические social-события подписчикам, подключившимся позже.
		StartOffset: kafka.LastOffset,
	})}
}

// Run читает события и подтверждает их только после успешной обработки.
func (c *Consumer) Run(ctx context.Context, handle func(context.Context, ports.IntegrationEvent) error) error {
	defer c.reader.Close()
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
			// Повреждённый envelope не должен блокировать последующие события.
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
		ID:            envelope.EventID,
		EventType:     envelope.EventType,
		AggregateID:   envelope.AggregateID,
		Payload:       envelope.Payload,
		CorrelationID: envelope.CorrelationID,
		CreatedAt:     envelope.CreatedAt,
	}, nil
}
