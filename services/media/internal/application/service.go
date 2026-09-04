package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"general-project/libs/platform/correlation"
	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

const maxUploadSize int64 = 50 * 1024 * 1024

const mediaCacheTTL = time.Minute

var (
	// ErrValidation означает, что параметры загрузки некорректны.
	ErrValidation = errors.New("media validation failed")
	// ErrForbidden означает, что пользователь не может изменить медиаобъект.
	ErrForbidden = errors.New("media management forbidden")
)

// UploadInput содержит нормализованные данные загружаемого файла.
type UploadInput struct {
	Filename    string
	ContentType string
	Content     []byte
}

// MediaDTO представляет безопасный результат application-сценария медиа.
type MediaDTO struct {
	ID               uuid.UUID
	OriginalFilename string
	ContentType      string
	MediaType        string
	Size             int64
	Checksum         string
	UploadedBy       uuid.UUID
	CreatedAt        time.Time
}

// MediaService реализует сценарии загрузки и управления метаданными медиа.
type MediaService struct {
	media   ports.MediaRepository
	cache   ports.MediaCache
	storage ports.ObjectStorage
	bucket  string
	outbox  ports.OutboxRepository
}

// NewMediaService создаёт application-сервис медиа.
func NewMediaService(media ports.MediaRepository, cache ports.MediaCache, storage ports.ObjectStorage, bucket string, outboxes ...ports.OutboxRepository) *MediaService {
	var outbox ports.OutboxRepository
	if len(outboxes) > 0 {
		outbox = outboxes[0]
	}
	return &MediaService{media: media, cache: cache, storage: storage, bucket: bucket, outbox: outbox}
}

// Upload валидирует файл, сохраняет объект и создаёт его метаданные.
func (s *MediaService) Upload(ctx context.Context, userID uuid.UUID, input UploadInput) (MediaDTO, error) {
	media, err := s.uploadMedia(ctx, userID, input)
	if err != nil {
		return MediaDTO{}, err
	}
	return toMediaDTO(media), nil
}

func (s *MediaService) uploadMedia(ctx context.Context, userID uuid.UUID, input UploadInput) (domain.Media, error) {
	filename := filepath.Base(strings.TrimSpace(input.Filename))
	if filename == "." || filename == ".." || filename == "" || len([]rune(filename)) > 255 || len(input.Content) < 1 || int64(len(input.Content)) > maxUploadSize {
		return domain.Media{}, ErrValidation
	}
	mediaType := mediaTypeFromContentType(input.ContentType)
	mediaID := uuid.New()
	objectKey := fmt.Sprintf("objects/%s/%s-%s", userID, mediaID, filename)
	if err := s.storage.Put(ctx, objectKey, input.ContentType, bytes.NewReader(input.Content), int64(len(input.Content))); err != nil {
		return domain.Media{}, err
	}

	digest := sha256.Sum256(input.Content)
	metadata := domain.Media{
		ID:               mediaID,
		ObjectKey:        objectKey,
		Bucket:           s.bucket,
		OriginalFilename: filename,
		ContentType:      input.ContentType,
		MediaType:        mediaType,
		Size:             int64(len(input.Content)),
		Checksum:         hex.EncodeToString(digest[:]),
		UploadedBy:       userID,
	}
	var media domain.Media
	var err error
	if s.outbox == nil {
		media, err = s.media.Create(ctx, metadata)
	} else {
		payload, marshalErr := json.Marshal(map[string]any{
			"media_id":    metadata.ID,
			"uploaded_by": metadata.UploadedBy,
			"media_type":  metadata.MediaType,
			"size":        metadata.Size,
		})
		if marshalErr != nil {
			_ = s.storage.Delete(ctx, objectKey)
			return domain.Media{}, marshalErr
		}
		now := time.Now().UTC()
		media, err = s.media.CreateWithOutbox(ctx, metadata, ports.OutboxEvent{ID: uuid.New(), EventType: "media.media.created", AggregateID: &metadata.ID, Payload: payload, CorrelationID: correlation.ID(ctx), AvailableAt: now, CreatedAt: now})
	}
	if err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return domain.Media{}, err
	}
	return media, nil
}

// GetByID возвращает активные метаданные медиаобъекта.
func (s *MediaService) GetByID(ctx context.Context, mediaID uuid.UUID) (MediaDTO, error) {
	cacheKey := mediaCacheKey(mediaID)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached MediaDTO
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	media, err := s.media.FindByID(ctx, mediaID)
	if err != nil {
		return MediaDTO{}, err
	}
	dto := toMediaDTO(media)
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(dto); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, mediaCacheTTL)
		}
	}
	return dto, nil
}

// Delete удаляет объект из storage и помечает метаданные удалёнными.
func (s *MediaService) Delete(ctx context.Context, userID uuid.UUID, mediaID uuid.UUID) error {
	media, err := s.media.FindByID(ctx, mediaID)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !media.CanBeManagedBy(userID) {
		return ErrForbidden
	}
	if err := s.storage.Delete(ctx, media.ObjectKey); err != nil {
		return err
	}
	if err := s.media.Delete(ctx, mediaID, time.Now().UTC()); err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.Delete(ctx, mediaCacheKey(mediaID))
	}
	return nil
}

func mediaTypeFromContentType(contentType string) string {
	prefix := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, "/", 2)[0]))
	if prefix == "image" || prefix == "video" || prefix == "audio" {
		return prefix
	}
	return "document"
}

func toMediaDTO(media domain.Media) MediaDTO {
	return MediaDTO{ID: media.ID, OriginalFilename: media.OriginalFilename, ContentType: media.ContentType, MediaType: media.MediaType, Size: media.Size, Checksum: media.Checksum, UploadedBy: media.UploadedBy, CreatedAt: media.CreatedAt}
}

func mediaCacheKey(mediaID uuid.UUID) string {
	return fmt.Sprintf("media:v1:metadata:%s", mediaID)
}
