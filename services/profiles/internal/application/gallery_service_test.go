package application

import (
	"context"
	"general-project/profiles/internal/domain"
	"github.com/stretchr/testify/require"
	"math"
	"strings"
	"testing"
	"uuid"
)

func TestGalleryPrivacyAndArchiveAreAppliedAfterCache(t *testing.T) {
	owner, viewer := uuid.New(), uuid.New()
	ctx := context.Background()
	photos := &fakePhotoRepository{albums: map[uuid.UUID]domain.ProfilePhotoAlbum{}}
	cache := &fakeProfileCache{values: map[string][]byte{}}
	s := NewProfileMediaService(&fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{owner: {UserID: owner}}}, photos, &fakeAvatarRepository{}, cache)
	album, err := s.SaveAlbum(ctx, owner, uuid.Nil(), AlbumInput{Title: "Отпуск", Visibility: "private", Description: "Море"})
	require.NoError(t, err)
	album, err = s.AddPhoto(ctx, owner, *album.ID, uuid.New(), "")
	require.NoError(t, err)
	// Сначала кэш заполняется владельцем; чужой запрос не должен получить закрытый альбом.
	mine, err := s.ListPhotos(ctx, owner, owner)
	require.NoError(t, err)
	require.Len(t, mine.Albums, 2)
	other, err := s.ListPhotos(ctx, owner, viewer)
	require.NoError(t, err)
	require.Len(t, other.Albums, 1)
	_, err = s.SaveAlbum(ctx, viewer, *album.ID, AlbumInput{Title: "Чужой"})
	require.ErrorIs(t, err, ErrForbidden)
	_, err = s.SaveAlbum(ctx, owner, *album.ID, AlbumInput{Title: "Отпуск", Visibility: "public"})
	require.NoError(t, err)
	photo := album.Photos[0]
	require.NoError(t, s.UpdatePhoto(ctx, owner, photo.ID, PhotoUpdateInput{Archived: true}))
	other, err = s.ListPhotos(ctx, owner, viewer)
	require.NoError(t, err)
	require.Len(t, other.Albums, 2)
	require.Empty(t, other.Albums[1].Photos)
	// Кэш, прочитанный гостем, сохраняет исходные данные для владельца.
	mine, err = s.ListPhotos(ctx, owner, owner)
	require.NoError(t, err)
	require.Len(t, mine.Albums[1].Photos, 1)
	require.True(t, mine.Albums[1].Photos[0].Archived)
	require.NoError(t, s.UpdatePhoto(ctx, owner, photo.ID, PhotoUpdateInput{Archived: false}))
	other, err = s.ListPhotos(ctx, owner, viewer)
	require.NoError(t, err)
	require.Len(t, other.Albums[1].Photos, 1)
}

func TestGalleryValidationAndCommentPermissions(t *testing.T) {
	owner, viewer := uuid.New(), uuid.New()
	ctx := context.Background()
	photos := &fakePhotoRepository{albums: map[uuid.UUID]domain.ProfilePhotoAlbum{}}
	s := NewProfileMediaService(&fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{owner: {UserID: owner}, viewer: {UserID: viewer}}}, photos, &fakeAvatarRepository{}, nil)
	for _, input := range []AlbumInput{{Title: " "}, {Title: strings.Repeat("я", 129)}, {Title: "a", Description: strings.Repeat("я", 513)}, {Title: "a", Visibility: "invalid"}, {Title: "a", CommentPolicy: "invalid"}} {
		_, err := s.SaveAlbum(ctx, owner, uuid.Nil(), input)
		require.ErrorIs(t, err, ErrValidation)
	}
	album, err := s.SaveAlbum(ctx, owner, uuid.Nil(), AlbumInput{Title: strings.Repeat("я", 128), Description: strings.Repeat("я", 512)})
	require.NoError(t, err)
	album, err = s.AddPhoto(ctx, owner, *album.ID, uuid.New(), "")
	require.NoError(t, err)
	photo := album.Photos[0]
	lat, lon := 53.2, 50.1
	require.ErrorIs(t, s.UpdatePhoto(ctx, viewer, photo.ID, PhotoUpdateInput{}), ErrForbidden)
	require.ErrorIs(t, s.UpdatePhoto(ctx, owner, photo.ID, PhotoUpdateInput{Latitude: &lat}), ErrValidation)
	bad := math.NaN()
	require.ErrorIs(t, s.UpdatePhoto(ctx, owner, photo.ID, PhotoUpdateInput{Latitude: &bad, Longitude: &lon}), ErrValidation)
	require.NoError(t, s.UpdatePhoto(ctx, owner, photo.ID, PhotoUpdateInput{Latitude: &lat, Longitude: &lon, Caption: " Город "}))
	require.NoError(t, s.AddPhotoComment(ctx, viewer, photo.ID, " Красиво! "))
	comments, err := s.ListPhotoComments(ctx, owner, photo.ID)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	require.Equal(t, "Красиво!", comments[0].Body)
	require.ErrorIs(t, s.AddPhotoComment(ctx, owner, photo.ID, " "), ErrValidation)
	_, err = s.SaveAlbum(ctx, owner, *album.ID, AlbumInput{Title: "a", CommentPolicy: "private"})
	require.NoError(t, err)
	require.ErrorIs(t, s.AddPhotoComment(ctx, viewer, photo.ID, "Нет доступа"), ErrForbidden)
	require.NoError(t, s.AddPhotoComment(ctx, owner, photo.ID, "Мой комментарий"))
	_, err = s.SaveAlbum(ctx, owner, *album.ID, AlbumInput{Title: "a", Visibility: "private"})
	require.NoError(t, err)
	_, err = s.ListPhotoComments(ctx, viewer, photo.ID)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = s.SaveAlbum(ctx, owner, *album.ID, AlbumInput{Title: "a", CommentPolicy: "nobody"})
	require.NoError(t, err)
	require.ErrorIs(t, s.AddPhotoComment(ctx, owner, photo.ID, "Отключено"), ErrForbidden)
}
