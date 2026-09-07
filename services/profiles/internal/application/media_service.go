package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"uuid"

	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
)

const defaultAvatarURL = "/media/default-avatar"

// ProfileMediaService реализует сценарии фотоальбомов и аватаров профиля.
type ProfileMediaService struct {
	mediaReader ports.PhotoMediaReader
	profiles    ports.ProfileRepository
	photos      ports.PhotoRepository
	avatars     ports.AvatarRepository
	cache       ports.ProfileCache
}

// NewGalleryService создаёт сервис галереи с проверкой владельца загружаемых файлов.
func NewGalleryService(profiles ports.ProfileRepository, photos ports.PhotoRepository, avatars ports.AvatarRepository, cache ports.ProfileCache, mediaReader ports.PhotoMediaReader) *ProfileMediaService {
	service := NewProfileMediaService(profiles, photos, avatars, cache)
	service.mediaReader = mediaReader
	return service
}

// NewProfileMediaService создаёт application-сервис медиа профиля.
func NewProfileMediaService(profiles ports.ProfileRepository, photos ports.PhotoRepository, avatars ports.AvatarRepository, cache ports.ProfileCache) *ProfileMediaService {
	return &ProfileMediaService{profiles: profiles, photos: photos, avatars: avatars, cache: cache}
}

// ListPhotos возвращает альбом аватаров и пользовательские фотоальбомы.
func (s *ProfileMediaService) ListPhotos(ctx context.Context, ownerID uuid.UUID, viewerID uuid.UUID) (ProfilePhotosDTO, error) {
	cacheKey := profileMediaCacheKey(ownerID)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached ProfilePhotosDTO
			if json.Unmarshal(payload, &cached) == nil {
				cached.IsOwnProfile = ownerID == viewerID
				return visiblePhotos(cached), nil
			}
		}
	}
	_, err := s.profiles.FindByUserID(ctx, ownerID)
	if err != nil {
		return ProfilePhotosDTO{}, err
	}
	avatars, err := s.avatars.ListHistory(ctx, ownerID)
	if err != nil {
		return ProfilePhotosDTO{}, err
	}
	albums, err := s.photos.ListAlbums(ctx, ownerID)
	if err != nil {
		return ProfilePhotosDTO{}, err
	}

	avatarPhotos := make([]ProfilePhotoDTO, 0, len(avatars))
	for _, avatar := range avatars {
		avatarPhotos = append(avatarPhotos, ProfilePhotoDTO{
			ID: avatar.ID, MediaID: avatar.MediaID, URL: mediaURL(avatar.MediaID), CreatedAt: avatar.CreatedAt,
		})
	}
	result := ProfilePhotosDTO{
		OwnerID: ownerID, IsOwnProfile: ownerID == viewerID,
		Albums: []ProfilePhotoAlbumDTO{{Title: "Аватары профиля", Kind: "avatars", Photos: avatarPhotos}},
	}
	if len(avatars) > 0 {
		result.Albums[0].CreatedAt = avatars[0].CreatedAt
	}
	for _, album := range albums {
		result.Albums = append(result.Albums, toPhotoAlbumDTO(album))
	}
	s.cacheJSON(ctx, cacheKey, result)
	return visiblePhotos(result), nil
}

// CreateAlbum создаёт пользовательский фотоальбом.
func (s *ProfileMediaService) CreateAlbum(ctx context.Context, userID uuid.UUID, title string) (ProfilePhotoAlbumDTO, error) {
	return s.SaveAlbum(ctx, userID, uuid.Nil(), AlbumInput{Title: title})
}

// AlbumInput содержит редактируемые настройки альбома.
type AlbumInput struct {
	Title         string
	Description   string
	Visibility    string
	CommentPolicy string
}

