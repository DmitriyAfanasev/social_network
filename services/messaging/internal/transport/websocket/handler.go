package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
	"uuid"

	"github.com/gorilla/websocket"

	platformauth "general-project/libs/platform/auth"
	"general-project/messaging/internal/application"
	"general-project/messaging/internal/ports"
)

const (
	messageSendEvent  = "message.send"
	messageReadEvent  = "message.read"
	messageErrorEvent = "message.error"
)

// Handler обслуживает realtime-соединения сообщений.
type Handler struct {
	messages *application.MessageService
	verifier ports.AccessTokenVerifier
	hub      *Hub
	presence ports.PresenceStore
}

// NewHandler создаёт WebSocket-обработчик messaging-сервиса.
func NewHandler(messages *application.MessageService, verifier ports.AccessTokenVerifier, hub *Hub, presence ports.PresenceStore) *Handler {
	return &Handler{messages: messages, verifier: verifier, hub: hub, presence: presence}
}

// ServeHTTP аутентифицирует WebSocket и обрабатывает события сообщений.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if token, ok := platformauth.BearerToken(r.Header.Get("Authorization")); ok {
		r = r.WithContext(platformauth.ContextWithAccessToken(r.Context(), token))
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: sameOrigin}
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	item := h.hub.add(userID, connection)
	presenceCtx, cancelPresence := context.WithCancel(context.Background())
	defer cancelPresence()
	if h.presence != nil {
		go h.heartbeat(presenceCtx, userID)
	}
	defer func() {
		h.hub.remove(userID, item)
		_ = connection.Close()
	}()

	connection.SetReadLimit(1 << 20)
	for {
		var event incomingEvent
		if err := connection.ReadJSON(&event); err != nil {
			return
		}
		if err := h.handleEvent(r, userID, event); err != nil {
			payload, marshalErr := json.Marshal(map[string]any{"type": messageErrorEvent, "message": err.Error()})
			if marshalErr != nil || !item.enqueue(payload) {
				return
			}
		}
	}
}

type incomingEvent struct {
	Type           string  `json:"type"`
	ConversationID string  `json:"conversation_id"`
	MessageID      string  `json:"message_id"`
	Body           string  `json:"body"`
	MediaID        *string `json:"media_id,omitempty"`
}

func (h *Handler) handleEvent(r *http.Request, userID uuid.UUID, event incomingEvent) error {
	conversationID, err := uuid.Parse(event.ConversationID)
	if err != nil {
		return application.ErrValidation
	}
	switch event.Type {
	case messageSendEvent:
		var mediaID *uuid.UUID
		if event.MediaID != nil {
			parsed, parseErr := uuid.Parse(*event.MediaID)
			if parseErr != nil {
				return application.ErrValidation
			}
			mediaID = &parsed
		}
		_, _, sendErr := h.messages.SendMessage(r.Context(), userID, conversationID, application.SendMessageInput{Body: event.Body, MediaID: mediaID})
		if sendErr != nil {
			return sendErr
		}
		return nil
	case messageReadEvent:
		messageID, parseErr := uuid.Parse(event.MessageID)
		if parseErr != nil {
			return application.ErrValidation
		}
		if err := h.messages.MarkRead(r.Context(), userID, conversationID, messageID); err != nil {
			return err
		}
		return nil
	default:
		return application.ErrValidation
	}
}

func (h *Handler) heartbeat(ctx context.Context, userID uuid.UUID) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		_ = h.presence.Heartbeat(ctx, userID, 30*time.Second)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
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

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host == r.Host
}
