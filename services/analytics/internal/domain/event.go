// Package domain содержит сущности analytics-сервиса.
package domain

import (
	"time"
	"uuid"
)

// Event представляет версионируемое интеграционное событие.
type Event struct {
	ID            uuid.UUID
	EventType     string
	AggregateID   *uuid.UUID
	Payload       []byte
	CorrelationID string
	Attempts      int
	AvailableAt   time.Time
	CreatedAt     time.Time
}
