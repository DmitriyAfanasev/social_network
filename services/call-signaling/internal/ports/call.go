// Package ports содержит интерфейсы, которыми владеет application-слой calls.
package ports

import (
	"context"
	"errors"
	"uuid"

	"general-project/call-signaling/internal/domain"
)

var (
	// ErrNotFound означает, что короткоживущая сессия не найдена.
	ErrNotFound = errors.New("call session not found")
)

// SessionStore хранит короткоживущие состояния звонков.
type SessionStore interface {
	Create(context.Context, domain.Session) error
	Get(context.Context, uuid.UUID) (domain.Session, error)
	Refresh(context.Context, uuid.UUID) error
	Transition(context.Context, uuid.UUID, uuid.UUID, string) (domain.Session, error)
}

// CallAuthorizer проверяет принадлежность пользователя диалогу и звонку.
type CallAuthorizer interface {
	CanSubscribe(context.Context, uuid.UUID, uuid.UUID) error
	CanStart(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	CanUseCall(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error
}

// ReadinessChecker проверяет доступность обязательной зависимости сервиса.
type ReadinessChecker interface {
	Check(context.Context) error
}

// SignalingEvent содержит payload для локальной доставки в conversation room.
type SignalingEvent struct {
	ConversationID uuid.UUID
	Payload        []byte
}

// EventBroker публикует signaling-события между экземплярами сервиса.
type EventBroker interface {
	Publish(context.Context, SignalingEvent) error
	Subscribe(context.Context, func(SignalingEvent) error) error
}
