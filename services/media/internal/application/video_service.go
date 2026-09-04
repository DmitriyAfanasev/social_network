package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

const videoCacheTTL = 30 * time.Second

// CreateVideoInput содержит параметры создания видео.
type CreateVideoInput struct {
	Title            string
	AlbumID          *uuid.UUID
	File             UploadInput
	RequestedHeights []int
}

// VideoDTO представляет безопасный результат application-сценария видео.
type VideoDTO struct {
	ID                 uuid.UUID
	MediaID            uuid.UUID
	AlbumID            *uuid.UUID
	OwnerID            uuid.UUID
	Title              string
	OriginalFilename   string
	Status             string
	Duration           *float64
	ViewsCount         int
	LikesCount         int
	LikedByViewer      bool
	BookmarkedByViewer bool
	FavoritedByViewer  bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Renditions         []RenditionDTO
}

// RenditionDTO представляет готовое видео конкретного разрешения.
type RenditionDTO struct {
	ID          uuid.UUID
	Height      int
	ContentType string
	Size        int64
	Duration    float64
	CreatedAt   time.Time
}

// VideoService реализует создание видео и постановку transcode-задачи.
type VideoService struct {
	media     *MediaService
	videos    ports.VideoRepository
	publisher ports.TranscodePublisher
	cache     ports.MediaCache
}

// NewVideoService создаёт application-сервис видео.
func NewVideoService(media *MediaService, videos ports.VideoRepository, publisher ports.TranscodePublisher, cache ports.MediaCache) *VideoService {
	return &VideoService{media: media, videos: videos, publisher: publisher, cache: cache}
}

// Create загружает исходное видео, создаёт video-метаданные и ставит задачу worker.
func (s *VideoService) Create(ctx context.Context, userID uuid.UUID, input CreateVideoInput) (VideoDTO, error) {
	if mediaTypeFromContentType(input.File.ContentType) != "video" || !validHeights(input.RequestedHeights) {
		return VideoDTO{}, ErrValidation
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		input.Title = "Видео"
	}
	if len([]rune(input.Title)) > 200 {
		return VideoDTO{}, ErrValidation
	}
	if input.AlbumID != nil {
		belongs, err := s.videos.AlbumBelongsToUser(ctx, *input.AlbumID, userID)
		if err != nil {
			return VideoDTO{}, err
		}
		if !belongs {
			return VideoDTO{}, ports.ErrNotFound
		}
	}
	media, err := s.media.uploadMedia(ctx, userID, input.File)
	if err != nil {
		return VideoDTO{}, err
	}
	video, err := s.videos.Create(ctx, domain.VideoAsset{ID: uuid.New(), MediaID: media.ID, AlbumID: input.AlbumID, Title: input.Title, Status: domain.VideoStatusProcessing})
	if err != nil {
		_ = s.media.Delete(ctx, userID, media.ID)
		return VideoDTO{}, err
	}
	if err := s.publisher.Publish(ctx, ports.TranscodeRequest{VideoID: video.ID, MediaID: media.ID, Bucket: media.Bucket, ObjectKey: media.ObjectKey, RequestedHeights: input.RequestedHeights}); err != nil {
		_ = s.videos.Delete(ctx, video.ID)
		_ = s.media.Delete(ctx, userID, media.ID)
		return VideoDTO{}, err
	}
	return toVideoDTO(ports.VideoListItem{ID: video.ID, MediaID: video.MediaID, AlbumID: video.AlbumID, Title: video.Title, Status: video.Status, CreatedAt: video.CreatedAt}, nil), nil
}

// GetByID возвращает video-метаданные и список готовых renditions.
func (s *VideoService) GetByID(ctx context.Context, videoID uuid.UUID, viewerIDs ...*uuid.UUID) (VideoDTO, error) {
	var viewerID *uuid.UUID
	if len(viewerIDs) > 0 {
		viewerID = viewerIDs[0]
	}
	cacheKey := videoCacheKey(videoID)
	if viewerID == nil && s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached VideoDTO
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	details, err := s.videos.FindDetails(ctx, videoID, viewerID)
	if err != nil {
		return VideoDTO{}, err
	}
	dto := toVideoDTO(details.VideoListItem, details.Renditions)
	if viewerID == nil && s.cache != nil {
		if payload, marshalErr := json.Marshal(dto); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, videoCacheTTL)
		}
	}
	return dto, nil
}

// ListAlbums возвращает видео текущего владельца или выбранную вкладку interactions.
func (s *VideoService) ListAlbums(ctx context.Context, ownerID uuid.UUID, viewerID uuid.UUID, tab string, search string) ([]VideoAlbumDTO, error) {
	if strings.TrimSpace(tab) == "" {
		tab = "uploaded"
	}
	if tab != "uploaded" && tab != "favorite" && tab != "viewed" && tab != "bookmarked" {
		return nil, ErrValidation
	}
	items, err := s.videos.ListAlbums(ctx, ownerID, viewerID, tab, strings.TrimSpace(search))
	if err != nil {
		return nil, err
	}
	result := make([]VideoAlbumDTO, 0, len(items))
	for _, album := range items {
		videos := make([]VideoDTO, 0, len(album.Videos))
		for _, video := range album.Videos {
			videos = append(videos, toVideoDTO(video, nil))
		}
		result = append(result, VideoAlbumDTO{ID: album.ID, Title: album.Title, Videos: videos})
	}
	return result, nil
}

