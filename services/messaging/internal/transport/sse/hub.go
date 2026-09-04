// Package sse содержит bounded fan-out для HTTP Server-Sent Events.
package sse

import (
	"sync"
	"uuid"

	"general-project/messaging/internal/ports"
)

const subscriberBuffer = 32

// Hub хранит подписчиков SSE только в памяти текущего процесса.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan []byte]struct{}
}

// NewHub создаёт пустой SSE hub.
func NewHub() *Hub {
	return &Hub{subscribers: make(map[uuid.UUID]map[chan []byte]struct{})}
}

// Subscribe регистрирует пользователя и возвращает bounded-канал событий.
func (h *Hub) Subscribe(userID uuid.UUID) (<-chan []byte, func()) {
	channel := make(chan []byte, subscriberBuffer)
	h.mu.Lock()
	if h.subscribers[userID] == nil {
		h.subscribers[userID] = make(map[chan []byte]struct{})
	}
	h.subscribers[userID][channel] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers[userID], channel)
			if len(h.subscribers[userID]) == 0 {
				delete(h.subscribers, userID)
			}
			h.mu.Unlock()
			close(channel)
		})
	}
	return channel, unsubscribe
}

// Publish доставляет событие локальным подписчикам без блокировки Redis consumer.
func (h *Hub) Publish(event ports.RealtimeEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, recipientID := range event.RecipientIDs {
		for channel := range h.subscribers[recipientID] {
			select {
			case channel <- append([]byte(nil), event.Payload...):
			default:
				// Медленный SSE-клиент пропускает событие, чтобы не блокировать fan-out.
			}
		}
	}
}
