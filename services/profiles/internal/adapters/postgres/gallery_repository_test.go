package postgres

import (
	"context"
	"general-project/profiles/internal/domain"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestGalleryRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("GALLERY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("GALLERY_TEST_DATABASE_URL не задан")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer pool.Close()
	r := NewProfileMediaRepository(pool)
	owner, viewer, mediaID := uuid.New(), uuid.New(), uuid.New()
	album, err := r.CreateAlbum(ctx, domain.ProfilePhotoAlbum{ID: uuid.New(), UserID: owner, Title: "Фото", Description: "Описание", Visibility: "private", CommentPolicy: "public"})
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM profiles.profile_photo_albums WHERE id=$1", album.ID)
	}()
	album, err = r.AddPhoto(ctx, domain.ProfilePhoto{ID: uuid.New(), AlbumID: album.ID, UserID: owner, MediaID: mediaID})
	require.NoError(t, err)
	require.Len(t, album.Photos, 1)
	listed, err := r.ListAlbums(ctx, owner)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "Описание", listed[0].Description)
	allowed, err := r.CanViewMedia(ctx, viewer, mediaID)
	require.NoError(t, err)
	require.False(t, allowed)
	allowed, err = r.CanViewMedia(ctx, owner, mediaID)
	require.NoError(t, err)
	require.True(t, allowed)
	album.Visibility = "public"
	album, err = r.UpdateAlbum(ctx, album)
	require.NoError(t, err)
	allowed, err = r.CanViewMedia(ctx, viewer, mediaID)
	require.NoError(t, err)
	require.True(t, allowed)
	photo := album.Photos[0]
	photo.Archived = true
	lat, lon := 53.2, 50.1
	photo.Latitude, photo.Longitude = &lat, &lon
	require.NoError(t, r.UpdatePhoto(ctx, photo))
	found, err := r.FindPhoto(ctx, photo.ID)
	require.NoError(t, err)
	require.Equal(t, photo, found)
	allowed, err = r.CanViewMedia(ctx, viewer, mediaID)
	require.NoError(t, err)
	require.False(t, allowed)
	_, err = r.AddComment(ctx, domain.PhotoComment{ID: uuid.New(), PhotoID: photo.ID, UserID: owner, Body: "Комментарий"})
	require.NoError(t, err)
	comments, err := r.ListComments(ctx, photo.ID)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	require.Error(t, r.DeletePhoto(ctx, viewer, photo.ID))
	require.NoError(t, r.DeletePhoto(ctx, owner, photo.ID))
	allowed, err = r.CanViewMedia(ctx, viewer, mediaID)
	require.NoError(t, err)
	require.False(t, allowed)
	_, err = r.FindPhoto(ctx, photo.ID)
	require.Error(t, err)
	comments, err = r.ListComments(ctx, photo.ID)
	require.NoError(t, err)
	require.Empty(t, comments)
}
