package postgres

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"general-project/analytics/internal/domain"
)

func TestSummaryRepositoryMapsAggregates(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewSummaryRepository(db)
	db.ExpectQuery("SELECT").WillReturnRows(pgxmock.NewRows([]string{"events", "users", "registered", "posts", "comments", "added", "removed", "photos"}).AddRow(int64(8), int64(3), int64(2), int64(1), int64(1), int64(2), int64(1), int64(1)))

	// Act
	summary, err := repository.GetSummary(context.Background())

	// Assert
	require.NoError(t, err)
	require.Equal(t, domain.Summary{EventsCount: 8, UniqueUsers: 3, UsersRegistered: 2, PostsCreated: 1, CommentsCreated: 1, LikesAdded: 2, LikesRemoved: 1, PhotosDeleted: 1}, summary)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestVideoRepositoryMapsNullableLastViewedAt(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewVideoAnalyticsRepository(db)
	videoID := uuid.New()
	db.ExpectQuery("WITH sessions").WithArgs(videoID).WillReturnRows(pgxmock.NewRows([]string{"video_id", "views", "unique_viewers", "watch", "average", "completed", "last_viewed"}).AddRow(videoID, int64(4), int64(2), 91.5, 45.75, int64(1), nil))

	// Act
	stats, err := repository.GetVideoStats(context.Background(), videoID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, videoID, stats.VideoID)
	require.Equal(t, int64(4), stats.Views)
	require.Nil(t, stats.LastViewedAt)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestProcessedEventRepositoryClaimsEvent(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewProcessedEventRepository(db)
	eventID := uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectQuery("INSERT INTO analytics.processed_events").WithArgs(eventID, now).WillReturnRows(pgxmock.NewRows([]string{"event_id", "attempts"}).AddRow(eventID, 2))

	// Act
	result, err := repository.Claim(context.Background(), eventID, now)

	// Assert
	require.NoError(t, err)
	require.True(t, result.Claimed)
	require.Equal(t, 2, result.Attempts)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestEventRecordHandlerCommitsNewNonVideoEvent(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	handler := NewEventRecordHandler(db)
	event := domain.Event{ID: uuid.New(), EventType: "content.post.created", Payload: []byte(`{}`), CreatedAt: time.Now().UTC()}
	db.ExpectBegin()
	db.ExpectExec("INSERT INTO analytics.event_records").WithArgs(event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	// Act
	err = handler.Handle(context.Background(), event)

	// Assert
	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestEventRecordHandlerRollsBackWhenRecordInsertFails(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	handler := NewEventRecordHandler(db)
	event := domain.Event{ID: uuid.New(), EventType: "content.post.created", Payload: []byte(`{}`), CreatedAt: time.Now().UTC()}
	db.ExpectBegin()
	db.ExpectExec("INSERT INTO analytics.event_records").WithArgs(event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.CreatedAt).
		WillReturnError(errors.New("record unavailable"))
	db.ExpectRollback()

	// Act
	err = handler.Handle(context.Background(), event)

	// Assert
	require.EqualError(t, err, "record unavailable")
	require.NoError(t, db.ExpectationsWereMet())
}