// CreateAlbum создаёт альбом видео владельца.
func (s *VideoService) CreateAlbum(ctx context.Context, userID uuid.UUID, title string) (domain.VideoAlbum, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 80 {
		return domain.VideoAlbum{}, ErrValidation
	}
	return s.videos.CreateAlbum(ctx, domain.VideoAlbum{ID: uuid.New(), UserID: userID, Title: title})
}

// DeleteAlbum удаляет альбом, его видео и соответствующие объекты media.
func (s *VideoService) DeleteAlbum(ctx context.Context, userID uuid.UUID, albumID uuid.UUID) error {
	deletions, err := s.videos.DeleteAlbum(ctx, albumID, userID)
	if err != nil {
		return err
	}
	for _, deletion := range deletions {
		if err := s.media.Delete(ctx, userID, deletion.MediaID); err != nil && !errors.Is(err, ports.ErrNotFound) {
			return err
		}
		s.invalidateCache(ctx, deletion.VideoID)
	}
	return nil
}

// RecordView фиксирует просмотр видео и возвращает новый счётчик.
func (s *VideoService) RecordView(ctx context.Context, videoID uuid.UUID, userID *uuid.UUID) (int, error) {
	count, err := s.videos.RecordView(ctx, videoID, userID)
	if err == nil {
		s.invalidateCache(ctx, videoID)
	}
	return count, err
}

// ToggleLike переключает like видео.
func (s *VideoService) ToggleLike(ctx context.Context, videoID uuid.UUID, userID uuid.UUID) (ports.InteractionResult, error) {
	result, err := s.videos.ToggleLike(ctx, videoID, userID)
	if err == nil {
		s.invalidateCache(ctx, videoID)
	}
	return result, err
}

// SetBookmark изменяет состояние закладки видео.
func (s *VideoService) SetBookmark(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error) {
	result, err := s.videos.SetBookmark(ctx, videoID, userID, value)
	if err == nil {
		s.invalidateCache(ctx, videoID)
	}
	return result, err
}

// SetFavorite изменяет состояние избранного видео.
func (s *VideoService) SetFavorite(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error) {
	result, err := s.videos.SetFavorite(ctx, videoID, userID, value)
	if err == nil {
		s.invalidateCache(ctx, videoID)
	}
	return result, err
}

// VideoAlbumDTO представляет альбом видео на application-транспортной границе.
type VideoAlbumDTO struct {
	ID     *uuid.UUID
	Title  string
	Videos []VideoDTO
}

// Complete принимает результат video-worker и обновляет статус с renditions.
func (s *VideoService) Complete(ctx context.Context, completion ports.TranscodeCompletion) error {
	if completion.Status != domain.VideoStatusReady && completion.Status != domain.VideoStatusFailed {
		return ErrValidation
	}
	renditions := make([]domain.VideoRendition, 0, len(completion.Renditions))
	for _, rendition := range completion.Renditions {
		if rendition.Height < 1 || rendition.ObjectKey == "" || rendition.Size < 1 || rendition.Duration < 0 {
			return ErrValidation
		}
		renditions = append(renditions, domain.VideoRendition{ID: uuid.New(), VideoID: completion.VideoID, Height: rendition.Height, ObjectKey: rendition.ObjectKey, ContentType: rendition.ContentType, Size: rendition.Size, Duration: rendition.Duration})
	}
	err := s.videos.Complete(ctx, completion.VideoID, completion.Status, completion.Duration, renditions, time.Now().UTC())
	if err == nil {
		s.invalidateCache(ctx, completion.VideoID)
	}
	return err
}

// Delete удаляет видео только его владельцем исходного media-объекта.
func (s *VideoService) Delete(ctx context.Context, userID uuid.UUID, videoID uuid.UUID) error {
	video, err := s.videos.FindByID(ctx, videoID)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.media.Delete(ctx, userID, video.MediaID); err != nil {
		return err
	}
	if err := s.videos.Delete(ctx, videoID); err != nil {
		return err
	}
	s.invalidateCache(ctx, videoID)
	return nil
}

func validHeights(heights []int) bool {
	if len(heights) == 0 || len(heights) > 5 {
		return false
	}
	seen := make(map[int]struct{}, len(heights))
	for _, height := range heights {
		if height < 144 || height > 2160 {
			return false
		}
		if _, ok := seen[height]; ok {
			return false
		}
		seen[height] = struct{}{}
	}
	return true
}

func toVideoDTO(video ports.VideoListItem, renditions []domain.VideoRendition) VideoDTO {
	dto := VideoDTO{ID: video.ID, MediaID: video.MediaID, AlbumID: video.AlbumID, OwnerID: video.OwnerID, Title: video.Title, OriginalFilename: video.OriginalFilename, Status: video.Status, Duration: video.Duration, ViewsCount: video.ViewsCount, LikesCount: video.LikesCount, LikedByViewer: video.LikedByViewer, BookmarkedByViewer: video.BookmarkedByViewer, FavoritedByViewer: video.FavoritedByViewer, CreatedAt: video.CreatedAt, UpdatedAt: video.UpdatedAt, Renditions: make([]RenditionDTO, 0, len(renditions))}
	for _, rendition := range renditions {
		dto.Renditions = append(dto.Renditions, RenditionDTO{ID: rendition.ID, Height: rendition.Height, ContentType: rendition.ContentType, Size: rendition.Size, Duration: rendition.Duration, CreatedAt: rendition.CreatedAt})
	}
	return dto
}

func videoCacheKey(videoID uuid.UUID) string {
	return "media:v1:video:" + videoID.String()
}

func (s *VideoService) invalidateCache(ctx context.Context, videoID uuid.UUID) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, videoCacheKey(videoID))
	}
}
