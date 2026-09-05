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

const musicCacheTTL = time.Minute

// CreateMusicInput содержит параметры создания музыкального трека.
type CreateMusicInput struct {
	Title  string
	Artist string
	File   UploadInput
}

// MusicDTO представляет безопасный результат application-сценария трека.
type MusicDTO struct {
	ID        uuid.UUID
	MediaID   uuid.UUID
	UserID    uuid.UUID
	Title     string
	Artist    string
	Duration  *float64
	CreatedAt time.Time
}

// MusicService реализует сценарии музыкальных треков.
type MusicService struct {
	media      *MediaService
	music      ports.MusicRepository
	cache      ports.MediaCache
	visibility ports.MusicVisibilityReader
}

// NewMusicService создаёт application-сервис музыкальных треков.
func NewMusicService(media *MediaService, music ports.MusicRepository, cache ports.MediaCache, visibility ...ports.MusicVisibilityReader) *MusicService {
	var reader ports.MusicVisibilityReader
	if len(visibility) > 0 {
		reader = visibility[0]
	}
	return &MusicService{media: media, music: music, cache: cache, visibility: reader}
}

// Create загружает audio-файл и создаёт метаданные музыкального трека.
func (s *MusicService) Create(ctx context.Context, userID uuid.UUID, input CreateMusicInput) (MusicDTO, error) {
	if mediaTypeFromContentType(input.File.ContentType) != "audio" {
		return MusicDTO{}, ErrValidation
	}
	media, err := s.media.uploadMedia(ctx, userID, input.File)
	if err != nil {
		return MusicDTO{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = media.OriginalFilename
	}
	if len([]rune(title)) > 200 {
		title = string([]rune(title)[:200])
	}
	artist := strings.TrimSpace(input.Artist)
	if len([]rune(artist)) > 120 {
		artist = string([]rune(artist)[:120])
	}
	track, err := s.music.Create(ctx, domain.MusicTrack{ID: uuid.New(), MediaID: media.ID, UserID: userID, Title: title, Artist: artist, Duration: media.Duration})
	if err != nil {
		_ = s.media.Delete(ctx, userID, media.ID)
		return MusicDTO{}, err
	}
	s.invalidateUserCache(ctx, userID)
	return toMusicDTO(track), nil
}

// ListMine возвращает музыкальные треки текущего пользователя.
func (s *MusicService) ListMine(ctx context.Context, userID uuid.UUID) ([]MusicDTO, error) {
	return s.list(ctx, userID)
}

// ListForViewer возвращает музыку владельца с учётом политики видимости.
func (s *MusicService) ListForViewer(ctx context.Context, ownerID uuid.UUID, viewerID uuid.UUID) ([]MusicDTO, error) {
	if s.visibility != nil {
		allowed, err := s.visibility.CanViewMusic(ctx, viewerID, ownerID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrForbidden
		}
	}
	return s.list(ctx, ownerID)
}

func (s *MusicService) list(ctx context.Context, userID uuid.UUID) ([]MusicDTO, error) {
	cacheKey := musicCacheKey(userID)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached []MusicDTO
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	tracks, err := s.music.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]MusicDTO, 0, len(tracks))
	for _, track := range tracks {
		result = append(result, toMusicDTO(track))
	}
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(result); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, musicCacheTTL)
		}
	}
	return result, nil
}

// Delete удаляет трек и принадлежащий ему media-объект.
func (s *MusicService) Delete(ctx context.Context, userID uuid.UUID, trackID uuid.UUID) error {
	track, err := s.music.FindByID(ctx, trackID)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !track.CanBeManagedBy(userID) {
		return ErrForbidden
	}
	if err := s.media.Delete(ctx, userID, track.MediaID); err != nil {
		return err
	}
	if err := s.music.Delete(ctx, trackID); err != nil {
		return err
	}
	s.invalidateUserCache(ctx, userID)
	return nil
}

func toMusicDTO(track domain.MusicTrack) MusicDTO {
	return MusicDTO{ID: track.ID, MediaID: track.MediaID, UserID: track.UserID, Title: track.Title, Artist: track.Artist, Duration: track.Duration, CreatedAt: track.CreatedAt}
}

func musicCacheKey(userID uuid.UUID) string {
	return "media:v1:music:" + userID.String()
}

func (s *MusicService) invalidateUserCache(ctx context.Context, userID uuid.UUID) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, musicCacheKey(userID))
	}
}
