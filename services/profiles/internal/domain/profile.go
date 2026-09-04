// Package domain содержит сущности и правила profiles-сервиса.
package domain

import (
	"time"

	"uuid"
)

// Profile описывает публичные данные профиля пользователя.
type Profile struct {
	UserID      uuid.UUID
	Handle      *string
	DisplayName string
	Bio         string
	AvatarURL   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProfilePhotoAlbum описывает принадлежащий пользователю фотоальбом.
type ProfilePhotoAlbum struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Photos    []ProfilePhoto
}

// ProfilePhoto описывает фотографию, связанную с медиаобъектом.
type ProfilePhoto struct {
	ID        uuid.UUID
	AlbumID   uuid.UUID
	UserID    uuid.UUID
	MediaID   uuid.UUID
	Caption   *string
	CreatedAt time.Time
}

// ProfileAvatar описывает медиаобъект из истории аватаров пользователя.
type ProfileAvatar struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	MediaID   uuid.UUID
	CreatedAt time.Time
}
