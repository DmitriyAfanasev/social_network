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

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

func TestMediaRepositoryFindByIDMapsMetadata(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewMediaRepository(db)
	mediaID, ownerID := uuid.New(), uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectQuery("SELECT id, object_key, bucket").WithArgs(mediaID).WillReturnRows(
		pgxmock.NewRows([]string{"id", "object_key", "bucket", "original_filename", "content_type", "media_type", "size", "checksum", "width", "height", "duration", "uploaded_by", "created_at", "deleted_at"}).
			AddRow(mediaID, "media/file.jpg", "general-project", "file.jpg", "image/jpeg", "image", int64(123), "checksum", nil, nil, nil, ownerID, now, nil),
	)

	// Act
	media, err := repository.FindByID(context.Background(), mediaID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, mediaID, media.ID)
	require.Equal(t, ownerID, media.UploadedBy)
	require.Equal(t, int64(123), media.Size)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestMediaRepositoryMapsMissingMediaToNotFound(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewMediaRepository(db)
	mediaID := uuid.New()
	db.ExpectQuery("SELECT id, object_key, bucket").WithArgs(mediaID).WillReturnError(pgx.ErrNoRows)

	// Act
	_, err = repository.FindByID(context.Background(), mediaID)

	// Assert
	require.ErrorIs(t, err, ports.ErrNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestMusicRepositoryDeleteMapsZeroRowsToNotFound(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewMusicRepository(db)
	trackID := uuid.New()
	db.ExpectExec("DELETE FROM media.music_tracks").WithArgs(trackID).WillReturnResult(pgxmock.NewResult("DELETE", 0))

	// Act
	err = repository.Delete(context.Background(), trackID)

	// Assert
	require.ErrorIs(t, err, ports.ErrNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestMediaRepositoryRollsBackWhenOutboxInsertFails(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewMediaRepository(db)
	mediaID, eventID, ownerID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectBegin()
	db.ExpectQuery("INSERT INTO media.media").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows([]string{"id", "object_key", "bucket", "original_filename", "content_type", "media_type", "size", "checksum", "width", "height", "duration", "uploaded_by", "created_at", "deleted_at"}).AddRow(mediaID, "file.jpg", "bucket", "file.jpg", "image/jpeg", "image", int64(10), "sum", nil, nil, nil, ownerID, now, nil),
	)
	db.ExpectExec("INSERT INTO media.outbox_events").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("outbox unavailable"))
	db.ExpectRollback()

	// Act
	_, err = repository.CreateWithOutbox(context.Background(), domain.Media{ID: mediaID, ObjectKey: "file.jpg", Bucket: "bucket", OriginalFilename: "file.jpg", ContentType: "image/jpeg", MediaType: "image", UploadedBy: ownerID}, ports.OutboxEvent{ID: eventID, EventType: "media.created", Payload: []byte(`{}`), CreatedAt: now})

	// Assert
	require.EqualError(t, err, "outbox unavailable")
	require.NoError(t, db.ExpectationsWereMet())
}
