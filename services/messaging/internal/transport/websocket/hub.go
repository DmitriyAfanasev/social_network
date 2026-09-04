// Package websocket содержит realtime transport messaging-сервиса.
package websocket

import (
	"encoding/json"
	"sync"
	"time"
	"uuid"

	"github.com/gorilla/websocket"
)

const clientQueueSize = 64

// Hub управляет локальными WebSocket-подключениями пользователей.
type Hub struct {
	mu          sync.RWMutex
	broadcastMu sync.Mutex
	clients     map[uuid.UUID]map[*client]struct{}
}

type client struct {
	userID     uuid.UUID
	connection *websocket.Conn
	send       chan []byte
	closed     chan struct{}
	closeOnce  sync.Once
}

// NewHub создаёт пустой hub realtime-подключений.
func NewHub() *Hub {
	return &Hub{clients: make(map[uuid.UUID]map[*client]struct{})}
}

func (h *Hub) add(userID uuid.UUID, connection *websocket.Conn) *client {
	item := &client{userID: userID, connection: connection, send: make(chan []byte, clientQueueSize), closed: make(chan struct{})}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*client]struct{})
	}
	h.clients[userID][item] = struct{}{}
	go item.writeLoop(h)
	return item
}

func (h *Hub) remove(userID uuid.UUID, item *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[userID], item)
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
	item.close()
}

// Broadcast доставляет JSON-событие локальным соединениям выбранных пользователей.
func (h *Hub) Broadcast(userIDs []uuid.UUID, payload json.RawMessage) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()
	h.mu.RLock()
	clients := make([]*client, 0)
	seen := make(map[*client]struct{})
	for _, userID := range userIDs {
		for item := range h.clients[userID] {
			if _, ok := seen[item]; !ok {
				clients = append(clients, item)
				seen[item] = struct{}{}
			}
		}
	}
	h.mu.RUnlock()

	for _, item := range clients {
		if !item.enqueue(payload) {
			h.remove(item.userID, item)
		}
	}
}

// BroadcastAll доставляет presence-событие всем локальным подключениям.
func (h *Hub) BroadcastAll(payload json.RawMessage) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()
	h.mu.RLock()
	clients := make([]*client, 0)
	for _, userClients := range h.clients {
		for item := range userClients {
			clients = append(clients, item)
		}
	}
	h.mu.RUnlock()
	for _, item := range clients {
		if !item.enqueue(payload) {
			h.remove(item.userID, item)
		}
	}
}

func (c *client) enqueue(payload []byte) bool {
	select {
	case <-c.closed:
		return false
	default:
	}
	select {
	case c.send <- append([]byte(nil), payload...):
		return true
	case <-c.closed:
		return false
	default:
		return false
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.connection.Close()
	})
}

func (c *client) writeLoop(h *Hub) {
	for {
		select {
		case <-c.closed:
			return
		case payload := <-c.send:
			if err := c.connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				h.remove(c.userID, c)
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, payload); err != nil {
				h.remove(c.userID, c)
				return
			}
		}
	}
}
