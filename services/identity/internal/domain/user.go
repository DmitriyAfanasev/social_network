// Package domain содержит сущности и правила identity-сервиса.
package domain

import (
	"time"

	"uuid"
)

const (
	// UserStatusPending означает, что email пользователя ещё не подтверждён.
	UserStatusPending = "pending"
	// UserStatusActive означает, что пользователь может пройти аутентификацию.
	UserStatusActive = "active"
	// UserStatusBlocked означает, что пользователь заблокирован.
	UserStatusBlocked = "blocked"
)

// User описывает пользователя внутри доменного слоя identity.
type User struct {
	ID         uuid.UUID
	Email      string
	Status     string
	LastSeenAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AuthUser объединяет пользователя и его секрет, необходимый только для входа.
type AuthUser struct {
	User         User
	PasswordHash string
}
