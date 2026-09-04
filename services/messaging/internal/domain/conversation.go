package domain

import (
	"time"
	"uuid"
)

// Conversation описывает прямой диалог между пользователями.
type Conversation struct {
	ID             uuid.UUID
	DirectKey      string
	ParticipantIDs []uuid.UUID
	Archived       bool
	Pinned         bool
	Muted          bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastMessage    *Message
}

// Message описывает сообщение без инфраструктурных зависимостей.
type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Body           string
	MediaID        *uuid.UUID
	CreatedAt      time.Time
	EditedAt       *time.Time
	DeletedAt      *time.Time
}

// PairKey создаёт стабильный ключ прямого диалога для двух UUID.
func PairKey(first uuid.UUID, second uuid.UUID) string {
	if first.String() > second.String() {
		first, second = second, first
	}
	return first.String() + ":" + second.String()
}