// SaveAlbum создаёт альбом или изменяет настройки альбома владельца.
func (s *ProfileMediaService) SaveAlbum(ctx context.Context, userID, albumID uuid.UUID, input AlbumInput) (ProfilePhotoAlbumDTO, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Visibility == "" {
		input.Visibility = "public"
	}
	if input.CommentPolicy == "" {
		input.CommentPolicy = "public"
	}
	if input.Title == "" || len([]rune(input.Title)) > 128 || len([]rune(input.Description)) > 512 ||
		(input.Visibility != "public" && input.Visibility != "private") ||
		(input.CommentPolicy != "public" && input.CommentPolicy != "private" && input.CommentPolicy != "nobody") {
		return ProfilePhotoAlbumDTO{}, ErrValidation
	}
	album := domain.ProfilePhotoAlbum{ID: albumID, UserID: userID, Title: input.Title, Description: input.Description, Visibility: input.Visibility, CommentPolicy: input.CommentPolicy}
	var err error
	if albumID == uuid.Nil() {
		album.ID = uuid.New()
		album, err = s.photos.CreateAlbum(ctx, album)
	} else {
		var existing domain.ProfilePhotoAlbum
		existing, err = s.photos.FindAlbum(ctx, albumID)
		if err != nil {
			return ProfilePhotoAlbumDTO{}, err
		}
		if existing.UserID != userID {
			return ProfilePhotoAlbumDTO{}, ErrForbidden
		}
		album, err = s.photos.UpdateAlbum(ctx, album)
	}
	if err != nil {
		return ProfilePhotoAlbumDTO{}, err
	}
	s.invalidate(ctx, userID)
	return toPhotoAlbumDTO(album), nil
}

// AddPhoto добавляет ранее загруженное медиа в принадлежащий пользователю альбом.
func (s *ProfileMediaService) AddPhoto(ctx context.Context, userID uuid.UUID, albumID uuid.UUID, mediaID uuid.UUID, caption string) (ProfilePhotoAlbumDTO, error) {
	if mediaID == uuid.Nil() {
		return ProfilePhotoAlbumDTO{}, ErrValidation
	}
	album, err := s.photos.FindAlbum(ctx, albumID)
	if err != nil {
		return ProfilePhotoAlbumDTO{}, err
	}
	if album.UserID != userID {
		return ProfilePhotoAlbumDTO{}, ErrForbidden
	}
	if s.mediaReader != nil {
		allowed, err := s.mediaReader.CanAttachPhoto(ctx, userID, mediaID)
		if err != nil {
			return ProfilePhotoAlbumDTO{}, err
		}
		if !allowed {
			return ProfilePhotoAlbumDTO{}, ErrValidation
		}
	}
	caption = strings.TrimSpace(caption)
	if len([]rune(caption)) > 2000 {
		return ProfilePhotoAlbumDTO{}, ErrValidation
	}
	var captionValue *string
	if caption != "" {
		captionValue = &caption
	}
	updated, err := s.photos.AddPhoto(ctx, domain.ProfilePhoto{
		ID: uuid.New(), AlbumID: albumID, UserID: userID, MediaID: mediaID, Caption: captionValue,
	})
	if errors.Is(err, ports.ErrAlreadyExists) {
		return ProfilePhotoAlbumDTO{}, ErrMediaConflict
	}
	if err != nil {
		return ProfilePhotoAlbumDTO{}, err
	}
	s.invalidate(ctx, userID)
	return toPhotoAlbumDTO(updated), nil
}

// DeletePhoto удаляет фотографию из альбома владельца.
func (s *ProfileMediaService) DeletePhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) error {
	if photoID == uuid.Nil() {
		return ErrValidation
	}
	err := s.photos.DeletePhoto(ctx, userID, photoID)
	if err == nil {
		s.invalidate(ctx, userID)
	}
	return err
}

// SetAvatar назначает медиаобъект текущим аватаром и добавляет его в историю.
func (s *ProfileMediaService) SetAvatar(ctx context.Context, userID uuid.UUID, mediaID uuid.UUID) (AvatarHistoryDTO, error) {
	if mediaID == uuid.Nil() {
		return AvatarHistoryDTO{}, ErrValidation
	}
	if _, err := s.profiles.UpdateAvatar(ctx, userID, mediaURL(mediaID)); err != nil {
		return AvatarHistoryDTO{}, err
	}
	if err := s.avatars.AddToHistory(ctx, domain.ProfileAvatar{ID: uuid.New(), UserID: userID, MediaID: mediaID}); err != nil {
		return AvatarHistoryDTO{}, err
	}
	s.invalidate(ctx, userID)
	return s.AvatarHistory(ctx, userID)
}

