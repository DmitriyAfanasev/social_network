package ports

import (
	"context"
	"encoding/json"
	"time"
	"uuid"
)

// RealtimeEvent описывает событие, которое нужно доставить выбранным пользователям.
type RealtimeEvent struct {
	RecipientIDs []uuid.UUID     `json:"recipient_ids"`
	Payload      json.RawMessage `json:"payload"`
}

// RealtimeBroker предоставляет межинстансовую доставку realtime-событий.
type RealtimeBroker interface {
	Publish(ctx context.Context, event RealtimeEvent) error
	Subscribe(ctx context.Context, handler func(context.Context, RealtimeEvent) error) error
	Close() error
}

// NotificationStream доставляет realtime payload подписчикам SSE по UUID пользователя.
type NotificationStream interface {
	Subscribe(userID uuid.UUID) (<-chan []byte, func())
	Publish(event RealtimeEvent)
}

// PresenceStore хранит heartbeat пользователей и вычисляет их online-состояние.
type PresenceStore interface {
	Heartbeat(ctx context.Context, userID uuid.UUID, ttl time.Duration) error
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
}

// PresenceEvent сообщает экземплярам messaging об изменении online-состояния.
type PresenceEvent struct {
	UserID uuid.UUID `json:"user_id"`
	Online bool      `json:"online"`
}

// PresenceFanout доставляет presence-события всем экземплярам сервиса.
type PresenceFanout interface {
	SubscribePresence(ctx context.Context, handler func(context.Context, PresenceEvent) error) error
}
