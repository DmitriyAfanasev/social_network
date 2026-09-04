package httptransport

import (
	"net/http"
	"strconv"
	"uuid"

	"general-project/libs/platform/httpx"
	"general-project/messaging/internal/application"
	"github.com/go-chi/chi/v5"
)

type messageRequest struct {
	Body    string  `json:"body"`
	MediaID *string `json:"media_id,omitempty"`
}

// ListMessages возвращает страницу сообщений участника диалога.
// @Summary Получить сообщения
// @Tags messaging
// @Produce json
// @Param conversationID path string true "UUID диалога"
// @Param offset query int false "Смещение"
// @Param limit query int false "Размер страницы" default(50)
// @Success 200 {object} messagesResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/messaging/conversations/{conversationID}/messages [get]
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseUUIDParam(w, r, "conversationID", "conversation")
	if !ok {
		return
	}
	offset, limit, ok := pagination(w, r)
	if !ok {
		return
	}
	messages, hasMore, err := h.messages.ListMessages(r.Context(), userID, conversationID, offset, limit)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	writeMessages(w, http.StatusOK, messages, hasMore)
}

// SendMessage отправляет сообщение участником диалога.
// @Summary Отправить сообщение
// @Tags messaging
// @Accept json
// @Produce json
// @Param conversationID path string true "UUID диалога"
// @Param request body messageRequest true "Сообщение"
// @Success 201 {object} messageResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/messaging/conversations/{conversationID}/messages [post]
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseUUIDParam(w, r, "conversationID", "conversation")
	if !ok {
		return
	}
	request, err := decodeMessageRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	message, _, err := h.messages.SendMessage(r.Context(), userID, conversationID, request)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	writeMessage(w, http.StatusCreated, message)
}

// UpdateMessage изменяет сообщение его автором.
// @Summary Изменить сообщение
// @Tags messaging
// @Accept json
// @Produce json
// @Param messageID path string true "UUID сообщения"
// @Param request body messageRequest true "Новый текст"
// @Success 200 {object} messageResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/messaging/messages/{messageID} [patch]
func (h *Handler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	messageID, ok := parseUUIDParam(w, r, "messageID", "message")
	if !ok {
		return
	}
	request, err := decodeMessageRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	message, err := h.messages.UpdateMessage(r.Context(), userID, messageID, request)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	writeMessage(w, http.StatusOK, message)
}

// DeleteMessage помечает сообщение удалённым.
// @Summary Удалить сообщение
// @Tags messaging
// @Param messageID path string true "UUID сообщения"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/messaging/messages/{messageID} [delete]
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	messageID, ok := parseUUIDParam(w, r, "messageID", "message")
	if !ok {
		return
	}
	if err := h.messages.DeleteMessage(r.Context(), userID, messageID); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveMessageMedia удаляет ссылку на media из сообщения его автора.
// @Summary Удалить media сообщения
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Param messageID path string true "UUID сообщения"
// @Success 200 {object} messageResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/messaging/conversations/{conversationID}/messages/{messageID}/media [delete]
func (h *Handler) RemoveMessageMedia(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseUUIDParam(w, r, "conversationID", "conversation")
	if !ok {
		return
	}
	messageID, ok := parseUUIDParam(w, r, "messageID", "message")
	if !ok {
		return
	}
	message, err := h.messages.RemoveMessageMedia(r.Context(), userID, conversationID, messageID)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	writeMessage(w, http.StatusOK, message)
}

// MarkRead отмечает сообщение прочитанным.
// @Summary Отметить сообщение прочитанным
// @Tags messaging
// @Accept json
// @Param conversationID path string true "UUID диалога"
// @Param messageID path string true "UUID сообщения"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/messaging/conversations/{conversationID}/messages/{messageID}/read [post]
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseUUIDParam(w, r, "conversationID", "conversation")
	if !ok {
		return
	}
	messageID, ok := parseUUIDParam(w, r, "messageID", "message")
	if !ok {
		return
	}
	if err := h.messages.MarkRead(r.Context(), userID, conversationID, messageID); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeMessageRequest(r *http.Request) (application.SendMessageInput, error) {
	var request messageRequest
	if err := decodeJSON(r, &request); err != nil {
		return application.SendMessageInput{}, err
	}
	var mediaID *uuid.UUID
	if request.MediaID != nil {
		parsed, err := uuid.Parse(*request.MediaID)
		if err != nil {
			return application.SendMessageInput{}, err
		}
		mediaID = &parsed
	}
	return application.SendMessageInput{Body: request.Body, MediaID: mediaID}, nil
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, parameter string, resource string) (uuid.UUID, bool) {
	value, err := uuid.Parse(chi.URLParam(r, parameter))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_"+resource+"_id", "некорректный UUID ресурса")
		return uuid.Nil(), false
	}
	return value, true
}

func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	offset, limit := 0, 50
	var err error
	if value := r.URL.Query().Get("offset"); value != "" {
		offset, err = strconv.Atoi(value)
	}
	if err == nil {
		if value := r.URL.Query().Get("limit"); value != "" {
			limit, err = strconv.Atoi(value)
		}
	}
	if err != nil {
		writeMessagingError(w, application.ErrValidation)
		return 0, 0, false
	}
	return offset, limit, true
}
