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
	UserID      string  `json:"user_id"`
	Handle      *string `json:"handle,omitempty"`
	DisplayName string  `json:"display_name"`
	Bio         string  `json:"bio"`
	AvatarURL   string  `json:"avatar_url"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
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
		CreatedAt:   profile.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt:   profile.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
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
