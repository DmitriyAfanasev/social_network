package postgres

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"general-project/social/internal/domain"
	"general-project/social/internal/ports"
)

func TestFriendshipRepositoryNormalizesPairForLookup(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewFriendshipRepository(db)
	first, second := uuid.New(), uuid.New()
	left, right := domain.Pair(first, second)
	db.ExpectQuery("SELECT EXISTS \\(SELECT 1 FROM social.friendships").WithArgs(left, right).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	// Act
	got, err := repository.IsFriend(context.Background(), second, first)

	// Assert
	require.NoError(t, err)
	require.True(t, got)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestBlockRepositoryUnblockReturnsRowsAffected(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewBlockRepository(db)
	blockerID, blockedID := uuid.New(), uuid.New()
	db.ExpectExec("DELETE FROM social.user_blocks").WithArgs(blockerID, blockedID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// Act
	removed, err := repository.Unblock(context.Background(), blockerID, blockedID)

	// Assert
	require.NoError(t, err)
	require.True(t, removed)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestFriendshipRepositoryRollsBackWhenOutboxInsertFails(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewFriendshipRepository(db)
	first, second, eventID := uuid.New(), uuid.New(), uuid.New()
	db.ExpectBegin()
	db.ExpectExec("INSERT INTO social.friendships").WithArgs(first, second, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("outbox unavailable"))
	db.ExpectRollback()

	// Act
	err = repository.CreateFriendshipWithOutbox(context.Background(), first, second, ports.OutboxEvent{ID: eventID, EventType: "social.friendship.created", Payload: []byte(`{}`)})

	// Assert
	require.EqualError(t, err, "outbox unavailable")
	require.NoError(t, db.ExpectationsWereMet())
}
