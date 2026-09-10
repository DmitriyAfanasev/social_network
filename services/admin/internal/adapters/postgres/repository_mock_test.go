package postgres

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"general-project/admin/internal/domain"
)

func TestAuditRepositoryRecordsIdempotently(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewAuditRepository(db)
	logID, eventID := uuid.New(), uuid.New()
	var actorID, targetID *uuid.UUID
	db.ExpectExec("INSERT INTO admin.moderation_audit_logs").WithArgs(logID, eventID, actorID, "delete", "post", targetID, []byte(`{}`), "corr-1", time.Time{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 0))

	// Act
	err = repository.Record(context.Background(), domain.AuditLog{ID: logID, EventID: eventID, Action: "delete", TargetType: "post", Details: []byte(`{}`), CorrelationID: "corr-1"})

	// Assert
	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestAuditRepositoryListsNullableActors(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewAuditRepository(db)
	eventID, targetID := uuid.New(), uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectQuery("SELECT id, event_id, actor_id").WithArgs(10).WillReturnRows(
		pgxmock.NewRows([]string{"id", "event_id", "actor_id", "action", "target_type", "target_id", "details", "correlation_id", "created_at"}).
			AddRow(uuid.New(), eventID, nil, "delete", "post", &targetID, []byte(`{}`), "corr-1", now),
	)

	// Act
	logs, err := repository.List(context.Background(), 10)

	// Assert
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Nil(t, logs[0].ActorID)
	require.Equal(t, targetID, *logs[0].TargetID)
	require.NoError(t, db.ExpectationsWereMet())
}
