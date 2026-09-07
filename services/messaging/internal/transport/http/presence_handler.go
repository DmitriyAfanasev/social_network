package httptransport

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/libs/platform/httpx"
)

type presenceResponse struct {
	UserID string `json:"user_id"`
	Online bool   `json:"online"`
}

type presenceListResponse struct {
	Presence []presenceResponse `json:"presence"`
}

// Heartbeat отмечает текущего пользователя активным независимо от открытого чата.
// @Summary Обновить presence текущего пользователя
// @Tags messaging
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/messaging/presence/heartbeat [post]
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	if h.presence == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "presence недоступен")
		return
	}
	if err := h.presence.Heartbeat(r.Context(), userID, 30*time.Second); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "не удалось обновить presence")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetPresenceBatch возвращает online-состояние нескольких пользователей одним запросом.
// @Summary Проверить presence списка пользователей
// @Tags messaging
// @Produce json
// @Param user_ids query string true "UUID пользователей через запятую"
// @Success 200 {object} presenceListResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/messaging/presence [get]
func (h *Handler) GetPresenceBatch(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticatedUser(w, r); !ok {
		return
	}
	if h.presence == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "presence недоступен")
		return
	}
	values := r.URL.Query()["user_ids"]
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{})
	for _, value := range values {
		for _, rawID := range strings.Split(value, ",") {
			rawID = strings.TrimSpace(rawID)
			if rawID == "" {
				continue
			}
			userID, err := uuid.Parse(rawID)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "invalid_user_id", "некорректный UUID пользователя")
				return
			}
			if _, exists := seen[userID]; exists {
				continue
			}
			seen[userID] = struct{}{}
			ids = append(ids, userID)
		}
	}
	if len(ids) == 0 || len(ids) > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_user_ids", "нужно передать от 1 до 100 UUID пользователей")
		return
	}
	response := presenceListResponse{Presence: make([]presenceResponse, 0, len(ids))}
	for _, userID := range ids {
		online, err := h.presence.IsOnline(r.Context(), userID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "не удалось проверить presence")
			return
		}
		response.Presence = append(response.Presence, presenceResponse{UserID: userID.String(), Online: online})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
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
