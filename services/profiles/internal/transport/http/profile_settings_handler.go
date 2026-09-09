package httptransport

import (
	"net/http"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/profiles/internal/application"
	"general-project/profiles/internal/domain"
)

// SetHandle назначает текущему пользователю публичный handle.










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

type setHandleRequest = SetHandleRequest
type updateProfileRequest = UpdateProfileRequest
type privacyRequest = PrivacyRequest

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
