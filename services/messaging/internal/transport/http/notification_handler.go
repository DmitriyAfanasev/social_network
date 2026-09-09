package httptransport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
)

// StreamNotifications открывает SSE-поток realtime-уведомлений пользователя.
// Для EventSource поддерживается access_token в query; обычные клиенты могут
// передать Bearer-токен в Authorization.
func (h *Handler) StreamNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.notificationUser(w, r)
	if !ok {
		return
	}
	if h.notifications == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "notifications_unavailable", "поток уведомлений недоступен")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "stream_unsupported", "сервер не поддерживает потоковую выдачу")
		return
	}
	channel, unsubscribe := h.notifications.Subscribe(userID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "retry: 3000\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case payload, open := <-channel:
			if !open {
				return
			}
			name := notificationEventName(payload)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
func (h *Handler) notificationUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	token := strings.TrimSpace(r.URL.Query().Get("access_token"))
	if token == "" {
		var ok bool
		token, ok = auth.BearerToken(r.Header.Get("Authorization"))
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
			return uuid.Nil(), false
		}
	}
	if h.verifier == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "auth_unavailable", "аутентификация недоступна")
		return uuid.Nil(), false
	}
	userID, err := h.verifier.UserID(token)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return uuid.Nil(), false
	}
	return userID, true
}
func notificationEventName(payload []byte) string {
	var envelope struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(payload, &envelope) != nil || envelope.Type == "" || strings.ContainsAny(envelope.Type, "\r\n") {
		return "notification"
	}
	return envelope.Type
}
