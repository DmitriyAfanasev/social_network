package application

import (
	"time"
	"uuid"

	"general-project/identity/internal/domain"
)

// UserDTO представляет безопасный результат application-сценария чтения пользователя.
type UserDTO struct {
	ID         uuid.UUID
	Email      string
	Status     string
	LastSeenAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AuthDTO содержит DTO пользователя и выданный токен доступа.
type AuthDTO struct {
	User             UserDTO
	AccessToken      string
	TokenType        string
	ExpiresIn        int64
	RefreshToken     string
	RefreshExpiresIn int64
}

// RegistrationDTO содержит результат создания неподтверждённой учётной записи.
type RegistrationDTO struct {
	User    UserDTO
	Message string
}

// MessageDTO содержит безопасный результат операции, не раскрывающий наличие email.
type MessageDTO struct {
	Message string
}

func toUserDTO(user domain.User) UserDTO {
	return UserDTO{
		ID:         user.ID,
		Email:      user.Email,
		Status:     user.Status,
		LastSeenAt: user.LastSeenAt,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}
