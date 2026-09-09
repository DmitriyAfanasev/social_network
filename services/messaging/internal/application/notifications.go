// Package application содержит сценарии уведомлений messaging-сервиса.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
	"uuid"

	"general-project/messaging/internal/ports"
)

const (
	friendshipCreatedEvent    = "social.friendship.created"
	friendRequestCreatedEvent = "social.friend_request.created"
	friendAcceptedType        = "friend.accepted"
	friendRequestedType       = "friend.requested"
	seenEventTTL              = 10 * time.Minute
	seenEventLimit            = 4096
)

var errInvalidFriendshipEvent = errors.New("invalid social friendship event")

// NotificationEventHandler преобразует межсервисные события в SSE-уведомления.
type NotificationEventHandler struct {
	stream ports.NotificationStream

	mu   sync.Mutex
	seen map[uuid.UUID]time.Time
}

// NewNotificationEventHandler создаёт обработчик уведомлений social-событий.
func NewNotificationEventHandler(stream ports.NotificationStream) *NotificationEventHandler {
	return &NotificationEventHandler{stream: stream, seen: make(map[uuid.UUID]time.Time)}
}

// Handle обрабатывает поддержанные события и безопасно игнорирует остальные.
func (h *NotificationEventHandler) Handle(ctx context.Context, event ports.IntegrationEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.EventType != friendshipCreatedEvent && event.EventType != friendRequestCreatedEvent {
		return nil
	}

	var payload friendshipCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return errors.Join(errInvalidFriendshipEvent, err)
	}
	targetID, actorID := friendshipRecipients(event, payload)
	if targetID == uuid.Nil() || actorID == uuid.Nil() || targetID == actorID {
		return errInvalidFriendshipEvent
	}
	if !h.markSeen(event.ID, time.Now().UTC()) {
		return nil
	}

	notificationType, message := friendAcceptedType, "Заявку в друзья приняли"
	if event.EventType == friendRequestCreatedEvent {
		notificationType, message = friendRequestedType, "Вам отправили заявку в друзья"
	}
	notification, err := json.Marshal(map[string]any{
		"type":     notificationType,
		"actor_id": actorID,
		"message":  message,
	})
	if err != nil {
		return err
	}
	h.stream.Publish(ports.RealtimeEvent{RecipientIDs: []uuid.UUID{targetID}, Payload: notification})
	return nil
}

type friendshipCreatedPayload struct {
	ActorID  uuid.UUID `json:"actor_id"`
	TargetID uuid.UUID `json:"target_id"`
	UserID   uuid.UUID `json:"user_id"`
	FriendID uuid.UUID `json:"friend_id"`
}

func friendshipRecipients(event ports.IntegrationEvent, payload friendshipCreatedPayload) (uuid.UUID, uuid.UUID) {
	targetID := payload.TargetID
	if targetID == uuid.Nil() && event.AggregateID != nil {
		targetID = *event.AggregateID
	}
	actorID := payload.ActorID
	if actorID == uuid.Nil() {
		// Старый payload не содержал actor_id, но aggregate_id у social указывает
		// на адресата, поэтому направление можно восстановить из пары UUID.
		if payload.UserID != uuid.Nil() && payload.UserID != targetID {
			actorID = payload.UserID
		} else if payload.FriendID != uuid.Nil() && payload.FriendID != targetID {
			actorID = payload.FriendID
		}
	}
	return targetID, actorID
}

func (h *NotificationEventHandler) markSeen(eventID uuid.UUID, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, seenAt := range h.seen {
		if now.Sub(seenAt) >= seenEventTTL {
			delete(h.seen, id)
		}
	}
	if _, exists := h.seen[eventID]; exists {
		return false
	}
	if len(h.seen) >= seenEventLimit {
		var oldestID uuid.UUID
		var oldestAt time.Time
		for id, seenAt := range h.seen {
			if oldestAt.IsZero() || seenAt.Before(oldestAt) {
				oldestID, oldestAt = id, seenAt
			}
		}
		delete(h.seen, oldestID)
	}
	h.seen[eventID] = now
	return true
}
