package application

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"general-project/admin/internal/domain"
)

// EventHandler переносит moderation-события в admin audit read model.
type EventHandler struct {
	audit *AuditService
}

// NewEventHandler создаёт обработчик moderation-событий.
func NewEventHandler(audit *AuditService) *EventHandler {
	return &EventHandler{audit: audit}
}

// Handle сохраняет поддерживаемые moderation-события и пропускает остальные.
func (h *EventHandler) Handle(ctx context.Context, eventID uuid.UUID, eventType string, correlationID string, payload []byte, createdAt time.Time) error {
	if eventType != "social.block.created" {
		return nil
	}
	var data struct {
		BlockerID uuid.UUID `json:"blocker_id"`
		BlockedID uuid.UUID `json:"blocked_id"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}
	if data.BlockerID == uuid.Nil() || data.BlockedID == uuid.Nil() {
		return ErrValidation
	}
	details, err := json.Marshal(map[string]any{"source_event_type": eventType, "payload": json.RawMessage(payload)})
	if err != nil {
		return err
	}
	return h.audit.Record(ctx, domain.AuditLog{ID: uuid.New(), EventID: eventID, ActorID: &data.BlockerID, Action: "block", TargetType: "user", TargetID: &data.BlockedID, Details: details, CorrelationID: correlationID, CreatedAt: createdAt})
}
