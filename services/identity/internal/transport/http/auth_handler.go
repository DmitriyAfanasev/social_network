package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"general-project/identity/internal/application"
	platformauth "general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
)

// Register принимает данные нового пользователя и отправляет confirmation-ссылку.










func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var request RegisterRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.Register(r.Context(), application.RegisterInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeRegistration(w, http.StatusAccepted, result)
}

// RequestRegistrationConfirmation повторно отправляет confirmation-ссылку.








func (h *Handler) RequestRegistrationConfirmation(w http.ResponseWriter, r *http.Request) {
	var request EmailRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.RequestRegistrationConfirmation(r.Context(), request.Email)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeMessage(w, http.StatusAccepted, result)
}

// ConfirmRegistration активирует пользователя по одноразовому токену.








func (h *Handler) ConfirmRegistration(w http.ResponseWriter, r *http.Request) {
	var request TokenRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.ConfirmRegistration(r.Context(), request.Token)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeAuth(w, http.StatusOK, result)
}

// RequestPasswordReset отправляет одноразовую ссылку для сброса пароля.








func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var request EmailRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.RequestPasswordReset(r.Context(), request.Email)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeMessage(w, http.StatusAccepted, result)
}

// ResetPassword меняет пароль по одноразовому reset-токену и выдаёт новую сессию.








func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request ResetPasswordRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.ResetPassword(r.Context(), application.PasswordResetInput{Token: request.Token, NewPassword: request.NewPassword})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeAuth(w, http.StatusOK, result)
}

// Permissions возвращает permissions текущего пользователя.






func (h *Handler) Permissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := platformauth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	permissions, err := h.auth.Permissions(r.Context(), userID)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(PermissionsResponse{Permissions: permissions})
}

// Login проверяет credentials и выдает access-токен.









func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.Login(r.Context(), application.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeAuth(w, http.StatusOK, result)
}

// Refresh ротирует refresh-токен и возвращает новую пару токенов.









func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var request RefreshRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	result, err := h.auth.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		writeApplicationError(w, err)
		return
	}
	writeAuth(w, http.StatusOK, result)
}

// Logout отзывает refresh-токен и завершает сессию пользователя.







func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var request RefreshRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	if err := h.auth.Logout(r.Context(), request.RefreshToken); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeAuth(w http.ResponseWriter, status int, result application.AuthDTO) {
	response := AuthResponse{
		User: UserResponse{
			Id:         result.User.ID.String(),
			Email:      result.User.Email,
			Status:     UserResponseStatus(result.User.Status),
			LastSeenAt: result.User.LastSeenAt,
			CreatedAt:  result.User.CreatedAt,
			UpdatedAt:  result.User.UpdatedAt,
		},
		AccessToken:      result.AccessToken,
		TokenType:        result.TokenType,
		ExpiresIn:        result.ExpiresIn,
		RefreshToken:     result.RefreshToken,
		RefreshExpiresIn: result.RefreshExpiresIn,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeRegistration(w http.ResponseWriter, status int, result application.RegistrationDTO) {
	writeMessage(w, status, application.MessageDTO{Message: result.Message})
}

func writeMessage(w http.ResponseWriter, status int, result application.MessageDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(MessageResponse{Message: result.Message})
}

func writeApplicationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrValidation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "данные не прошли проверку")
	case errors.Is(err, application.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "identity_conflict", "email уже используется")
	case errors.Is(err, application.ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "неверный email или пароль")
	case errors.Is(err, application.ErrInvalidToken):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh-токен недействителен")
	case errors.Is(err, application.ErrInvalidVerificationToken):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_verification_token", "одноразовый токен недействителен")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера")
	}
}
