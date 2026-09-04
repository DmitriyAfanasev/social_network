package domain

import (
	"time"
	"uuid"
)

// VideoAsset описывает видео как доменный объект поверх исходного media.
type VideoAsset struct {
	ID        uuid.UUID
	MediaID   uuid.UUID
	AlbumID   *uuid.UUID
	Title     string
	Status    string
	Duration  *float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VideoAlbum описывает пользовательский альбом видео.
type VideoAlbum struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VideoRendition описывает готовое видео определённой высоты.
type VideoRendition struct {
	ID          uuid.UUID
	VideoID     uuid.UUID
	Height      int
	ObjectKey   string
	ContentType string
	Size        int64
	Duration    float64
	CreatedAt   time.Time
}

const (
	// VideoStatusUploaded означает, что исходный файл загружен.
	VideoStatusUploaded = "uploaded"
	// VideoStatusProcessing означает, что worker обрабатывает исходный файл.
	VideoStatusProcessing = "processing"
	// VideoStatusReady означает, что renditions готовы.
	VideoStatusReady = "ready"
	// VideoStatusFailed означает, что обработка завершилась ошибкой.
	VideoStatusFailed = "failed"
)
