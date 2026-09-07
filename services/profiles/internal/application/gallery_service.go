package application

import (
	"context"
	"general-project/profiles/internal/domain"
	"math"
	"strings"
	"time"
	"uuid"
)

// CanViewPhotoMedia проверяет актуальные ограничения файла без кэширования решения.
func (s *ProfileMediaService) CanViewPhotoMedia(ctx context.Context, viewerID, mediaID uuid.UUID) (bool, error) {
	return s.photos.CanViewMedia(ctx, viewerID, mediaID)
}

// PhotoUpdateInput содержит полное редактируемое состояние фотографии.
type PhotoUpdateInput struct {
	Caption   string
	Archived  bool
	Latitude  *float64
	Longitude *float64
}

// UpdatePhoto сохраняет изменения фотографии после проверки владельца.
func (s *ProfileMediaService) UpdatePhoto(ctx context.Context, userID, photoID uuid.UUID, input PhotoUpdateInput) error {
	photo, err := s.photos.FindPhoto(ctx, photoID)
	if err != nil {
		return err
	}
	if photo.UserID != userID {
		return ErrForbidden
	}
	input.Caption = strings.TrimSpace(input.Caption)
	if len([]rune(input.Caption)) > 2000 || (input.Latitude == nil) != (input.Longitude == nil) {
		return ErrValidation
	}
	if input.Latitude != nil && (!validCoordinate(*input.Latitude, 90) || !validCoordinate(*input.Longitude, 180)) {
		return ErrValidation
	}
	photo.Caption = nil
	if input.Caption != "" {
		photo.Caption = &input.Caption
	}
	photo.Archived, photo.Latitude, photo.Longitude = input.Archived, input.Latitude, input.Longitude
	if err := s.photos.UpdatePhoto(ctx, photo); err != nil {
		return err
	}
	s.invalidate(ctx, userID)
	return nil
}

func validCoordinate(value, limit float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -limit && value <= limit
}

// PhotoCommentDTO содержит комментарий и отображаемое имя его автора.
type PhotoCommentDTO struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

func (s *ProfileMediaService) accessiblePhoto(ctx context.Context, viewerID, photoID uuid.UUID) (domain.ProfilePhotoAlbum, error) {
	photo, err := s.photos.FindPhoto(ctx, photoID)
	if err != nil {
		return domain.ProfilePhotoAlbum{}, err
	}
	album, err := s.photos.FindAlbum(ctx, photo.AlbumID)
	if err != nil {
		return album, err
	}
	if album.UserID != viewerID && (album.Visibility == "private" || photo.Archived) {
		return album, ErrForbidden
	}
	return album, nil
}

// ListPhotoComments проверяет доступ к фото и возвращает его комментарии.
func (s *ProfileMediaService) ListPhotoComments(ctx context.Context, viewerID, photoID uuid.UUID) ([]PhotoCommentDTO, error) {
	if _, err := s.accessiblePhoto(ctx, viewerID, photoID); err != nil {
		return nil, err
	}
	comments, err := s.photos.ListComments(ctx, photoID)
	if err != nil {
		return nil, err
	}
	result := make([]PhotoCommentDTO, 0, len(comments))
	names := make(map[uuid.UUID]string)
	for _, c := range comments {
		name, ok := names[c.UserID]
		if !ok {
			name = "Пользователь"
			if profile, err := s.profiles.FindByUserID(ctx, c.UserID); err == nil {
				if full := strings.TrimSpace(profile.Details.FirstName + " " + profile.Details.LastName); full != "" {
					name = full
				}
			}
			names[c.UserID] = name
		}
		result = append(result, PhotoCommentDTO{ID: c.ID, UserID: c.UserID, AuthorName: name, Body: c.Body, CreatedAt: c.CreatedAt})
	}
	return result, nil
}

// AddPhotoComment проверяет настройки комментирования и сохраняет комментарий.
func (s *ProfileMediaService) AddPhotoComment(ctx context.Context, userID, photoID uuid.UUID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > 2000 {
		return ErrValidation
	}
	album, err := s.accessiblePhoto(ctx, userID, photoID)
	if err != nil {
		return err
	}
	if album.CommentPolicy == "nobody" || (album.CommentPolicy == "private" && album.UserID != userID) {
		return ErrForbidden
	}
	_, err = s.photos.AddComment(ctx, domain.PhotoComment{ID: uuid.New(), PhotoID: photoID, UserID: userID, Body: body})
	return err
}
