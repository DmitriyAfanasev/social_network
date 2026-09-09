package httptransport

import (
	"net/http"
	"strconv"
	"uuid"

	"general-project/libs/platform/httpx"
	"general-project/messaging/internal/application"
)

type directConversationRequest = DirectConversationRequest

// GetOrCreateDirect возвращает или создаёт прямой диалог.









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





func (h *Handler) ArchiveConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationArchived(w, r, true)
}

// UnarchiveConversation возвращает диалог из архива.





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





func (h *Handler) PinConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationPinned(w, r, true)
}

// UnpinConversation открепляет диалог.





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





func (h *Handler) MuteConversation(w http.ResponseWriter, r *http.Request) {
	h.setConversationMuted(w, r, true)
}

// UnmuteConversation включает уведомления диалога.





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
