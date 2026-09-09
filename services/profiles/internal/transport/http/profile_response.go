package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"general-project/libs/platform/httpx"
	"general-project/profiles/internal/application"
	"general-project/profiles/internal/ports"
)

type profileResponse = ProfileResponse
type privacyResponse = PrivacyResponse
type profilesResponse = ProfilesResponse

func writeProfile(w http.ResponseWriter, status int, profile application.ProfileDTO) { //nolint:unparam // status is part of the response helper contract.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapProfileResponse(profile))
}
func mapProfileResponse(profile application.ProfileDTO) profileResponse {
	return profileResponse{
		UserID:      profile.UserID.String(),
		Handle:      profile.Handle,
		Bio:         profile.Bio,
		AvatarURL:   profile.AvatarURL,
		FirstName:   profile.Details.FirstName,
		LastName:    profile.Details.LastName,
		MiddleName:  profile.Details.MiddleName,
		BirthDate:   profile.Details.BirthDate,
		Gender:      profile.Details.Gender,
		PhoneNumber: profile.Details.PhoneNumber,
		Country:     profile.Details.Country,
		City:        profile.Details.City,
		Street:      profile.Details.Street,
		Status:      profile.Details.Status,
		Privacy: privacyResponse{
			ProfileVisibility: profile.Privacy.ProfileVisibility, FriendRequestPolicy: profile.Privacy.FriendRequestPolicy,
			MessagePolicy: profile.Privacy.MessagePolicy, PhoneVisibility: profile.Privacy.PhoneVisibility,
			BirthDateVisibility: profile.Privacy.BirthDateVisibility, GenderVisibility: profile.Privacy.GenderVisibility,
			LocationVisibility: profile.Privacy.LocationVisibility, StatusVisibility: profile.Privacy.StatusVisibility,
			FriendsVisibility: profile.Privacy.FriendsVisibility, PostsVisibility: profile.Privacy.PostsVisibility,
			MusicVisibility: profile.Privacy.MusicVisibility,
		},
		CreatedAt: profile.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt: profile.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
	}
}
func profileErrorStatus(err error) (int, string, string) {
	switch {
	case errors.Is(err, application.ErrValidation):
		return http.StatusUnprocessableEntity, "validation_error", "данные профиля некорректны"
	case errors.Is(err, application.ErrConflict):
		return http.StatusConflict, "handle_conflict", "handle уже занят"
	case errors.Is(err, application.ErrMediaConflict):
		return http.StatusConflict, "media_conflict", "медиаобъект уже добавлен"
	case errors.Is(err, application.ErrForbidden):
		return http.StatusForbidden, "forbidden", "операция доступна только владельцу ресурса"
	case errors.Is(err, ports.ErrNotFound):
		return http.StatusNotFound, "profile_not_found", "профиль не найден"
	default:
		return http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера"
	}
}
func writeProfileError(w http.ResponseWriter, err error) {
	status, code, message := profileErrorStatus(err)
	httpx.WriteError(w, status, code, message)
}
