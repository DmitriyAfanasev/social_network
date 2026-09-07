package application

import (
	"context"
	"errors"
	"uuid"

	"general-project/analytics/internal/domain"
	"general-project/analytics/internal/ports"
)

var (
	// ErrVideoAnalyticsValidation означает, что параметры запроса аналитики некорректны.
	ErrVideoAnalyticsValidation = errors.New("video analytics validation failed")
)

// VideoAnalyticsService возвращает агрегаты для будущих рекомендаций.
type VideoAnalyticsService struct {
	repository ports.VideoAnalyticsRepository
}

// NewVideoAnalyticsService создаёт сервис аналитики видео.
func NewVideoAnalyticsService(repository ports.VideoAnalyticsRepository) *VideoAnalyticsService {
	return &VideoAnalyticsService{repository: repository}
}

// GetVideoStats возвращает просмотры и удержание конкретного видео.
func (s *VideoAnalyticsService) GetVideoStats(ctx context.Context, videoID uuid.UUID) (domain.VideoStats, error) {
	if videoID == uuid.Nil() {
		return domain.VideoStats{}, ErrVideoAnalyticsValidation
	}
	return s.repository.GetVideoStats(ctx, videoID)
}

// ListViewerVideoStats возвращает историю просмотренных видео пользователя.
func (s *VideoAnalyticsService) ListViewerVideoStats(ctx context.Context, userID uuid.UUID, limit int) ([]domain.ViewerVideoStats, error) {
	if userID == uuid.Nil() || limit < 1 || limit > 100 {
		return nil, ErrVideoAnalyticsValidation
	}
	return s.repository.ListViewerVideoStats(ctx, userID, limit)
}
