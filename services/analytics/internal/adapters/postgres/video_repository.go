package postgres

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/analytics/internal/domain"
)

// VideoAnalyticsRepository читает агрегаты видео из analytics-схемы.
type VideoAnalyticsRepository struct {
	pool *pgxpool.Pool
}

// NewVideoAnalyticsRepository создаёт репозиторий аналитики видео.
func NewVideoAnalyticsRepository(pool *pgxpool.Pool) *VideoAnalyticsRepository {
	return &VideoAnalyticsRepository{pool: pool}
}

// GetVideoStats возвращает просмотры, уникальных зрителей и удержание видео.
func (r *VideoAnalyticsRepository) GetVideoStats(ctx context.Context, videoID uuid.UUID) (domain.VideoStats, error) {
	const query = `
		WITH sessions AS (
			SELECT
				COALESCE(user_id::text, NULLIF(session_id, ''), event_id::text) AS viewer_key,
				COUNT(*) FILTER (WHERE event_type = 'media.video.viewed') AS views,
				COALESCE(SUM(watch_seconds), 0) AS watch_seconds,
				BOOL_OR(completed) AS completed,
				MAX(occurred_at) AS last_viewed_at
			FROM analytics.video_events
			WHERE video_id = $1 AND event_type IN ('media.video.viewed', 'media.video.playback')
			GROUP BY COALESCE(user_id::text, NULLIF(session_id, ''), event_id::text)
		)
		SELECT
			$1,
			COALESCE(SUM(views), 0)::bigint,
			COUNT(*) FILTER (WHERE views > 0)::bigint,
			COALESCE(SUM(watch_seconds), 0)::double precision,
			COALESCE(AVG(watch_seconds) FILTER (WHERE views > 0), 0)::double precision,
			COUNT(*) FILTER (WHERE views > 0 AND completed)::bigint,
			MAX(last_viewed_at) FILTER (WHERE views > 0)
		FROM sessions`
	var stats domain.VideoStats
	var lastViewedAt *time.Time
	if err := r.pool.QueryRow(ctx, query, videoID).Scan(
		&stats.VideoID, &stats.Views, &stats.UniqueViewers, &stats.TotalWatchSeconds,
		&stats.AverageWatchSeconds, &stats.CompletedViews, &lastViewedAt,
	); err != nil {
		return domain.VideoStats{}, err
	}
	stats.LastViewedAt = lastViewedAt
	return stats, nil
}

// ListViewerVideoStats возвращает историю видео, просмотренных пользователем.
func (r *VideoAnalyticsRepository) ListViewerVideoStats(ctx context.Context, userID uuid.UUID, limit int) ([]domain.ViewerVideoStats, error) {
	const query = `
		WITH sessions AS (
			SELECT
				video_id,
				COALESCE(NULLIF(session_id, ''), event_id::text) AS session_key,
				COUNT(*) FILTER (WHERE event_type = 'media.video.viewed') AS views,
				COALESCE(SUM(watch_seconds), 0) AS watch_seconds,
				MAX(occurred_at) AS last_viewed_at
			FROM analytics.video_events
			WHERE user_id = $1 AND event_type IN ('media.video.viewed', 'media.video.playback')
			GROUP BY video_id, COALESCE(NULLIF(session_id, ''), event_id::text)
		)
		SELECT video_id, COALESCE(SUM(views), 0)::bigint,
			COALESCE(SUM(watch_seconds), 0)::double precision,
			COALESCE(AVG(watch_seconds) FILTER (WHERE views > 0), 0)::double precision,
			MAX(last_viewed_at)
		FROM sessions
		GROUP BY video_id
		ORDER BY MAX(last_viewed_at) DESC, video_id
		LIMIT $2`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.ViewerVideoStats, 0, limit)
	for rows.Next() {
		var item domain.ViewerVideoStats
		if err := rows.Scan(&item.VideoID, &item.Views, &item.WatchSeconds, &item.AverageWatchSeconds, &item.LastViewedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

var _ interface {
	GetVideoStats(context.Context, uuid.UUID) (domain.VideoStats, error)
	ListViewerVideoStats(context.Context, uuid.UUID, int) ([]domain.ViewerVideoStats, error)
} = (*VideoAnalyticsRepository)(nil)
