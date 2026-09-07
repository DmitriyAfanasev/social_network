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
	IsSaved   bool
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
	if ownerID == viewerID {
		return s.list(ctx, ownerID)
	}
	tracks, err := s.music.ListByOwner(ctx, ownerID, viewerID)
	if err != nil {
		return nil, err
	}
	return mapMusicDTOs(tracks), nil
}

// AddToLibrary добавляет доступный пользователю трек в его личную аудиотеку.
func (s *MusicService) AddToLibrary(ctx context.Context, userID uuid.UUID, trackID uuid.UUID) error {
	track, err := s.music.FindByID(ctx, trackID)
	if err != nil {
		return err
	}
	if track.UserID == userID {
		return nil
	}
	if s.visibility != nil {
		allowed, visibilityErr := s.visibility.CanViewMusic(ctx, userID, track.UserID)
		if visibilityErr != nil {
			return visibilityErr
		}
		if !allowed {
			return ErrForbidden
		}
	}
	if err := s.music.AddToLibrary(ctx, userID, trackID); err != nil {
		return err
	}
	s.invalidateUserCache(ctx, userID)
	return nil
}

// RemoveFromLibrary удаляет сохранённый чужой трек из личной аудиотеки.
func (s *MusicService) RemoveFromLibrary(ctx context.Context, userID uuid.UUID, trackID uuid.UUID) error {
	if err := s.music.RemoveFromLibrary(ctx, userID, trackID); err != nil {
		return err
	}
	s.invalidateUserCache(ctx, userID)
	return nil
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
	result = append(result, mapMusicDTOs(tracks)...)
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
	return MusicDTO{ID: track.ID, MediaID: track.MediaID, UserID: track.UserID, Title: track.Title, Artist: track.Artist, Duration: track.Duration, CreatedAt: track.CreatedAt, IsSaved: track.Saved}
}

func mapMusicDTOs(tracks []domain.MusicTrack) []MusicDTO {
	result := make([]MusicDTO, 0, len(tracks))
	for _, track := range tracks {
		result = append(result, toMusicDTO(track))
	}
	return result
}

func musicCacheKey(userID uuid.UUID) string {
	return "media:v1:music:" + userID.String()
}

func (s *MusicService) invalidateUserCache(ctx context.Context, userID uuid.UUID) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, musicCacheKey(userID))
	}
}
