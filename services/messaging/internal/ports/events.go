package ports

import (
	"encoding/json"
	"time"
	"uuid"
)

// IntegrationEvent описывает событие из общего Kafka topic на границе messaging.
type IntegrationEvent struct {
	ID            uuid.UUID
	EventType     string
	AggregateID   *uuid.UUID
	Payload       json.RawMessage
	CorrelationID string
	CreatedAt     time.Time
}
