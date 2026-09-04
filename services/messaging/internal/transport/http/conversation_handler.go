package httptransport

import (
	"net/http"
	"strconv"
	"uuid"

	"general-project/libs/platform/httpx"
	"general-project/messaging/internal/application"
)

type directConversationRequest struct {
	OtherUserID string `json:"other_user_id"`
}

// GetOrCreateDirect возвращает или создаёт прямой диалог.
// @Summary Получить прямой диалог
// @Tags messaging
// @Accept json
// @Produce json
// @Param request body directConversationRequest true "Второй пользователь"
// @Success 200 {object} conversationResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/messaging/conversations/direct [post]
func (h *Handler) GetOrCreateDirect(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	var request directConversationRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	otherUserID, err := uuid.Parse(request.OtherUserID)
	if err != nil {
		writeMessagingError(w, application.ErrValidation)
		return
	}
	conversation, err := h.messages.GetOrCreateDirect(r.Context(), userID, otherUserID)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	writeConversation(w, http.StatusOK, conversation)
}

// ListConversations возвращает доступные текущему пользователю диалоги.
// @Summary Список диалогов
// @Tags messaging
// @Produce json
// @Success 200 {object} conversationsResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/messaging/conversations [get]
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	archived := false
	if value := r.URL.Query().Get("archived"); value != "" {
		parsed, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			writeMessagingError(w, application.ErrValidation)
			return
		}
		archived = parsed
	}
	conversations, err := h.messages.ListConversations(r.Context(), userID, archived)
	if err != nil {
		writeMessagingError(w, err)
		return
	}
	writeConversations(w, http.StatusOK, conversations)
}

// ArchiveConversation архивирует диалог текущего пользователя.
// @Summary Архивировать диалог
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/archive [patch]
func (h *Handler) ArchiveConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationArchived(w, r, true)
}

// UnarchiveConversation возвращает диалог из архива.
// @Summary Вернуть диалог из архива
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/unarchive [patch]
func (h *Handler) UnarchiveConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationArchived(w, r, false)
}

func (h *Handler) setConversationArchived(w http.ResponseWriter, r *http.Request, archived bool) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseConversationID(w, r)
	if !ok {
		return
	}
	if err := h.messages.ArchiveConversation(r.Context(), userID, conversationID, archived); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HideConversation скрывает диалог только у текущего пользователя.
// @Summary Скрыть диалог
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID} [delete]
func (h *Handler) HideConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseConversationID(w, r)
	if !ok {
		return
	}
	if err := h.messages.HideConversation(r.Context(), userID, conversationID); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkConversationUnread сбрасывает отметку прочтения диалога.
// @Summary Пометить диалог непрочитанным
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/unread [post]
func (h *Handler) MarkConversationUnread(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseConversationID(w, r)
	if !ok {
		return
	}
	if err := h.messages.MarkConversationUnread(r.Context(), userID, conversationID); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PinConversation закрепляет диалог.
// @Summary Закрепить диалог
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/pin [patch]
func (h *Handler) PinConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationPinned(w, r, true)
}

// UnpinConversation открепляет диалог.
// @Summary Открепить диалог
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/unpin [patch]
func (h *Handler) UnpinConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationPinned(w, r, false)
}

func (h *Handler) setConversationPinned(w http.ResponseWriter, r *http.Request, pinned bool) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseConversationID(w, r)
	if !ok {
		return
	}
	if err := h.messages.SetConversationPinned(r.Context(), userID, conversationID, pinned); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MuteConversation отключает уведомления диалога.
// @Summary Отключить уведомления диалога
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/mute [patch]
func (h *Handler) MuteConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationMuted(w, r, true)
}

// UnmuteConversation включает уведомления диалога.
// @Summary Включить уведомления диалога
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/unmute [patch]
func (h *Handler) UnmuteConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationMuted(w, r, false)
}

func (h *Handler) setConversationMuted(w http.ResponseWriter, r *http.Request, muted bool) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseConversationID(w, r)
	if !ok {
		return
	}
	if err := h.messages.SetConversationMuted(r.Context(), userID, conversationID, muted); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearHistory очищает историю диалога для текущего пользователя.
// @Summary Очистить историю диалога
// @Tags messaging
// @Param conversationID path string true "UUID диалога"
// @Success 204
// @Router /v1/messaging/conversations/{conversationID}/history [delete]
func (h *Handler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	conversationID, ok := parseConversationID(w, r)
	if !ok {
		return
	}
	if err := h.messages.ClearConversationHistory(r.Context(), userID, conversationID); err != nil {
		writeMessagingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseConversationID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return parseUUIDParam(w, r, "conversationID", "conversation")
}
