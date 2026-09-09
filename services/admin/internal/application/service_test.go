package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/admin/internal/domain"
)

type fakeAuditRepository struct {
	recorded []domain.AuditLog
}

func (f *fakeAuditRepository) Record(_ context.Context, log domain.AuditLog) error {
	f.recorded = append(f.recorded, log)
	return nil
}

func (f *fakeAuditRepository) List(_ context.Context, _ int) ([]domain.AuditLog, error) {
	return f.recorded, nil
}

func TestAuditServiceRejectsIncompleteRecord(t *testing.T) {
	service := NewAuditService(&fakeAuditRepository{})

	err := service.Record(context.Background(), domain.AuditLog{ID: uuid.New()})

	require.ErrorIs(t, err, ErrValidation)
}

func TestAuditServiceRecordsValidEvent(t *testing.T) {
	repository := &fakeAuditRepository{}
	service := NewAuditService(repository)
	wanted := domain.AuditLog{ID: uuid.New(), EventID: uuid.New(), Action: "block", TargetType: "user"}

	err := service.Record(context.Background(), wanted)

	require.NoError(t, err)
	require.Equal(t, []domain.AuditLog{wanted}, repository.recorded)
}
