package ports

import (
	"context"
	"errors"
	"time"
	"uuid"

	"general-project/media/internal/domain"
)

var (
	// ErrForbidden означает, что пользователь не является владельцем ресурса.
	ErrForbidden = errors.New("video management forbidden")
)

// VideoRepository предоставляет media-сервису доступ к video-метаданным.
type VideoRepository interface {
	Create(ctx context.Context, video domain.VideoAsset) (domain.VideoAsset, error)
	FindByID(ctx context.Context, videoID uuid.UUID) (domain.VideoAsset, error)
	FindDetails(ctx context.Context, videoID uuid.UUID, viewerID *uuid.UUID) (VideoDetails, error)
	ListRenditions(ctx context.Context, videoID uuid.UUID) ([]domain.VideoRendition, error)
	ListAlbums(ctx context.Context, ownerID uuid.UUID, viewerID uuid.UUID, tab string, search string) ([]VideoAlbumView, error)
	CreateAlbum(ctx context.Context, album domain.VideoAlbum) (domain.VideoAlbum, error)
	AlbumBelongsToUser(ctx context.Context, albumID uuid.UUID, userID uuid.UUID) (bool, error)
	DeleteAlbum(ctx context.Context, albumID uuid.UUID, userID uuid.UUID) ([]VideoDeletion, error)
	Delete(ctx context.Context, videoID uuid.UUID) error
	MarkProcessing(ctx context.Context, videoID uuid.UUID) error
	Complete(ctx context.Context, videoID uuid.UUID, status string, duration float64, renditions []domain.VideoRendition, updatedAt time.Time) error
	RecordView(ctx context.Context, videoID uuid.UUID, userID *uuid.UUID) (int, error)
	RecordViewWithOutbox(ctx context.Context, videoID uuid.UUID, userID *uuid.UUID, event OutboxEvent) (int, error)
	ToggleLike(ctx context.Context, videoID uuid.UUID, userID uuid.UUID) (InteractionResult, error)
	SetBookmark(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error)
	SetFavorite(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error)
}

// VideoDetails содержит read model видео и пользовательские признаки взаимодействия.
type VideoDetails struct {
	VideoListItem
	Renditions []domain.VideoRendition
}

// VideoListItem содержит отображаемые поля видео для списков и карточек.
type VideoListItem struct {
	ID                 uuid.UUID
	MediaID            uuid.UUID
	AlbumID            *uuid.UUID
	OwnerID            uuid.UUID
	Title              string
	OriginalFilename   string
	Status             string
	Duration           *float64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ViewsCount         int
	LikesCount         int
	LikedByViewer      bool
	BookmarkedByViewer bool
	FavoritedByViewer  bool
}

// VideoAlbumView содержит альбом и видео, сгруппированные для выдачи списком.
type VideoAlbumView struct {
	ID     *uuid.UUID
	Title  string
	Videos []VideoListItem
}

// VideoDeletion содержит идентификаторы видео и media для удаления альбома.
type VideoDeletion struct {
	VideoID uuid.UUID
	MediaID uuid.UUID
}

// InteractionResult содержит новый статус like и актуальное количество likes.
type InteractionResult struct {
	Liked      bool
	LikesCount int
}

// TranscodePublisher публикует команду для асинхронной обработки видео.
type TranscodePublisher interface {
	Publish(ctx context.Context, request TranscodeRequest) error
}

// TranscodeRequest содержит данные, необходимые video-worker для обработки.
type TranscodeRequest struct {
	VideoID          uuid.UUID `json:"video_id"`
	MediaID          uuid.UUID `json:"media_id"`
	Bucket           string    `json:"bucket"`
	ObjectKey        string    `json:"object_key"`
	RequestedHeights []int     `json:"requested_heights"`
}

// TranscodeCompletion содержит результат работы video-worker.
type TranscodeCompletion struct {
	VideoID    uuid.UUID            `json:"video_id"`
	Status     string               `json:"status"`
	Duration   float64              `json:"duration"`
	Renditions []TranscodeRendition `json:"renditions"`
}

// TranscodeRendition содержит результат обработки одного разрешения.
type TranscodeRendition struct {
	Height      int     `json:"height"`
	ObjectKey   string  `json:"object_key"`
	ContentType string  `json:"content_type"`
	Size        int64   `json:"size"`
	Duration    float64 `json:"duration"`
}
