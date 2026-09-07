package application

import (
	"time"

	"uuid"

	"general-project/profiles/internal/domain"
)

// ProfileDTO представляет безопасный результат application-сценария профиля.
type ProfileDTO struct {
	UserID    uuid.UUID
	Handle    *string
	Bio       string
	AvatarURL string
	Details   ProfileDetailsDTO
	Privacy   ProfilePrivacyDTO
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProfileDetailsDTO представляет дополнительные сведения профиля на границе application-слоя.
type ProfileDetailsDTO struct {
	FirstName   string
	LastName    string
	MiddleName  string
	BirthDate   string
	Gender      string
	PhoneNumber string
	Country     string
	City        string
	Street      string
	Status      string
}

// ProfilePrivacyDTO представляет политики видимости и взаимодействия профиля.
type ProfilePrivacyDTO struct {
	ProfileVisibility   string
	FriendRequestPolicy string
	MessagePolicy       string
	PhoneVisibility     string
	BirthDateVisibility string
	GenderVisibility    string
	LocationVisibility  string
	StatusVisibility    string
	FriendsVisibility   string
	PostsVisibility     string
	MusicVisibility     string
}

func toProfileDTO(profile domain.Profile) ProfileDTO {
	return ProfileDTO{
		UserID:    profile.UserID,
		Handle:    profile.Handle,
		Bio:       profile.Bio,
		AvatarURL: profile.AvatarURL,
		Details: ProfileDetailsDTO{
			FirstName: profile.Details.FirstName, LastName: profile.Details.LastName, MiddleName: profile.Details.MiddleName,
			BirthDate: profile.Details.BirthDate, Gender: profile.Details.Gender, PhoneNumber: profile.Details.PhoneNumber,
			Country: profile.Details.Country, City: profile.Details.City, Street: profile.Details.Street, Status: profile.Details.Status,
		},
		Privacy: ProfilePrivacyDTO{
			ProfileVisibility: profile.Privacy.ProfileVisibility, FriendRequestPolicy: profile.Privacy.FriendRequestPolicy,
			MessagePolicy: profile.Privacy.MessagePolicy, PhoneVisibility: profile.Privacy.PhoneVisibility,
			BirthDateVisibility: profile.Privacy.BirthDateVisibility, GenderVisibility: profile.Privacy.GenderVisibility,
			LocationVisibility: profile.Privacy.LocationVisibility, StatusVisibility: profile.Privacy.StatusVisibility,
			FriendsVisibility: profile.Privacy.FriendsVisibility, PostsVisibility: profile.Privacy.PostsVisibility,
			MusicVisibility: profile.Privacy.MusicVisibility,
		},
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}

// ProfilePhotoDTO представляет фотографию в application-ответе.
type ProfilePhotoDTO struct {
	Archived  bool
	Latitude  *float64
	Longitude *float64
	ID        uuid.UUID
	AlbumID   *uuid.UUID
	MediaID   uuid.UUID
	URL       string
	Caption   *string
	CreatedAt time.Time
}

// ProfilePhotoAlbumDTO представляет фотоальбом в application-ответе.
type ProfilePhotoAlbumDTO struct {
	Description   string
	Visibility    string
	CommentPolicy string
	ID            *uuid.UUID
	Title         string
	Kind          string
	CreatedAt     time.Time
	Photos        []ProfilePhotoDTO
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
