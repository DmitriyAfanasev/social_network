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
	Details     ProfileDetails
	Privacy     ProfilePrivacy
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProfileDetails содержит дополнительные сведения, которые пользователь указывает в профиле.
type ProfileDetails struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	MiddleName  string `json:"middle_name"`
	BirthDate   string `json:"birth_date"`
	Gender      string `json:"gender"`
	PhoneNumber string `json:"phone_number"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Street      string `json:"street"`
	Status      string `json:"status"`
}

// ProfilePrivacy содержит политики доступа к профилю и его разделам.
type ProfilePrivacy struct {
	ProfileVisibility   string `json:"profile_visibility"`
	FriendRequestPolicy string `json:"friend_request_policy"`
	MessagePolicy       string `json:"message_policy"`
	PhoneVisibility     string `json:"phone_visibility"`
	BirthDateVisibility string `json:"birth_date_visibility"`
	GenderVisibility    string `json:"gender_visibility"`
	LocationVisibility  string `json:"location_visibility"`
	StatusVisibility    string `json:"status_visibility"`
	FriendsVisibility   string `json:"friends_visibility"`
	PostsVisibility     string `json:"posts_visibility"`
	MusicVisibility     string `json:"music_visibility"`
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
