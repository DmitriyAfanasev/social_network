package httptransport

import (
	"net/http"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/profiles/internal/application"
	"general-project/profiles/internal/domain"
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

// UpdatePublicProfile изменяет описание и данные текущего профиля.
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
	profile, err := h.profiles.UpdateProfile(r.Context(), userID, application.UpdateProfileInput{
		Bio: request.Bio,
		Details: domain.ProfileDetails{
			FirstName: request.FirstName, LastName: request.LastName, MiddleName: request.MiddleName,
			BirthDate: request.BirthDate, Gender: request.Gender, PhoneNumber: request.PhoneNumber,
			Country: request.Country, City: request.City, Street: request.Street, Status: request.Status,
		},
	})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfile(w, http.StatusOK, profile)
}

// GetPrivacy возвращает настройки приватности текущего пользователя.
// @Summary Получить настройки приватности
// @Tags profiles
// @Produce json
// @Success 200 {object} privacyResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/profiles/me/privacy [get]
func (h *Handler) GetPrivacy(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	privacy, err := h.profiles.GetPrivacy(r.Context(), userID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapPrivacyResponse(privacy))
}

// UpdatePrivacy сохраняет настройки приватности текущего пользователя.
// @Summary Обновить настройки приватности
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body privacyRequest true "Настройки приватности"
// @Success 200 {object} privacyResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/profiles/me/privacy [put]
func (h *Handler) UpdatePrivacy(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	var request privacyRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	privacy, err := h.profiles.UpdatePrivacy(r.Context(), userID, application.UpdatePrivacyInput{Privacy: domain.ProfilePrivacy{
		ProfileVisibility: request.ProfileVisibility, FriendRequestPolicy: request.FriendRequestPolicy,
		MessagePolicy: request.MessagePolicy, PhoneVisibility: request.PhoneVisibility,
		BirthDateVisibility: request.BirthDateVisibility, GenderVisibility: request.GenderVisibility,
		LocationVisibility: request.LocationVisibility, StatusVisibility: request.StatusVisibility,
		FriendsVisibility: request.FriendsVisibility, PostsVisibility: request.PostsVisibility,
		MusicVisibility: request.MusicVisibility,
	}})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapPrivacyResponse(privacy))
}

type setHandleRequest struct {
	Handle string `json:"handle"`
}

type updateProfileRequest struct {
	Bio         string `json:"bio"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	MiddleName  string `json:"middle_name"`
	BirthDate   string `json:"birth_date"`
	Gender      string `json:"gender"`
	PhoneNumber string `json:"phone_number"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Street      string `json:"street"`
	Status      string `json:"status"`
}

type privacyRequest struct {
	ProfileVisibility   string `json:"profile_visibility"`
	FriendRequestPolicy string `json:"friend_request_policy"`
	MessagePolicy       string `json:"message_policy"`
	PhoneVisibility     string `json:"phone_visibility"`
	BirthDateVisibility string `json:"birth_date_visibility"`
	GenderVisibility    string `json:"gender_visibility"`
	LocationVisibility  string `json:"location_visibility"`
	StatusVisibility    string `json:"status_visibility"`
	FriendsVisibility   string `json:"friends_visibility"`
	PostsVisibility     string `json:"posts_visibility"`
	MusicVisibility     string `json:"music_visibility"`
}

func mapPrivacyResponse(privacy application.ProfilePrivacyDTO) privacyResponse {
	return privacyResponse{
		ProfileVisibility: privacy.ProfileVisibility, FriendRequestPolicy: privacy.FriendRequestPolicy,
		MessagePolicy: privacy.MessagePolicy, PhoneVisibility: privacy.PhoneVisibility,
		BirthDateVisibility: privacy.BirthDateVisibility, GenderVisibility: privacy.GenderVisibility,
		LocationVisibility: privacy.LocationVisibility, StatusVisibility: privacy.StatusVisibility,
		FriendsVisibility: privacy.FriendsVisibility, PostsVisibility: privacy.PostsVisibility,
		MusicVisibility: privacy.MusicVisibility,
	}
}
