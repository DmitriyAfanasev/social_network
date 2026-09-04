package application

import (
	"time"

	"uuid"

	"general-project/profiles/internal/domain"
)

// ProfileDTO представляет безопасный результат application-сценария профиля.
type ProfileDTO struct {
	UserID      uuid.UUID
	Handle      *string
	DisplayName string
	Bio         string
	AvatarURL   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toProfileDTO(profile domain.Profile) ProfileDTO {
	return ProfileDTO{
		UserID:      profile.UserID,
		Handle:      profile.Handle,
		DisplayName: profile.DisplayName,
		Bio:         profile.Bio,
		AvatarURL:   profile.AvatarURL,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
	}
}

// ProfilePhotoDTO представляет фотографию в application-ответе.
type ProfilePhotoDTO struct {
	ID        uuid.UUID
	AlbumID   *uuid.UUID
	MediaID   uuid.UUID
	URL       string
	Caption   *string
	CreatedAt time.Time
}

// ProfilePhotoAlbumDTO представляет фотоальбом в application-ответе.
type ProfilePhotoAlbumDTO struct {
	ID        *uuid.UUID
	Title     string
	Kind      string
	CreatedAt time.Time
	Photos    []ProfilePhotoDTO
}

// ProfilePhotosDTO представляет публичную модель фотоальбомов профиля.
type ProfilePhotosDTO struct {
	OwnerID      uuid.UUID
	IsOwnProfile bool
	Albums       []ProfilePhotoAlbumDTO
}

// AvatarDTO представляет запись истории аватаров.
type AvatarDTO struct {
	ID        uuid.UUID
	MediaID   uuid.UUID
	URL       string
	CreatedAt time.Time
	IsCurrent bool
}

// AvatarHistoryDTO представляет текущий аватар и его историю.
type AvatarHistoryDTO struct {
	CurrentURL string
	Avatars    []AvatarDTO
}
