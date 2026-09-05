package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"general-project/libs/platform/httpx"
	"general-project/profiles/internal/application"
	"general-project/profiles/internal/ports"
)

type profileResponse struct {
	UserID      string          `json:"user_id"`
	Handle      *string         `json:"handle,omitempty"`
	DisplayName string          `json:"display_name"`
	Bio         string          `json:"bio"`
	AvatarURL   string          `json:"avatar_url"`
	FirstName   string          `json:"first_name"`
	LastName    string          `json:"last_name"`
	MiddleName  string          `json:"middle_name"`
	BirthDate   string          `json:"birth_date,omitempty"`
	Gender      string          `json:"gender,omitempty"`
	PhoneNumber string          `json:"phone_number,omitempty"`
	Country     string          `json:"country,omitempty"`
	City        string          `json:"city,omitempty"`
	Street      string          `json:"street,omitempty"`
	Status      string          `json:"status,omitempty"`
	Privacy     privacyResponse `json:"privacy"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type privacyResponse struct {
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

type profilesResponse struct {
	Profiles []profileResponse `json:"profiles"`
}

func writeProfile(w http.ResponseWriter, status int, profile application.ProfileDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapProfileResponse(profile))
}

func mapProfileResponse(profile application.ProfileDTO) profileResponse {
	return profileResponse{
		UserID:      profile.UserID.String(),
		Handle:      profile.Handle,
		DisplayName: profile.DisplayName,
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
