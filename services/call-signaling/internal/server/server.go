// Package server содержит WebSocket transport call signaling.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"uuid"

	"github.com/gorilla/websocket"

	"general-project/call-signaling/internal/application"
	"general-project/call-signaling/internal/authz"
	"general-project/call-signaling/internal/domain"
	"general-project/call-signaling/internal/ports"
	platformauth "general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
)

const maxSignalBytes = 262144

// Signal содержит одно opaque-сообщение WebRTC negotiation.
type Signal struct {
	Kind      string          `json:"kind,omitempty"`
	SDP       json.RawMessage `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
}

// ClientEvent содержит единственную форму команды от браузера.
type ClientEvent struct {
	Type           string    `json:"type"`
	ConversationID uuid.UUID `json:"conversation_id"`
	TargetUserID   uuid.UUID `json:"target_user_id"`
	CallID         uuid.UUID `json:"call_id"`
	CallType       string    `json:"call_type"`
	Signal         *Signal   `json:"signal"`
}

// Event отправляется авторизованным участникам звонка.
type Event struct {
	Type           string      `json:"type"`
	Message        string      `json:"message,omitempty"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	SenderID       uuid.UUID   `json:"sender_id"`
	CallID         uuid.UUID   `json:"call_id"`
	CallerID       uuid.UUID   `json:"caller_id"`
	CalleeID       uuid.UUID   `json:"callee_id"`
	CallType       string      `json:"call_type"`
	Status         string      `json:"status"`
	RecipientIDs   []uuid.UUID `json:"recipient_ids"`
	Signal         *Signal     `json:"signal,omitempty"`
}

// client связывает WebSocket с идентификатором аутентифицированного пользователя.
type client struct {
	userID uuid.UUID
	conn   *websocket.Conn
	mu     sync.Mutex
}

// send сериализует записи: gorilla/websocket допускает только одного writer.
func (c *client) send(value Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(value)
}

// Hub хранит локальные сокеты, а межпроцессную доставку делегирует брокеру.
type Hub struct {
	mu     sync.RWMutex
	rooms  map[uuid.UUID]map[*client]struct{}
	broker ports.EventBroker
	logger *slog.Logger
}

// NewHub создаёт локальный реестр WebSocket-подключений.
func NewHub(broker ports.EventBroker, logger *slog.Logger) *Hub {
	return &Hub{rooms: make(map[uuid.UUID]map[*client]struct{}), broker: broker, logger: logger}
}

// Run принимает события от всех экземпляров call-signaling.
func (h *Hub) Run(ctx context.Context) error {
	return h.broker.Subscribe(ctx, func(event ports.SignalingEvent) error {
		var value Event
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			h.logger.Warn("invalid signaling event", "error", err)
			return nil
		}
		h.broadcastLocal(event.ConversationID, value)
		return nil
	})
}

func (h *Hub) broadcast(ctx context.Context, conversationID uuid.UUID, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return h.broker.Publish(ctx, ports.SignalingEvent{ConversationID: conversationID, Payload: payload})
}

func (h *Hub) broadcastLocal(conversationID uuid.UUID, event Event) {
	h.mu.RLock()
	clients := make([]*client, 0, len(h.rooms[conversationID]))
	allowed := make(map[uuid.UUID]struct{}, len(event.RecipientIDs))
	for _, id := range event.RecipientIDs {
		allowed[id] = struct{}{}
	}
	for item := range h.rooms[conversationID] {
		if _, ok := allowed[item.userID]; ok {
			clients = append(clients, item)
		}
	}
	h.mu.RUnlock()
	for _, item := range clients {
		if err := item.send(event); err != nil {
			h.remove(conversationID, item)
		}
	}
}

func (h *Hub) add(conversationID uuid.UUID, item *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[conversationID] == nil {
		h.rooms[conversationID] = make(map[*client]struct{})
	}
	h.rooms[conversationID][item] = struct{}{}
}

func (h *Hub) remove(conversationID uuid.UUID, item *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[conversationID], item)
	if len(h.rooms[conversationID]) == 0 {
		delete(h.rooms, conversationID)
	}
}

// Handler аутентифицирует WebSocket и запускает команды call signaling.
type Handler struct {
	hub            *Hub
	calls          *application.Service
	verifier       platformauth.TokenVerifier
	readiness      ports.ReadinessChecker
	allowedOrigins []string
	maxMessageSize int64
}

// NewHandler создаёт transport с application-сервисом и техническими портами.
func NewHandler(calls *application.Service, hub *Hub, verifier platformauth.TokenVerifier, readiness ports.ReadinessChecker, allowedOrigins []string, maxMessageSize int64) *Handler {
	return &Handler{hub: hub, calls: calls, verifier: verifier, readiness: readiness, allowedOrigins: allowedOrigins, maxMessageSize: maxMessageSize}
}

