package ports

import (
	"context"
	"uuid"

	"general-project/media/internal/domain"
)

// MusicRepository предоставляет media-сервису доступ к музыкальным трекам.
type MusicRepository interface {
	Create(ctx context.Context, track domain.MusicTrack) (domain.MusicTrack, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.MusicTrack, error)
	FindByID(ctx context.Context, trackID uuid.UUID) (domain.MusicTrack, error)
	Delete(ctx context.Context, trackID uuid.UUID) error
}

// MusicVisibilityReader проверяет право просматривать музыку владельца.
type MusicVisibilityReader interface {
	CanViewMusic(ctx context.Context, viewerID uuid.UUID, ownerID uuid.UUID) (bool, error)
}
