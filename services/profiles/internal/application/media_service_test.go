package application

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
)

type fakePhotoRepository struct {
	albums   map[uuid.UUID]domain.ProfilePhotoAlbum
	comments []domain.PhotoComment
}

func (f *fakePhotoRepository) CanViewMedia(_ context.Context, viewerID, mediaID uuid.UUID) (bool, error) {
	for _, album := range f.albums {
		for _, photo := range album.Photos {
			if photo.MediaID == mediaID && album.UserID != viewerID && (album.Visibility == "private" || photo.Archived) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (f *fakePhotoRepository) UpdateAlbum(_ context.Context, album domain.ProfilePhotoAlbum) (domain.ProfilePhotoAlbum, error) {
	existing := f.albums[album.ID]
	album.Photos, album.CreatedAt = existing.Photos, existing.CreatedAt
	f.albums[album.ID] = album
	return album, nil
}

func (f *fakePhotoRepository) FindPhoto(_ context.Context, id uuid.UUID) (domain.ProfilePhoto, error) {
	for _, album := range f.albums {
		for _, photo := range album.Photos {
			if photo.ID == id {
				return photo, nil
			}
		}
	}
	return domain.ProfilePhoto{}, ports.ErrNotFound
}

func (f *fakePhotoRepository) UpdatePhoto(_ context.Context, photo domain.ProfilePhoto) error {
	album := f.albums[photo.AlbumID]
	for i := range album.Photos {
		if album.Photos[i].ID == photo.ID {
			album.Photos[i] = photo
			f.albums[album.ID] = album
			return nil
		}
	}
	return ports.ErrNotFound
}

func (f *fakePhotoRepository) ListComments(_ context.Context, id uuid.UUID) ([]domain.PhotoComment, error) {
	result := []domain.PhotoComment{}
	for _, c := range f.comments {
		if c.PhotoID == id {
			result = append(result, c)
		}
	}
	return result, nil
}

func (f *fakePhotoRepository) AddComment(_ context.Context, c domain.PhotoComment) (domain.PhotoComment, error) {
	c.CreatedAt = time.Now()
	f.comments = append(f.comments, c)
	return c, nil
}

func (f *fakePhotoRepository) ListAlbums(_ context.Context, userID uuid.UUID) ([]domain.ProfilePhotoAlbum, error) {
	result := make([]domain.ProfilePhotoAlbum, 0)
	for _, album := range f.albums {
		if album.UserID == userID {
			result = append(result, album)
		}
	}
	return result, nil
}

func (f *fakePhotoRepository) CreateAlbum(_ context.Context, album domain.ProfilePhotoAlbum) (domain.ProfilePhotoAlbum, error) {
	album.CreatedAt = time.Now().UTC()
	album.UpdatedAt = album.CreatedAt
	f.albums[album.ID] = album
	return album, nil
}

func (f *fakePhotoRepository) FindAlbum(_ context.Context, albumID uuid.UUID) (domain.ProfilePhotoAlbum, error) {
	album, ok := f.albums[albumID]
	if !ok {
		return domain.ProfilePhotoAlbum{}, ports.ErrNotFound
	}
	return album, nil
}

func (f *fakePhotoRepository) AddPhoto(_ context.Context, photo domain.ProfilePhoto) (domain.ProfilePhotoAlbum, error) {
	album, ok := f.albums[photo.AlbumID]
	if !ok {
		return domain.ProfilePhotoAlbum{}, ports.ErrNotFound
	}
	photo.CreatedAt = time.Now().UTC()
	album.Photos = append(album.Photos, photo)
	f.albums[album.ID] = album
	return album, nil
}

func (f *fakePhotoRepository) DeletePhoto(_ context.Context, userID uuid.UUID, photoID uuid.UUID) error {
	for albumID, album := range f.albums {
		if album.UserID != userID {
			continue
		}
		for index, photo := range album.Photos {
			if photo.ID == photoID {
				album.Photos = append(album.Photos[:index], album.Photos[index+1:]...)
				f.albums[albumID] = album
				return nil
			}
		}
	}
	return ports.ErrNotFound
}

type fakeAvatarRepository struct {
	history map[uuid.UUID][]domain.ProfileAvatar
}

func (f *fakeAvatarRepository) ListHistory(_ context.Context, userID uuid.UUID) ([]domain.ProfileAvatar, error) {
	return append([]domain.ProfileAvatar(nil), f.history[userID]...), nil
}

func (f *fakeAvatarRepository) AddToHistory(_ context.Context, avatar domain.ProfileAvatar) error {
	for _, existing := range f.history[avatar.UserID] {
		if existing.MediaID == avatar.MediaID {
			return nil
		}
	}
	avatar.CreatedAt = time.Now().UTC()
	f.history[avatar.UserID] = append([]domain.ProfileAvatar{avatar}, f.history[avatar.UserID]...)
	return nil
}

func (f *fakeAvatarRepository) HasMedia(_ context.Context, userID uuid.UUID, mediaID uuid.UUID) (bool, error) {
	for _, avatar := range f.history[userID] {
		if avatar.MediaID == mediaID {
			return true, nil
		}
	}
	return false, nil
}

func TestProfileMediaServiceProtectsAlbumOwnership(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	otherID := uuid.New()
	profiles := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{
		ownerID: {UserID: ownerID, AvatarURL: defaultAvatarURL},
	}}
	photos := &fakePhotoRepository{albums: map[uuid.UUID]domain.ProfilePhotoAlbum{}}
	service := NewProfileMediaService(profiles, photos, &fakeAvatarRepository{history: map[uuid.UUID][]domain.ProfileAvatar{}}, nil)

	album, err := service.CreateAlbum(context.Background(), ownerID, " Мои фото ")
	require.NoError(t, err)
	require.Equal(t, "Мои фото", album.Title)

	_, err = service.AddPhoto(context.Background(), otherID, *album.ID, uuid.New(), "подпись")
	require.ErrorIs(t, err, ErrForbidden)

	photo, err := service.AddPhoto(context.Background(), ownerID, *album.ID, uuid.New(), " подпись ")
	require.NoError(t, err)
	require.Len(t, photo.Photos, 1)
	require.Equal(t, "подпись", *photo.Photos[0].Caption)
}

func TestProfileMediaServiceMaintainsAvatarHistory(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	mediaID := uuid.New()
	profiles := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{
		userID: {UserID: userID, AvatarURL: defaultAvatarURL},
	}}
	avatars := &fakeAvatarRepository{history: map[uuid.UUID][]domain.ProfileAvatar{}}
	service := NewProfileMediaService(profiles, &fakePhotoRepository{albums: map[uuid.UUID]domain.ProfilePhotoAlbum{}}, avatars, nil)

	history, err := service.SetAvatar(context.Background(), userID, mediaID)
	require.NoError(t, err)
	require.Equal(t, mediaURL(mediaID), history.CurrentURL)
	require.Len(t, history.Avatars, 1)
	require.True(t, history.Avatars[0].IsCurrent)

	_, err = service.SelectAvatar(context.Background(), userID, uuid.New())
	require.ErrorIs(t, err, ErrValidation)

	require.NoError(t, service.RemoveAvatar(context.Background(), userID))
	current, err := service.AvatarHistory(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, defaultAvatarURL, current.CurrentURL)
	require.False(t, current.Avatars[0].IsCurrent)
}

func TestProfileMediaServiceCachesPhotoList(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	profiles := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{
		userID: {UserID: userID, AvatarURL: defaultAvatarURL},
	}}
	photos := &fakePhotoRepository{albums: map[uuid.UUID]domain.ProfilePhotoAlbum{}}
	avatars := &fakeAvatarRepository{history: map[uuid.UUID][]domain.ProfileAvatar{}}
	profileCache := &fakeProfileCache{values: map[string][]byte{}}
	service := NewProfileMediaService(profiles, photos, avatars, profileCache)

	first, err := service.ListPhotos(context.Background(), userID, uuid.New())
	require.NoError(t, err)
	require.Contains(t, profileCache.values, profileMediaCacheKey(userID))

	photos.albums[uuid.New()] = domain.ProfilePhotoAlbum{ID: uuid.New(), UserID: userID, Title: "Новый альбом"}
	second, err := service.ListPhotos(context.Background(), userID, uuid.New())
	require.NoError(t, err)
	require.Equal(t, first.Albums, second.Albums)
}