// ServeHTTP аутентифицирует запрос, подписывает сокет на команды и пересылает сигналы.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		return origin == "" || slices.Contains(h.allowedOrigins, origin)
	}}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			return
		}
	}()
	readLimit := h.maxMessageSize
	if readLimit <= 0 {
		readLimit = maxSignalBytes
	}
	conn.SetReadLimit(readLimit)
	item := &client{userID: userID, conn: conn}
	rooms := make(map[uuid.UUID]struct{})
	for {
		var input ClientEvent
		if err := conn.ReadJSON(&input); err != nil {
			break
		}
		if err := h.handle(r.Context(), item, input, rooms); err != nil {
			if sendErr := item.send(Event{Type: "error", Message: publicError(err)}); sendErr != nil {
				break
			}
		}
	}
	for conversationID := range rooms {
		h.hub.remove(conversationID, item)
	}
}

// Ready возвращает готовность PostgreSQL и Redis-зависимостей.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.readiness.Check(r.Context()); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "dependencies_unavailable", "зависимости недоступны")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ready"}` + "\n")); err != nil {
		return
	}
}

// Health возвращает liveness-состояние процесса.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}` + "\n")); err != nil {
		return
	}
}

func (h *Handler) handle(ctx context.Context, item *client, input ClientEvent, rooms map[uuid.UUID]struct{}) error {
	switch input.Type {
	case "conversation.subscribe":
		if err := h.calls.Subscribe(ctx, input.ConversationID, item.userID); err != nil {
			return err
		}
		h.hub.add(input.ConversationID, item)
		rooms[input.ConversationID] = struct{}{}
		return item.send(Event{Type: "conversation.subscribed", ConversationID: input.ConversationID})
	case "ping":
		return item.send(Event{Type: "pong"})
	case "call.keepalive":
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		return h.calls.KeepAlive(ctx, input.ConversationID, item.userID, input.CallID)
	case "call.start":
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		session, err := h.calls.Start(ctx, input.ConversationID, item.userID, input.TargetUserID, domain.CallType(input.CallType))
		if err != nil {
			return err
		}
		return h.publishCall(ctx, input.ConversationID, item.userID, session, "call.invite", nil)
	case "call.accept", "call.reject", "call.end":
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		session, err := h.calls.Transition(ctx, input.ConversationID, item.userID, input.CallID, input.Type[len("call."):])
		if err != nil {
			return err
		}
		return h.publishCall(ctx, input.ConversationID, item.userID, session, input.Type, nil)
	case "call.signal":
		if _, ok := rooms[input.ConversationID]; !ok || !validSignal(input.Signal) {
			return errors.New("invalid signal")
		}
		session, err := h.calls.AuthorizeSignal(ctx, input.ConversationID, item.userID, input.CallID)
		if err != nil {
			return err
		}
		return h.publishCall(ctx, input.ConversationID, item.userID, session, input.Type, input.Signal)
	default:
		return fmt.Errorf("unknown event type %q", input.Type)
	}
}

func (h *Handler) publishCall(ctx context.Context, conversationID, senderID uuid.UUID, session application.SessionDTO, eventType string, signal *Signal) error {
	event := Event{Type: eventType, ConversationID: conversationID, SenderID: senderID, CallID: session.CallID, CallerID: session.CallerID, CalleeID: session.CalleeID, CallType: string(session.CallType), Status: string(session.Status), RecipientIDs: []uuid.UUID{session.CallerID, session.CalleeID}, Signal: signal}
	return h.hub.broadcast(ctx, conversationID, event)
}

func (h *Handler) userID(r *http.Request) (uuid.UUID, error) {
	token := r.URL.Query().Get("access_token")
	if token == "" {
		var ok bool
		token, ok = platformauth.BearerToken(r.Header.Get("Authorization"))
		if !ok {
			return uuid.Nil(), platformauth.ErrUnauthorized
		}
	}
	return h.verifier.UserID(token)
}

func validSignal(signal *Signal) bool {
	if signal == nil {
		return false
	}
	switch signal.Kind {
	case "offer", "answer":
		return len(signal.SDP) > 0
	case "ice":
		return len(signal.Candidate) > 0
	default:
		return false
	}
}

func publicError(err error) string {
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return "операция запрещена"
	case errors.Is(err, domain.ErrOnlyCallee):
		return "действие доступно только вызываемому"
	case errors.Is(err, domain.ErrInvalidState):
		return "недопустимое состояние звонка"
	case errors.Is(err, application.ErrCallNotFound):
		return "звонок не найден"
	case errors.Is(err, application.ErrCallNotActive):
		return "звонок не активен"
	default:
		return err.Error()
	}
}
