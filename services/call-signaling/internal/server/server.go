/*
Package server implements the WebRTC signaling control plane.

The server does not decode, mix or forward RTP/SRTP. Its job is intentionally
small but security-sensitive: authenticate the WebSocket, authorize the room,
validate the call lifecycle, and copy offer/answer/ICE JSON to the other peer.
The browsers still own the RTCPeerConnection and negotiate the encrypted media
path directly (or through TURN when ICE selects a relay).
*/
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

	"general-project/call-signaling/internal/auth"
	"general-project/call-signaling/internal/authz"
	"general-project/call-signaling/internal/domain"
	"general-project/call-signaling/internal/store"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	callEventsChannel = "calls:events"
	maxSignalBytes    = 262144
)

// Signal is the application envelope around one WebRTC negotiation message.
// SDP and ICE are kept as raw JSON because signaling must not rewrite browser
// generated candidates or codec attributes.
type Signal struct {
	Kind      string          `json:"kind,omitempty"`
	SDP       json.RawMessage `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
}

// ClientEvent is the only input shape accepted from a browser. The server gets
// sender identity from the JWT connection, never from this structure.
type ClientEvent struct {
	Type           string  `json:"type"`
	ConversationID int64   `json:"conversation_id"`
	TargetUserID   int64   `json:"target_user_id"`
	CallID         string  `json:"call_id"`
	CallType       string  `json:"call_type"`
	Signal         *Signal `json:"signal"`
}

// Event is sent to one or both authorized call participants. RecipientIDs is
// also used by the Redis fan-out layer to prevent delivery to unrelated rooms.
type Event struct {
	Type           string  `json:"type"`
	Message        string  `json:"message,omitempty"`
	ConversationID int64   `json:"conversation_id"`
	SenderID       int64   `json:"sender_id"`
	CallID         string  `json:"call_id"`
	CallerID       int64   `json:"caller_id"`
	CalleeID       int64   `json:"callee_id"`
	CallType       string  `json:"call_type"`
	Status         string  `json:"status"`
	RecipientIDs   []int64 `json:"recipient_ids"`
	Signal         *Signal `json:"signal,omitempty"`
}

// envelope is the Redis Pub/Sub wire format; conversation_id selects local rooms.
type envelope struct {
	ConversationID int64 `json:"conversation_id"`
	Event          Event `json:"event"`
}

// client couples a WebSocket to its authenticated user. The write mutex is
// required because Redis fan-out and request handling can write concurrently.
type client struct {
	userID int64
	conn   *websocket.Conn
	mu     sync.Mutex
}

// send serializes writes per connection; gorilla/websocket permits one writer.
func (c *client) send(value Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(value)
}

// Hub tracks local sockets. It is intentionally process-local; Redis Pub/Sub
// makes the same event visible to every replica for horizontal scaling.
type Hub struct {
	mu      sync.RWMutex
	rooms   map[int64]map[*client]struct{}
	redis   *redis.Client
	store   *store.RedisStore
	authorz *authz.Repository
	logger  *slog.Logger
}

// NewHub wires local connection tracking to shared Redis and PostgreSQL adapters.
func NewHub(redisClient *redis.Client, sessionStore *store.RedisStore, authorizer *authz.Repository, logger *slog.Logger) *Hub {
	return &Hub{rooms: make(map[int64]map[*client]struct{}), redis: redisClient, store: sessionStore, authorz: authorizer, logger: logger}
}

// Run consumes signaling events published by this or another service replica.
func (h *Hub) Run(ctx context.Context) error {
	subscriber := h.redis.Subscribe(ctx, callEventsChannel)
	defer subscriber.Close()
	for message := range subscriber.Channel() {
		var value envelope
		if err := json.Unmarshal([]byte(message.Payload), &value); err != nil {
			h.logger.Warn("invalid Redis signaling event", "error", err)
			continue
		}
		h.broadcastLocal(value.ConversationID, value.Event)
	}
	return ctx.Err()
}

// broadcast publishes once to Redis; local delivery happens in the subscriber
// loop, so all replicas follow the same delivery path.
func (h *Hub) broadcast(ctx context.Context, conversationID int64, event Event) error {
	payload, err := json.Marshal(envelope{ConversationID: conversationID, Event: event})
	if err != nil {
		return err
	}
	return h.redis.Publish(ctx, callEventsChannel, payload).Err()
}

// broadcastLocal filters both by conversation room and recipient user id.
func (h *Hub) broadcastLocal(conversationID int64, event Event) {
	h.mu.RLock()
	clients := make([]*client, 0, len(h.rooms[conversationID]))
	allowed := make(map[int64]struct{}, len(event.RecipientIDs))
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

// add subscribes a connection to one authorized conversation room.
func (h *Hub) add(conversationID int64, item *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[conversationID] == nil {
		h.rooms[conversationID] = make(map[*client]struct{})
	}
	h.rooms[conversationID][item] = struct{}{}
}

// remove unregisters a closed or failed connection from a room.
func (h *Hub) remove(conversationID int64, item *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[conversationID], item)
	if len(h.rooms[conversationID]) == 0 {
		delete(h.rooms, conversationID)
	}
}

// Handler authenticates connections and dispatches signaling commands.
type Handler struct {
	Hub            *Hub
	Verifier       *auth.Verifier
	Logger         *slog.Logger
	AllowedOrigins []string
	MaxMessageSize int64
}

// ServeHTTP upgrades an authenticated request and keeps reading commands until
// the browser closes the socket. A rejected handshake never enters the hub.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ws" {
		http.NotFound(w, r)
		return
	}
	userID, err := h.Verifier.UserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		return origin == "" || slices.Contains(h.AllowedOrigins, origin)
	}}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	readLimit := h.MaxMessageSize
	if readLimit <= 0 {
		readLimit = maxSignalBytes
	}
	conn.SetReadLimit(readLimit)
	item := &client{userID: userID, conn: conn}
	defer conn.Close()
	rooms := make(map[int64]struct{})
	for {
		var input ClientEvent
		if err := conn.ReadJSON(&input); err != nil {
			break
		}
		if err := h.handle(r.Context(), item, input, rooms); err != nil {
			_ = item.send(Event{Type: "error", Message: publicError(err)})
		}
	}
	for conversationID := range rooms {
		h.Hub.remove(conversationID, item)
	}
}

// handle contains the protocol state machine at the transport boundary:
// subscribe first, then start/transition/forward only for that subscribed room.
func (h *Handler) handle(ctx context.Context, item *client, input ClientEvent, rooms map[int64]struct{}) error {
	switch input.Type {
	case "conversation.subscribe":
		if err := h.Hub.authorz.CanSubscribe(ctx, input.ConversationID, item.userID); err != nil {
			return err
		}
		h.Hub.add(input.ConversationID, item)
		rooms[input.ConversationID] = struct{}{}
		return item.send(Event{Type: "conversation.subscribed", ConversationID: input.ConversationID})
	case "ping":
		return item.send(Event{Type: "pong"})
	case "call.keepalive":
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		session, err := h.Hub.store.Get(ctx, input.CallID)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return errors.New("call not found")
			}
			return err
		}
		if err := h.Hub.authorz.CanUseCall(ctx, input.ConversationID, item.userID, session.CallerID, session.CalleeID); err != nil {
			return err
		}
		if session.Status != domain.Active {
			return errors.New("call is not active")
		}
		return h.Hub.store.Refresh(ctx, input.CallID)
	case "call.start":
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		if err := h.Hub.authorz.CanStart(ctx, input.ConversationID, item.userID, input.TargetUserID); err != nil {
			return err
		}
		session, err := domain.NewSession(item.userID, input.TargetUserID, domain.CallType(input.CallType))
		if err != nil {
			return err
		}
		if err := h.Hub.store.Create(ctx, session); err != nil {
			return err
		}
		return h.publishCall(ctx, input.ConversationID, item.userID, session, "call.invite", nil)
	case "call.accept", "call.reject", "call.end":
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		session, err := h.Hub.store.Get(ctx, input.CallID)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return errors.New("call not found")
			}
			return err
		}
		if err := h.Hub.authorz.CanUseCall(ctx, input.ConversationID, item.userID, session.CallerID, session.CalleeID); err != nil {
			return err
		}
		session, err = h.Hub.store.Transition(ctx, input.CallID, item.userID, input.Type[len("call."):])
		if err != nil {
			return err
		}
		return h.publishCall(ctx, input.ConversationID, item.userID, session, input.Type, nil)
	case "call.signal":
		if input.Signal == nil || (input.Signal.Kind != "offer" && input.Signal.Kind != "answer" && input.Signal.Kind != "ice") {
			return errors.New("invalid signal")
		}
		if _, ok := rooms[input.ConversationID]; !ok {
			return authz.ErrForbidden
		}
		session, err := h.Hub.store.Get(ctx, input.CallID)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return errors.New("call not found")
			}
			return err
		}
		if err := h.Hub.authorz.CanUseCall(ctx, input.ConversationID, item.userID, session.CallerID, session.CalleeID); err != nil {
			return err
		}
		if session.Status != domain.Active {
			return errors.New("call is not active")
		}
		return h.publishCall(ctx, input.ConversationID, item.userID, session, "call.signal", input.Signal)
	default:
		return fmt.Errorf("unknown event type %q", input.Type)
	}
}

// publishCall converts a domain session into the stable browser protocol and
// sends it through Redis so caller and callee receive the same event shape.
func (h *Handler) publishCall(ctx context.Context, conversationID, senderID int64, session domain.Session, eventType string, signal *Signal) error {
	event := Event{Type: eventType, ConversationID: conversationID, SenderID: senderID, CallID: session.CallID, CallerID: session.CallerID, CalleeID: session.CalleeID, CallType: string(session.CallType), Status: string(session.Status), RecipientIDs: []int64{session.CallerID, session.CalleeID}, Signal: signal}
	return h.Hub.broadcast(ctx, conversationID, event)
}

// publicError avoids exposing database/Redis internals while retaining useful
// validation messages for the browser UI.
func publicError(err error) string {
	if errors.Is(err, authz.ErrForbidden) {
		return "операция запрещена"
	}
	return err.Error()
}

// Health is a lightweight readiness/liveness endpoint for Compose and probes.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
