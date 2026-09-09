package postgres

import (
	"context"
	"errors"
	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// CanViewMedia запрещает чужому зрителю файл, связанный с закрытым или архивным фото.
func (r *ProfileMediaRepository) CanViewMedia(ctx context.Context, viewerID, mediaID uuid.UUID) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `SELECT NOT EXISTS (
		SELECT 1 FROM profiles.profile_photos p
		JOIN profiles.profile_photo_albums a ON a.id=p.album_id
		WHERE p.media_id=$1 AND a.user_id<>$2 AND (a.visibility='private' OR p.archived OR p.deleted_at IS NOT NULL)
	)`, mediaID, viewerID).Scan(&allowed)
	return allowed, err
}

// UpdateAlbum сохраняет настройки альбома владельца.
func (r *ProfileMediaRepository) UpdateAlbum(ctx context.Context, album domain.ProfilePhotoAlbum) (domain.ProfilePhotoAlbum, error) {
	result, err := r.pool.Exec(ctx, `UPDATE profiles.profile_photo_albums SET title=$3, description=$4, visibility=$5, comment_policy=$6, updated_at=now() WHERE id=$1 AND user_id=$2`, album.ID, album.UserID, album.Title, album.Description, album.Visibility, album.CommentPolicy)
	if err != nil {
		return domain.ProfilePhotoAlbum{}, err
	}
	if result.RowsAffected() == 0 {
		return domain.ProfilePhotoAlbum{}, ports.ErrNotFound
	}
	return r.FindAlbum(ctx, album.ID)
}

// FindPhoto возвращает фотографию по идентификатору.
func (r *ProfileMediaRepository) FindPhoto(ctx context.Context, photoID uuid.UUID) (domain.ProfilePhoto, error) {
	var photo domain.ProfilePhoto
	err := r.pool.QueryRow(ctx, `SELECT id, album_id, user_id, media_id, caption, created_at, archived, latitude, longitude FROM profiles.profile_photos WHERE id=$1 AND deleted_at IS NULL`, photoID).Scan(&photo.ID, &photo.AlbumID, &photo.UserID, &photo.MediaID, &photo.Caption, &photo.CreatedAt, &photo.Archived, &photo.Latitude, &photo.Longitude)
	if errors.Is(err, pgx.ErrNoRows) {
		return photo, ports.ErrNotFound
	}
	return photo, err
}

// UpdatePhoto сохраняет подпись, координаты и состояние архива фотографии владельца.
func (r *ProfileMediaRepository) UpdatePhoto(ctx context.Context, photo domain.ProfilePhoto) error {
	result, err := r.pool.Exec(ctx, `UPDATE profiles.profile_photos SET caption=$3, archived=$4, latitude=$5, longitude=$6 WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, photo.ID, photo.UserID, photo.Caption, photo.Archived, photo.Latitude, photo.Longitude)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ListComments возвращает последние сто комментариев в хронологическом порядке.
func (r *ProfileMediaRepository) ListComments(ctx context.Context, photoID uuid.UUID) ([]domain.PhotoComment, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, photo_id, user_id, body, created_at FROM (SELECT id, photo_id, user_id, body, created_at FROM profiles.photo_comments WHERE photo_id=$1 ORDER BY created_at DESC, id DESC LIMIT 100) recent ORDER BY created_at, id`, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.PhotoComment, 0)
	for rows.Next() {
		var c domain.PhotoComment
		if err := rows.Scan(&c.ID, &c.PhotoID, &c.UserID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// AddComment сохраняет комментарий к фотографии.
func (r *ProfileMediaRepository) AddComment(ctx context.Context, c domain.PhotoComment) (domain.PhotoComment, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO profiles.photo_comments(id, photo_id, user_id, body) VALUES($1,$2,$3,$4) RETURNING created_at`, c.ID, c.PhotoID, c.UserID, c.Body).Scan(&c.CreatedAt)
	return c, err
}
