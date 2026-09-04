package httptransport

import (
	"errors"
	"net/http"

	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/social/internal/application"
)

func parseTargetID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "targetID"))
}

func parseRequestID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "requestID"))
}

func authenticatedUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return uuid.Nil(), false
	}
	return userID, true
}

func writeSocialError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrValidation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "недопустимая социальная операция")
	case errors.Is(err, application.ErrBlocked):
		httpx.WriteError(w, http.StatusConflict, "interaction_blocked", "взаимодействие запрещено блокировкой")
	case errors.Is(err, application.ErrFriendRequestNotFound):
		httpx.WriteError(w, http.StatusNotFound, "friend_request_not_found", "заявка не найдена")
	case errors.Is(err, application.ErrFriendRequestForbidden):
		httpx.WriteError(w, http.StatusForbidden, "friend_request_forbidden", "операция недоступна для этого пользователя")
	case errors.Is(err, application.ErrFriendRequestState):
		httpx.WriteError(w, http.StatusConflict, "friend_request_state_conflict", "заявка уже обработана")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера")
	}
}
