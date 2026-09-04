// Package ports содержит интерфейсы, которыми владеет application-слой.
package ports

import (
	"context"
	"errors"
	"time"

	"uuid"

	"general-project/profiles/internal/domain"
)

var (
	// ErrNotFound означает, что профиль отсутствует.
	ErrNotFound = errors.New("profile not found")
	// ErrAlreadyExists означает, что handle уже занят.
	ErrAlreadyExists = errors.New("profile handle already exists")
)

// ProfileRepository предоставляет application-слою доступ к профилям.
type ProfileRepository interface {
	FindByHandle(ctx context.Context, handle string) (domain.Profile, error)
	SearchByHandle(ctx context.Context, query string, limit int) ([]domain.Profile, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (domain.Profile, error)
	SetHandle(ctx context.Context, userID uuid.UUID, handle string) (domain.Profile, error)
	UpdatePublicProfile(ctx context.Context, userID uuid.UUID, displayName string, bio string) (domain.Profile, error)
	UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (domain.Profile, error)
}

// PhotoRepository предоставляет application-слою операции с фотоальбомами.
type PhotoRepository interface {
	ListAlbums(ctx context.Context, userID uuid.UUID) ([]domain.ProfilePhotoAlbum, error)
	CreateAlbum(ctx context.Context, album domain.ProfilePhotoAlbum) (domain.ProfilePhotoAlbum, error)
	FindAlbum(ctx context.Context, albumID uuid.UUID) (domain.ProfilePhotoAlbum, error)
	AddPhoto(ctx context.Context, photo domain.ProfilePhoto) (domain.ProfilePhotoAlbum, error)
	DeletePhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) error
}

// AvatarRepository предоставляет application-слою операции с историей аватаров.
type AvatarRepository interface {
	ListHistory(ctx context.Context, userID uuid.UUID) ([]domain.ProfileAvatar, error)
	AddToHistory(ctx context.Context, avatar domain.ProfileAvatar) error
	HasMedia(ctx context.Context, userID uuid.UUID, mediaID uuid.UUID) (bool, error)
}

// ProfileCache предоставляет application-слою абстракцию кэша read-моделей.
type ProfileCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}
