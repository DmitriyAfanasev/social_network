package domain

import (
	"time"
	"uuid"
)

// VideoStats содержит агрегаты просмотров и взаимодействий с видео.
type VideoStats struct {
	VideoID             uuid.UUID
	Views               int64
	UniqueViewers       int64
	TotalWatchSeconds   float64
	AverageWatchSeconds float64
	CompletedViews      int64
	LastViewedAt        *time.Time
}

// ViewerVideoStats содержит историю просмотра видео одним пользователем.
type ViewerVideoStats struct {
	VideoID             uuid.UUID
	Views               int64
	WatchSeconds        float64
	AverageWatchSeconds float64
	LastViewedAt        time.Time
}
