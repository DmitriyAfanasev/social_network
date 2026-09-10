package postgres

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"general-project/messaging/internal/domain"
	"general-project/messaging/internal/ports"
)

func TestRepositoryCreatesDirectConversationInTransaction(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewRepository(db)
	first, second := uuid.New(), uuid.New()
	conversationID := uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectBegin()
	db.ExpectQuery("INSERT INTO messaging.conversations").WithArgs(pgxmock.AnyArg(), domain.PairKey(first, second)).WillReturnRows(
		pgxmock.NewRows([]string{"id", "direct_key", "created_at", "updated_at"}).AddRow(conversationID, domain.PairKey(first, second), now, now),
	)
	db.ExpectExec("INSERT INTO messaging.conversation_participants").WithArgs(conversationID, first, second).WillReturnResult(pgxmock.NewResult("INSERT", 2))
	db.ExpectCommit()

	// Act
	conversation, err := repository.GetOrCreateDirect(context.Background(), first, second)

	// Assert
	require.NoError(t, err)
	require.Equal(t, conversationID, conversation.ID)
	require.Equal(t, []uuid.UUID{first, second}, conversation.ParticipantIDs)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestRepositoryDeleteMessageMapsZeroRowsToNotFound(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewRepository(db)
	messageID, senderID := uuid.New(), uuid.New()
	db.ExpectExec("UPDATE messaging.messages SET deleted_at").WithArgs(messageID, senderID).WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	// Act
	err = repository.DeleteMessage(context.Background(), senderID, messageID)

	// Assert
	require.ErrorIs(t, err, ports.ErrNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestRepositoryRollsBackWhenParticipantInsertFails(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewRepository(db)
	first, second, conversationID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectBegin()
	db.ExpectQuery("INSERT INTO messaging.conversations").WithArgs(pgxmock.AnyArg(), domain.PairKey(first, second)).WillReturnRows(
		pgxmock.NewRows([]string{"id", "direct_key", "created_at", "updated_at"}).AddRow(conversationID, domain.PairKey(first, second), now, now),
	)
	db.ExpectExec("INSERT INTO messaging.conversation_participants").WithArgs(conversationID, first, second).WillReturnError(errors.New("participants unavailable"))
	db.ExpectRollback()

	// Act
	_, err = repository.GetOrCreateDirect(context.Background(), first, second)

	// Assert
	require.EqualError(t, err, "participants unavailable")
	require.NoError(t, db.ExpectationsWereMet())
}
