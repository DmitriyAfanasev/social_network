// Package ports содержит интерфейсы, которыми владеет application-слой.
package ports

import (
	"context"
	"errors"
	"time"

	"uuid"

	"general-project/identity/internal/domain"
)

var (
	// ErrNotFound означает, что запрошенная сущность отсутствует.
	ErrNotFound = errors.New("entity not found")
	// ErrAlreadyExists означает конфликт уникального ограничения.
	ErrAlreadyExists = errors.New("entity already exists")
)

// UserRepository предоставляет application-слою доступ к пользователям.
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.AuthUser, error)
	Create(ctx context.Context, user domain.User, passwordHash string) (domain.User, error)
	Activate(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	ListPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}

// PasswordHasher скрывает алгоритм хранения и проверки паролей.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password string, passwordHash string) error
}

// TokenIssuer выпускает токен доступа для пользователя.
type TokenIssuer interface {
	IssueAccess(userID uuid.UUID) (string, error)
}

// RefreshTokenStore хранит, ротирует и отзывает refresh-токены.
type RefreshTokenStore interface {
	Save(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	Rotate(ctx context.Context, tokenHash string, replacementHash string, replacementExpiresAt time.Time) (uuid.UUID, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

// VerificationTokenStore хранит одноразовые токены подтверждения и сброса пароля.
type VerificationTokenStore interface {
	Save(ctx context.Context, userID uuid.UUID, purpose string, tokenHash string, expiresAt time.Time) error
	Consume(ctx context.Context, purpose string, tokenHash string) (uuid.UUID, error)
}

// Notification описывает безопасное уведомление с одноразовой ссылкой.
type Notification struct {
	Purpose string
	Email   string
	Token   string
	BaseURL string
}

// NotificationSender отправляет пользователю confirmation и password reset ссылки.
type NotificationSender interface {
	Send(ctx context.Context, notification Notification) error
}
