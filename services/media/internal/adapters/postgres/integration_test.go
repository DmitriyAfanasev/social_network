package postgres

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

func TestPostgresMediaRepositories(t *testing.T) {
	// Arrange
	dsn := os.Getenv("MEDIA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MEDIA_TEST_DATABASE_URL не задан")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	mediaRepository := NewMediaRepository(pool)
	musicRepository := NewMusicRepository(pool)
	mediaID, ownerID := uuid.New(), uuid.New()

	// Act
	media, err := mediaRepository.Create(ctx, domain.Media{ID: mediaID, ObjectKey: "integration/" + mediaID.String(), Bucket: "test", OriginalFilename: "track.mp3", ContentType: "audio/mpeg", MediaType: "audio", Size: 1024, UploadedBy: ownerID})

	// Assert
	require.NoError(t, err)
	require.Equal(t, mediaID, media.ID)
	loaded, err := mediaRepository.FindByID(ctx, mediaID)
	require.NoError(t, err)
	require.Equal(t, media.ObjectKey, loaded.ObjectKey)

	// Act
	track, err := musicRepository.Create(ctx, domain.MusicTrack{ID: uuid.New(), MediaID: mediaID, UserID: ownerID, Title: "Integration track", Artist: "Test artist"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, mediaID, track.MediaID)
	tracks, err := musicRepository.ListByOwner(ctx, ownerID, uuid.New())
	require.NoError(t, err)
	require.Len(t, tracks, 1)

	// Act
	err = mediaRepository.Delete(ctx, mediaID, time.Now().UTC())

	// Assert
	require.NoError(t, err)
	_, err = mediaRepository.FindByID(ctx, mediaID)
	require.ErrorIs(t, err, ports.ErrNotFound)
	_, err = musicRepository.FindByID(ctx, track.ID)
	require.ErrorIs(t, err, ports.ErrNotFound)
}
