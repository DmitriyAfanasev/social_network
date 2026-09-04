// Package kafka содержит Kafka-адаптеры analytics-сервиса.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"uuid"

	"github.com/segmentio/kafka-go"

	"general-project/analytics/internal/domain"
)

// Publisher публикует интеграционные события в общий Kafka topic.
type Publisher struct {
	writer    *kafka.Writer
	dlqWriter *kafka.Writer
}

// NewPublisher создаёт Kafka publisher.
func NewPublisher(broker string, topic string) *Publisher {
	return NewPublisherWithDLQ(broker, topic, "")
}

// NewPublisherWithDLQ создаёт publisher основного topic и dead-letter topic.
func NewPublisherWithDLQ(broker string, topic string, dlqTopic string) *Publisher {
	publisher := &Publisher{writer: &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}}}
	if dlqTopic != "" {
		publisher.dlqWriter = &kafka.Writer{Addr: kafka.TCP(broker), Topic: dlqTopic, Balancer: &kafka.Hash{}}
	}
	return publisher
}

// Publish сериализует и отправляет событие с ключом его UUID.
func (p *Publisher) Publish(ctx context.Context, event domain.Event) error {
	payload, err := json.Marshal(eventEnvelope{EventID: event.ID, EventType: event.EventType, AggregateID: event.AggregateID, Payload: json.RawMessage(event.Payload), CorrelationID: event.CorrelationID, CreatedAt: event.CreatedAt})
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Key: []byte(event.ID.String()), Value: payload})
}

// PublishDeadLetter отправляет неисправимое событие в DLQ вместе с причиной.
func (p *Publisher) PublishDeadLetter(ctx context.Context, event domain.Event, reason string, attempts int) error {
	if p.dlqWriter == nil {
		return errors.New("analytics DLQ publisher is not configured")
	}
	payload, err := json.Marshal(eventEnvelope{EventID: event.ID, EventType: event.EventType, AggregateID: event.AggregateID, Payload: json.RawMessage(event.Payload), CorrelationID: event.CorrelationID, CreatedAt: event.CreatedAt, DeadLetterReason: reason, DeliveryAttempts: attempts})
	if err != nil {
		return err
	}
	return p.dlqWriter.WriteMessages(ctx, kafka.Message{Key: []byte(event.ID.String()), Value: payload})
}

// Close закрывает Kafka writer.
func (p *Publisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return err
	}
	if p.dlqWriter != nil {
		return p.dlqWriter.Close()
	}
	return nil
}

// Consumer читает интеграционные события и подтверждает их после обработки.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создаёт Kafka consumer с выделенной consumer group.
func NewConsumer(broker string, topic string, groupID string, maxBytes int) *Consumer {
	return &Consumer{reader: kafka.NewReader(kafka.ReaderConfig{Brokers: []string{broker}, Topic: topic, GroupID: groupID, MinBytes: 1, MaxBytes: maxBytes})}
}

// Run принимает события, пропуская только явно некорректные сообщения.
func (c *Consumer) Run(ctx context.Context, handle func(context.Context, domain.Event) error) error {
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

type eventEnvelope struct {
	EventID          uuid.UUID       `json:"event_id"`
	EventType        string          `json:"event_type"`
	AggregateID      *uuid.UUID      `json:"aggregate_id,omitempty"`
	Payload          json.RawMessage `json:"payload"`
	CorrelationID    string          `json:"correlation_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	DeadLetterReason string          `json:"dead_letter_reason,omitempty"`
	DeliveryAttempts int             `json:"delivery_attempts,omitempty"`
}

func decodeEvent(value []byte) (domain.Event, error) {
	var envelope eventEnvelope
	if err := json.Unmarshal(value, &envelope); err != nil {
		return domain.Event{}, err
	}
	if envelope.EventID == uuid.Nil() || envelope.EventType == "" || len(envelope.Payload) == 0 || envelope.CreatedAt.IsZero() {
		return domain.Event{}, errors.New("invalid event envelope")
	}
	return domain.Event{ID: envelope.EventID, EventType: envelope.EventType, AggregateID: envelope.AggregateID, Payload: envelope.Payload, CorrelationID: envelope.CorrelationID, CreatedAt: envelope.CreatedAt}, nil
}
