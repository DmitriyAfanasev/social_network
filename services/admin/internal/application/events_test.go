package application

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/admin/internal/domain"
)

type fakeEventAuditRepository struct {
	recorded []domain.AuditLog
}

func (f *fakeEventAuditRepository) Record(_ context.Context, log domain.AuditLog) error {
	f.recorded = append(f.recorded, log)
	return nil
}

func (f *fakeEventAuditRepository) List(_ context.Context, _ int) ([]domain.AuditLog, error) {
	return f.recorded, nil
}

func TestEventHandlerMapsBlockEventToAudit(t *testing.T) {
	repository := &fakeEventAuditRepository{}
	handler := NewEventHandler(NewAuditService(repository))
	eventID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()
	payload := []byte(`{"blocker_id":"` + actorID.String() + `","blocked_id":"` + targetID.String() + `"}`)
	createdAt := time.Now().UTC()

	err := handler.Handle(context.Background(), eventID, "social.block.created", "request-123", payload, createdAt)

	require.NoError(t, err)
	require.Len(t, repository.recorded, 1)
	require.Equal(t, eventID, repository.recorded[0].EventID)
	require.Equal(t, actorID, *repository.recorded[0].ActorID)
	require.Equal(t, targetID, *repository.recorded[0].TargetID)
	require.Equal(t, "block", repository.recorded[0].Action)
	require.Equal(t, "request-123", repository.recorded[0].CorrelationID)
}

func TestEventHandlerIgnoresUnsupportedEvent(t *testing.T) {
	repository := &fakeEventAuditRepository{}
	handler := NewEventHandler(NewAuditService(repository))

	err := handler.Handle(context.Background(), uuid.New(), "content.post.created", "", []byte(`{}`), time.Now().UTC())

	require.NoError(t, err)
	require.Empty(t, repository.recorded)
}
