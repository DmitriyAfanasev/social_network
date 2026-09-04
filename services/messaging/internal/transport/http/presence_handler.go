package httptransport

import (
	"encoding/json"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/libs/platform/httpx"
)

type presenceResponse struct {
	UserID string `json:"user_id"`
	Online bool   `json:"online"`
}

// GetPresence возвращает online-состояние пользователя по свежему heartbeat.
// @Summary Проверить presence пользователя
// @Tags messaging
// @Produce json
// @Param userID path string true "UUID пользователя"
// @Success 200 {object} presenceResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/messaging/presence/{userID} [get]
func (h *Handler) GetPresence(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticatedUser(w, r); !ok {
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_user_id", "некорректный UUID пользователя")
		return
	}
	if h.presence == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "presence недоступен")
		return
	}
	online, err := h.presence.IsOnline(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "не удалось проверить presence")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = writePresenceResponse(w, presenceResponse{UserID: userID.String(), Online: online})
}

func writePresenceResponse(w http.ResponseWriter, response presenceResponse) error {
	return json.NewEncoder(w).Encode(response)
}
