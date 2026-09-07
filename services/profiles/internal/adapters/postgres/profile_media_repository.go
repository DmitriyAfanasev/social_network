// Package postgres содержит PostgreSQL-адаптеры profiles-сервиса.
package postgres

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
)

// ProfileMediaRepository реализует операции фотоальбомов и аватаров через pgx.
type ProfileMediaRepository struct {
	pool *pgxpool.Pool
}

// NewProfileMediaRepository создаёт PostgreSQL-адаптер фотоальбомов и аватаров.
func NewProfileMediaRepository(pool *pgxpool.Pool) *ProfileMediaRepository {
	return &ProfileMediaRepository{pool: pool}
}

// ListAlbums возвращает альбомы пользователя вместе с фотографиями.
func (r *ProfileMediaRepository) ListAlbums(ctx context.Context, userID uuid.UUID) ([]domain.ProfilePhotoAlbum, error) {
	const query = `
		SELECT id, user_id, title, created_at, updated_at, description, visibility, comment_policy
		FROM profiles.profile_photo_albums
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	albums := make([]domain.ProfilePhotoAlbum, 0)
	for rows.Next() {
		var album domain.ProfilePhotoAlbum
		if err := rows.Scan(&album.ID, &album.UserID, &album.Title, &album.CreatedAt, &album.UpdatedAt, &album.Description, &album.Visibility, &album.CommentPolicy); err != nil {
			return nil, err
		}
		albums = append(albums, album)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Освобождаем соединение до вложенных запросов, в том числе при MaxConns=1.
	rows.Close()
	for i := range albums {
		albums[i].Photos, err = r.listPhotos(ctx, albums[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return albums, nil
}

// CreateAlbum создаёт фотоальбом пользователя.
func (r *ProfileMediaRepository) CreateAlbum(ctx context.Context, album domain.ProfilePhotoAlbum) (domain.ProfilePhotoAlbum, error) {
	const query = `
		INSERT INTO profiles.profile_photo_albums (id, user_id, title, description, visibility, comment_policy)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, title, created_at, updated_at, description, visibility, comment_policy`
	var result domain.ProfilePhotoAlbum
	err := r.pool.QueryRow(ctx, query, album.ID, album.UserID, album.Title, album.Description, album.Visibility, album.CommentPolicy).Scan(
		&result.ID, &result.UserID, &result.Title, &result.CreatedAt, &result.UpdatedAt, &result.Description, &result.Visibility, &result.CommentPolicy,
	)
	if err != nil {
		return domain.ProfilePhotoAlbum{}, err
	}
	result.Photos = []domain.ProfilePhoto{}
	return result, nil
}

// FindAlbum возвращает альбом с фотографиями.
func (r *ProfileMediaRepository) FindAlbum(ctx context.Context, albumID uuid.UUID) (domain.ProfilePhotoAlbum, error) {
	const query = `
		SELECT id, user_id, title, created_at, updated_at, description, visibility, comment_policy
		FROM profiles.profile_photo_albums
		WHERE id = $1`
	var album domain.ProfilePhotoAlbum
	err := r.pool.QueryRow(ctx, query, albumID).Scan(
		&album.ID, &album.UserID, &album.Title, &album.CreatedAt, &album.UpdatedAt, &album.Description, &album.Visibility, &album.CommentPolicy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ProfilePhotoAlbum{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.ProfilePhotoAlbum{}, err
	}
	album.Photos, err = r.listPhotos(ctx, album.ID)
	if err != nil {
		return domain.ProfilePhotoAlbum{}, err
	}
	return album, nil
}

// AddPhoto добавляет ссылку на медиаобъект в альбом владельца.
func (r *ProfileMediaRepository) AddPhoto(ctx context.Context, photo domain.ProfilePhoto) (domain.ProfilePhotoAlbum, error) {
	const query = `
		INSERT INTO profiles.profile_photos (id, album_id, user_id, media_id, caption)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := r.pool.Exec(ctx, query, photo.ID, photo.AlbumID, photo.UserID, photo.MediaID, photo.Caption); err != nil {
		if isUniqueViolation(err) {
			return domain.ProfilePhotoAlbum{}, ports.ErrAlreadyExists
		}
		return domain.ProfilePhotoAlbum{}, err
	}
	return r.FindAlbum(ctx, photo.AlbumID)
}

// DeletePhoto удаляет фотографию только у указанного владельца.
func (r *ProfileMediaRepository) DeletePhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) error {
	// Метаданные сохраняют запрет доступа к удалённому приватному файлу по старой ссылке.
	const query = `WITH removed AS (
		UPDATE profiles.profile_photos SET deleted_at=now()
		WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL RETURNING id
	), comments AS (
		DELETE FROM profiles.photo_comments WHERE photo_id IN (SELECT id FROM removed)
	) SELECT count(*) FROM removed`
	var affected int
	err := r.pool.QueryRow(ctx, query, photoID, userID).Scan(&affected)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ListHistory возвращает историю аватаров пользователя от новых к старым.
func (r *ProfileMediaRepository) ListHistory(ctx context.Context, userID uuid.UUID) ([]domain.ProfileAvatar, error) {
	const query = `
		SELECT id, user_id, media_id, created_at
		FROM profiles.profile_avatars
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.ProfileAvatar, 0)
	for rows.Next() {
		var avatar domain.ProfileAvatar
		if err := rows.Scan(&avatar.ID, &avatar.UserID, &avatar.MediaID, &avatar.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, avatar)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// AddToHistory сохраняет новый уникальный аватар пользователя.
func (r *ProfileMediaRepository) AddToHistory(ctx context.Context, avatar domain.ProfileAvatar) error {
	const query = `
		INSERT INTO profiles.profile_avatars (id, user_id, media_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, media_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, avatar.ID, avatar.UserID, avatar.MediaID)
	return err
}

// HasMedia проверяет, что медиаобъект уже есть в истории аватаров пользователя.
func (r *ProfileMediaRepository) HasMedia(ctx context.Context, userID uuid.UUID, mediaID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM profiles.profile_avatars WHERE user_id = $1 AND media_id = $2
		)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, mediaID).Scan(&exists)
	return exists, err
}

func (r *ProfileMediaRepository) listPhotos(ctx context.Context, albumID uuid.UUID) ([]domain.ProfilePhoto, error) {
	const query = `
		SELECT id, album_id, user_id, media_id, caption, created_at, archived, latitude, longitude
		FROM profiles.profile_photos
		WHERE album_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC`
	rows, err := r.pool.Query(ctx, query, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photos := make([]domain.ProfilePhoto, 0)
	for rows.Next() {
		var photo domain.ProfilePhoto
		if err := rows.Scan(&photo.ID, &photo.AlbumID, &photo.UserID, &photo.MediaID, &photo.Caption, &photo.CreatedAt, &photo.Archived, &photo.Latitude, &photo.Longitude); err != nil {
			return nil, err
		}
		photos = append(photos, photo)
	}
	return photos, rows.Err()
}
