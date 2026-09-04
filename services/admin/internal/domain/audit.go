// Package domain содержит сущности admin-сервиса.
package domain

import (
	"time"
	"uuid"
)

// AuditLog представляет неизменяемую запись действия модерации.
type AuditLog struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	ActorID       *uuid.UUID
	Action        string
	TargetType    string
	TargetID      *uuid.UUID
	Details       []byte
	CorrelationID string
	CreatedAt     time.Time
}
