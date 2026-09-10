package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPostgresAnalyticsRepositories(t *testing.T) {
	// Arrange
	dsn := os.Getenv("ANALYTICS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("ANALYTICS_TEST_DATABASE_URL не задан")
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

	userID, videoID := uuid.New(), uuid.New()
	payload, err := json.Marshal(map[string]string{"user_id": userID.String()})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO analytics.event_records (event_id, event_type, aggregate_id, payload, created_at)
		VALUES ($1, 'identity.user.registered', $2, $3, now()),
		       ($4, 'content.post.created', $5, $3, now()),
		       ($6, 'content.post.like.created', $5, $3, now()),
		       ($7, 'profiles.photo.deleted', $2, $3, now())`,
		uuid.New(), userID, payload, uuid.New(), uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO analytics.video_events
			(event_id, event_type, video_id, user_id, session_id, watch_seconds, progress_seconds, duration_seconds, completed, occurred_at)
		VALUES ($1, 'media.video.viewed', $2, $3, 'session-a', 12.5, 12.5, 20, true, now()),
		       ($4, 'media.video.viewed', $2, $5, 'session-b', 7.5, 7.5, 20, false, now()),
		       ($6, 'media.video.playback', $2, $3, 'session-a', 3, 15.5, 20, false, now())`,
		uuid.New(), videoID, userID, uuid.New(), uuid.New(), uuid.New())
	require.NoError(t, err)

	summaryRepository := NewSummaryRepository(pool)
	videoRepository := NewVideoAnalyticsRepository(pool)

	// Act
	summary, err := summaryRepository.GetSummary(ctx)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 4, summary.EventsCount)
	require.EqualValues(t, 1, summary.UniqueUsers)
	require.EqualValues(t, 1, summary.UsersRegistered)
	require.EqualValues(t, 1, summary.PostsCreated)
	require.EqualValues(t, 1, summary.LikesAdded)
	require.EqualValues(t, 1, summary.PhotosDeleted)

	// Act
	stats, err := videoRepository.GetVideoStats(ctx, videoID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, videoID, stats.VideoID)
	require.EqualValues(t, 2, stats.Views)
	require.EqualValues(t, 2, stats.UniqueViewers)
	require.InDelta(t, 23, stats.TotalWatchSeconds, 0.001)
	require.EqualValues(t, 1, stats.CompletedViews)

	// Act
	history, err := videoRepository.ListViewerVideoStats(ctx, userID, 10)

	// Assert
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, videoID, history[0].VideoID)
	require.EqualValues(t, 1, history[0].Views)
	require.InDelta(t, 15.5, history[0].WatchSeconds, 0.001)
}
