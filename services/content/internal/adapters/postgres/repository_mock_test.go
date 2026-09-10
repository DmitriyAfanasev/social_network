package postgres

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

func TestLikeRepositoryAddIsIdempotentAtSQLBoundary(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewLikeRepository(db)
	postID, userID := uuid.New(), uuid.New()
	db.ExpectExec("INSERT INTO content.post_likes").WithArgs(postID, userID).WillReturnResult(pgxmock.NewResult("INSERT", 0))

	// Act
	err = repository.Add(context.Background(), postID, userID)

	// Assert
	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestLikeRepositoryCountReturnsDatabaseValue(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewLikeRepository(db)
	postID := uuid.New()
	db.ExpectQuery("SELECT COUNT\\(\\*\\) FROM content.post_likes").WithArgs(postID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))

	// Act
	count, err := repository.Count(context.Background(), postID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestCommentRepositoryMapsNoRowsToNotFound(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewCommentRepository(db)
	commentID := uuid.New()
	db.ExpectQuery("SELECT id, post_id, author_id").WithArgs(commentID).WillReturnError(pgx.ErrNoRows)

	// Act
	_, err = repository.FindByID(context.Background(), commentID)

	// Assert
	require.ErrorIs(t, err, ports.ErrNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostRepositoryFindByIDLoadsMediaIDs(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewPostRepository(db)
	postID, authorID, mediaID := uuid.New(), uuid.New(), uuid.New()
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectQuery("SELECT id, author_id, body").WithArgs(postID).WillReturnRows(
		pgxmock.NewRows([]string{"id", "author_id", "body", "created_at", "updated_at", "count"}).AddRow(postID, authorID, "hello", createdAt, createdAt, 0),
	)
	db.ExpectQuery("SELECT media_id FROM content.post_media").WithArgs(postID).WillReturnRows(
		pgxmock.NewRows([]string{"media_id"}).AddRow(mediaID),
	)

	// Act
	post, err := repository.FindByID(context.Background(), postID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, postID, post.ID)
	require.Equal(t, authorID, post.AuthorID)
	require.Equal(t, []uuid.UUID{mediaID}, post.MediaIDs)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostRepositoryRollsBackWhenOutboxInsertFails(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewPostRepository(db)
	postID, authorID, eventID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectBegin()
	db.ExpectQuery("INSERT INTO content.posts").WithArgs(postID, authorID, "hello").WillReturnRows(
		pgxmock.NewRows([]string{"id", "author_id", "body", "created_at", "updated_at", "count"}).AddRow(postID, authorID, "hello", now, now, 0),
	)
	db.ExpectExec("DELETE FROM content.post_media").WithArgs(postID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	db.ExpectExec("INSERT INTO content.outbox_events").WithArgs(eventID, "content.post.created", &postID, []byte(`{}`), "corr-1", now, now).
		WillReturnError(errors.New("outbox unavailable"))
	db.ExpectRollback()

	// Act
	_, err = repository.CreateWithOutbox(context.Background(), domain.Post{ID: postID, AuthorID: authorID, Body: "hello"}, ports.OutboxEvent{ID: eventID, EventType: "content.post.created", AggregateID: &postID, Payload: []byte(`{}`), CorrelationID: "corr-1", AvailableAt: now, CreatedAt: now})

	// Assert
	require.EqualError(t, err, "outbox unavailable")
	require.NoError(t, db.ExpectationsWereMet())
}
