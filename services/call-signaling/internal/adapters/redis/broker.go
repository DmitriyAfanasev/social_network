// Package redis содержит Redis-адаптер межпроцессной доставки signaling-событий.
package redis

import (
	"context"
	"encoding/json"
	"uuid"

	"github.com/redis/go-redis/v9"

	"general-project/call-signaling/internal/ports"
)

const eventsChannel = "calls:v1:events"

type envelope struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Payload        []byte    `json:"payload"`
}

// Broker реализует межпроцессный Pub/Sub для call-сигналов.
type Broker struct {
	client *redis.Client
}

// NewBroker создаёт Redis-брокер signaling-событий.
func NewBroker(client *redis.Client) *Broker {
	return &Broker{client: client}
}

// Publish публикует событие в общем канале calls.
func (b *Broker) Publish(ctx context.Context, event ports.SignalingEvent) error {
	payload, err := json.Marshal(envelope{ConversationID: event.ConversationID, Payload: event.Payload})
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, eventsChannel, payload).Err()
}

// Subscribe читает события от всех экземпляров call-signaling.
func (b *Broker) Subscribe(ctx context.Context, handle func(ports.SignalingEvent) error) (runErr error) {
	subscriber := b.client.Subscribe(ctx, eventsChannel)
	defer func() {
		if err := subscriber.Close(); err != nil && runErr == nil {
			runErr = err
		}
	}()
	for message := range subscriber.Channel() {
		var value envelope
		if err := json.Unmarshal([]byte(message.Payload), &value); err != nil {
			continue
		}
		if value.ConversationID == uuid.Nil() || len(value.Payload) == 0 {
			continue
		}
		if err := handle(ports.SignalingEvent{ConversationID: value.ConversationID, Payload: value.Payload}); err != nil {
			return err
		}
	}
	return ctx.Err()
}
