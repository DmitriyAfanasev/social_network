package httptransport

import (
	"testing"
	"time"

	"uuid"

	"general-project/analytics/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestMapVideoStatsUsesGeneratedResponseModel(t *testing.T) {
	t.Parallel()

	lastViewedAt := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	videoID := uuid.MustParse("b7af1319-3020-4371-bb97-7a5fd3d3de43")

	response := mapVideoStats(domain.VideoStats{
		VideoID:             videoID,
		Views:               42,
		UniqueViewers:       10,
		TotalWatchSeconds:   120.5,
		AverageWatchSeconds: 12.05,
		CompletedViews:      7,
		LastViewedAt:        &lastViewedAt,
	})

	require.Equal(t, videoID.String(), response.VideoId)
	require.Equal(t, int64(42), response.Views)
	require.Equal(t, &lastViewedAt, response.LastViewedAt)
}
