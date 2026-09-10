package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/analytics/internal/domain"
)

type fakeVideoAnalyticsRepository struct {
	videoID uuid.UUID
	userID  uuid.UUID
	limit   int
}

func (f *fakeVideoAnalyticsRepository) GetVideoStats(_ context.Context, videoID uuid.UUID) (domain.VideoStats, error) {
	f.videoID = videoID
	return domain.VideoStats{VideoID: videoID, Views: 7, UniqueViewers: 3}, nil
}

func (f *fakeVideoAnalyticsRepository) ListViewerVideoStats(_ context.Context, userID uuid.UUID, limit int) ([]domain.ViewerVideoStats, error) {
	f.userID = userID
	f.limit = limit
	return []domain.ViewerVideoStats{{VideoID: uuid.New(), Views: 2}}, nil
}

func TestVideoAnalyticsServiceReturnsStats(t *testing.T) {
	t.Parallel()

	repository := &fakeVideoAnalyticsRepository{}
	videoID := uuid.New()
	service := NewVideoAnalyticsService(repository)

	stats, err := service.GetVideoStats(context.Background(), videoID)

	require.NoError(t, err)
	require.Equal(t, videoID, repository.videoID)
	require.Equal(t, int64(7), stats.Views)
}

func TestVideoAnalyticsServiceValidatesViewerQuery(t *testing.T) {
	t.Parallel()

	repository := &fakeVideoAnalyticsRepository{}
	service := NewVideoAnalyticsService(repository)

	items, err := service.ListViewerVideoStats(context.Background(), uuid.New(), 101)

	require.Nil(t, items)
	require.ErrorIs(t, err, ErrVideoAnalyticsValidation)
	require.Zero(t, repository.limit)
}

func TestVideoAnalyticsServiceValidatesVideoAndViewerBoundaries(t *testing.T) {
	t.Parallel()

	repository := &fakeVideoAnalyticsRepository{}
	service := NewVideoAnalyticsService(repository)

	_, err := service.GetVideoStats(context.Background(), uuid.Nil())
	require.ErrorIs(t, err, ErrVideoAnalyticsValidation)

	for _, limit := range []int{0, 101} {
		_, err = service.ListViewerVideoStats(context.Background(), uuid.New(), limit)
		require.ErrorIs(t, err, ErrVideoAnalyticsValidation)
	}
	_, err = service.ListViewerVideoStats(context.Background(), uuid.Nil(), 10)
	require.ErrorIs(t, err, ErrVideoAnalyticsValidation)
	require.Zero(t, repository.limit)
}
