package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"general-project/identity/internal/application"
	platformauth "general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
)

// Register принимает данные нового пользователя и отправляет confirmation-ссылку.
// @Summary Регистрация пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body registerRequest true "Данные регистрации"
// @Success 202 {object} messageResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
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
// @Summary Повторно отправить подтверждение регистрации
// @Tags auth
// @Accept json
// @Produce json
// @Param request body emailRequest true "Email пользователя"
// @Success 202 {object} messageResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/auth/registration-confirmation-requests [post]
func (h *Handler) RequestRegistrationConfirmation(w http.ResponseWriter, r *http.Request) {
	var request emailRequest
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
// @Summary Подтвердить регистрацию
// @Tags auth
// @Accept json
// @Produce json
// @Param request body tokenRequest true "Confirmation-токен"
// @Success 200 {object} authResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/auth/confirm-registration [post]
func (h *Handler) ConfirmRegistration(w http.ResponseWriter, r *http.Request) {
	var request tokenRequest
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
// @Summary Запросить сброс пароля
// @Tags auth
// @Accept json
// @Produce json
// @Param request body emailRequest true "Email пользователя"
// @Success 202 {object} messageResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/auth/password-reset-requests [post]
func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var request emailRequest
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
// @Summary Сбросить пароль
// @Tags auth
// @Accept json
// @Produce json
// @Param request body resetPasswordRequest true "Reset-токен и новый пароль"
// @Success 200 {object} authResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/auth/password-reset [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request resetPasswordRequest
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
// @Summary Получить permissions пользователя
// @Tags auth
// @Produce json
// @Success 200 {object} permissionsResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/auth/me/permissions [get]
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
	_ = json.NewEncoder(w).Encode(permissionsResponse{Permissions: permissions})
}

// Login проверяет credentials и выдает access-токен.
// @Summary Вход пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body loginRequest true "Данные входа"
// @Success 200 {object} authResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
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
// @Summary Обновить токены
// @Tags auth
// @Accept json
// @Produce json
// @Param request body refreshRequest true "Refresh-токен"
// @Success 200 {object} authResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
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
// @Summary Завершить сессию
// @Tags auth
// @Accept json
// @Param request body refreshRequest true "Refresh-токен"
// @Success 204
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var request refreshRequest
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

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type emailRequest struct {
	Email string `json:"email"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type messageResponse struct {
	Message string `json:"message"`
}

type permissionsResponse struct {
	Permissions []string `json:"permissions"`
}

type authResponse struct {
	User             userResponse `json:"user"`
	AccessToken      string       `json:"access_token"`
	TokenType        string       `json:"token_type"`
	ExpiresIn        int64        `json:"expires_in"`
	RefreshToken     string       `json:"refresh_token"`
	RefreshExpiresIn int64        `json:"refresh_expires_in"`
}

type userResponse struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Status     string     `json:"status"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeAuth(w http.ResponseWriter, status int, result application.AuthDTO) {
	response := authResponse{
		User: userResponse{
			ID:         result.User.ID.String(),
			Email:      result.User.Email,
			Status:     result.User.Status,
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
	_ = json.NewEncoder(w).Encode(messageResponse{Message: result.Message})
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
