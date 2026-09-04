package httptransport

import (
	"net/http"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/profiles/internal/application"
)

// SetHandle назначает текущему пользователю публичный handle.
// @Summary Назначить handle профиля
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body setHandleRequest true "Новый handle"
// @Success 200 {object} profileResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/profiles/me/handle [put]
func (h *Handler) SetHandle(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}

	var request setHandleRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	profile, err := h.profiles.SetHandle(r.Context(), userID, request.Handle)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfile(w, http.StatusOK, profile)
}

// UpdatePublicProfile изменяет отображаемое имя и описание текущего профиля.
// @Summary Обновить публичный профиль
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body updateProfileRequest true "Публичные поля профиля"
// @Success 200 {object} profileResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/profiles/me [patch]
func (h *Handler) UpdatePublicProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}

	var request updateProfileRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	profile, err := h.profiles.UpdatePublicProfile(r.Context(), userID, application.UpdateProfileInput{
		DisplayName: request.DisplayName,
		Bio:         request.Bio,
	})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfile(w, http.StatusOK, profile)
}

type setHandleRequest struct {
	Handle string `json:"handle"`
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
}
