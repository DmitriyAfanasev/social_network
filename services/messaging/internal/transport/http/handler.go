// Package httptransport содержит HTTP transport messaging-сервиса.
package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/messaging/internal/application"
	"general-project/messaging/internal/ports"
)

// Handler содержит HTTP-обработчики messaging-сервиса.
type Handler struct {
	readiness     ports.ReadinessChecker
	messages      *application.MessageService
	presence      ports.PresenceStore
	verifier      ports.AccessTokenVerifier
	notifications ports.NotificationStream
}

// NewHandler создаёт HTTP-обработчик messaging-сервиса.
func NewHandler(readiness ports.ReadinessChecker, messages *application.MessageService, presence ports.PresenceStore, verifier ports.AccessTokenVerifier, notifications ports.NotificationStream) *Handler {
	return &Handler{readiness: readiness, messages: messages, presence: presence, verifier: verifier, notifications: notifications}
}

// Health возвращает состояние процесса без проверки внешних зависимостей.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

// Ready проверяет доступность PostgreSQL перед приёмом трафика.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.readiness.Check(r.Context()); err != nil {
		writeStatus(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	writeStatus(w, http.StatusOK, "ready")
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": value})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) authenticatedUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return uuid.Nil(), false
	}
	return userID, true
}

func writeMessagingError(w http.ResponseWriter, err error) {
	status, code, message := messagingErrorStatus(err)
	httpx.WriteError(w, status, code, message)
}

func messagingErrorStatus(err error) (int, string, string) {
	switch {
	case errors.Is(err, application.ErrValidation):
		return http.StatusUnprocessableEntity, "validation_error", "данные сообщения некорректны"
	case errors.Is(err, application.ErrInteractionForbidden):
		return http.StatusForbidden, "message_forbidden", "политика профиля запрещает отправлять этому пользователю сообщения"
	case errors.Is(err, ports.ErrForbidden):
		return http.StatusForbidden, "forbidden", "операция недоступна"
	case errors.Is(err, ports.ErrNotFound):
		return http.StatusNotFound, "messaging_resource_not_found", "диалог или сообщение не найдены"
	default:
		return http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера"
	}
}