// SelectAvatar выбирает ранее сохранённый аватар из истории.
func (s *ProfileMediaService) SelectAvatar(ctx context.Context, userID uuid.UUID, mediaID uuid.UUID) (AvatarHistoryDTO, error) {
	ok, err := s.avatars.HasMedia(ctx, userID, mediaID)
	if err != nil {
		return AvatarHistoryDTO{}, err
	}
	if !ok {
		return AvatarHistoryDTO{}, ErrValidation
	}
	if _, err := s.profiles.UpdateAvatar(ctx, userID, mediaURL(mediaID)); err != nil {
		return AvatarHistoryDTO{}, err
	}
	s.invalidate(ctx, userID)
	return s.AvatarHistory(ctx, userID)
}

// RemoveAvatar возвращает пользователю стабильный default-аватар.
func (s *ProfileMediaService) RemoveAvatar(ctx context.Context, userID uuid.UUID) error {
	_, err := s.profiles.UpdateAvatar(ctx, userID, defaultAvatarURL)
	if err == nil {
		s.invalidate(ctx, userID)
	}
	return err
}

// AvatarHistory возвращает текущий аватар и историю его медиаобъектов.
func (s *ProfileMediaService) AvatarHistory(ctx context.Context, userID uuid.UUID) (AvatarHistoryDTO, error) {
	profile, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return AvatarHistoryDTO{}, err
	}
	history, err := s.avatars.ListHistory(ctx, userID)
	if err != nil {
		return AvatarHistoryDTO{}, err
	}
	result := AvatarHistoryDTO{CurrentURL: profile.AvatarURL, Avatars: make([]AvatarDTO, 0, len(history))}
	for _, avatar := range history {
		url := mediaURL(avatar.MediaID)
		result.Avatars = append(result.Avatars, AvatarDTO{ID: avatar.ID, MediaID: avatar.MediaID, URL: url, CreatedAt: avatar.CreatedAt, IsCurrent: url == profile.AvatarURL})
	}
	return result, nil
}

func toPhotoAlbumDTO(album domain.ProfilePhotoAlbum) ProfilePhotoAlbumDTO {
	id := album.ID
	result := ProfilePhotoAlbumDTO{ID: &id, Title: album.Title, Description: album.Description, Visibility: album.Visibility, CommentPolicy: album.CommentPolicy, Kind: "custom", CreatedAt: album.CreatedAt, Photos: make([]ProfilePhotoDTO, 0, len(album.Photos))}
	for _, photo := range album.Photos {
		albumID := photo.AlbumID
		result.Photos = append(result.Photos, ProfilePhotoDTO{ID: photo.ID, AlbumID: &albumID, MediaID: photo.MediaID, URL: mediaURL(photo.MediaID), Caption: photo.Caption, CreatedAt: photo.CreatedAt, Archived: photo.Archived, Latitude: photo.Latitude, Longitude: photo.Longitude})
	}
	return result
}

func mediaURL(mediaID uuid.UUID) string {
	return "/v1/media/" + mediaID.String() + "/content"
}

func profileMediaCacheKey(userID uuid.UUID) string {
	return "profiles:v2:media:" + userID.String()
}

// visiblePhotos применяет приватность после чтения общей модели из кэша.
// Фильтрация не меняет срезы исходной модели и не кэширует решение о доступе.
func visiblePhotos(result ProfilePhotosDTO) ProfilePhotosDTO {
	if result.IsOwnProfile {
		return result
	}
	albums := make([]ProfilePhotoAlbumDTO, 0, len(result.Albums))
	for _, album := range result.Albums {
		if album.Visibility == "private" {
			continue
		}
		photos := make([]ProfilePhotoDTO, 0, len(album.Photos))
		for _, photo := range album.Photos {
			if !photo.Archived {
				photos = append(photos, photo)
			}
		}
		album.Photos = photos
		albums = append(albums, album)
	}
	result.Albums = albums
	return result
}

func (s *ProfileMediaService) cacheJSON(ctx context.Context, key string, value any) {
	if s.cache == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err == nil {
		_ = s.cache.Set(ctx, key, payload, profileCacheTTL)
	}
}

func (s *ProfileMediaService) invalidate(ctx context.Context, userID uuid.UUID) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, profileMediaCacheKey(userID))
	}
}
