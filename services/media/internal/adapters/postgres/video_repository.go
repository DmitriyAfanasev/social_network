package postgres

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

// VideoRepository реализует операции видео и renditions через pgx.
type VideoRepository struct {
	pool *pgxpool.Pool
}

// NewVideoRepository создаёт PostgreSQL-адаптер видео.
func NewVideoRepository(pool *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{pool: pool}
}

// Create сохраняет video-метаданные.
func (r *VideoRepository) Create(ctx context.Context, video domain.VideoAsset) (domain.VideoAsset, error) {
	const query = `
		INSERT INTO media.video_assets (id, media_id, album_id, title, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, media_id, album_id, title, status, duration, created_at, updated_at`
	return r.scanVideo(ctx, query, video.ID, video.MediaID, video.AlbumID, video.Title, video.Status)
}

// FindByID загружает видео, связанное с активным исходным media.
func (r *VideoRepository) FindByID(ctx context.Context, videoID uuid.UUID) (domain.VideoAsset, error) {
	const query = `
		SELECT video.id, video.media_id, video.album_id, video.title, video.status, video.duration,
			video.created_at, video.updated_at
		FROM media.video_assets AS video
		JOIN media.media AS source ON source.id = video.media_id AND source.deleted_at IS NULL
		WHERE video.id = $1`
	return r.scanVideo(ctx, query, videoID)
}

// FindDetails загружает видео с агрегатами interactions и признаками viewer.
func (r *VideoRepository) FindDetails(ctx context.Context, videoID uuid.UUID, viewerID *uuid.UUID) (ports.VideoDetails, error) {
	const query = `
		SELECT video.id, video.media_id, video.album_id, source.uploaded_by,
			video.title, source.original_filename, video.status, video.duration, video.created_at, video.updated_at,
			(SELECT COUNT(*) FROM media.video_views WHERE video_id = video.id),
			(SELECT COUNT(*) FROM media.video_likes WHERE video_id = video.id),
			EXISTS (SELECT 1 FROM media.video_likes WHERE video_id = video.id AND user_id = $2),
			EXISTS (SELECT 1 FROM media.video_bookmarks WHERE video_id = video.id AND user_id = $2),
			EXISTS (SELECT 1 FROM media.video_favorites WHERE video_id = video.id AND user_id = $2)
		FROM media.video_assets AS video
		JOIN media.media AS source ON source.id = video.media_id AND source.deleted_at IS NULL
		WHERE video.id = $1`
	var item ports.VideoListItem
	if err := r.pool.QueryRow(ctx, query, videoID, viewerID).Scan(
		&item.ID, &item.MediaID, &item.AlbumID, &item.OwnerID, &item.Title, &item.OriginalFilename,
		&item.Status, &item.Duration, &item.CreatedAt, &item.UpdatedAt, &item.ViewsCount, &item.LikesCount,
		&item.LikedByViewer, &item.BookmarkedByViewer, &item.FavoritedByViewer,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.VideoDetails{}, ports.ErrNotFound
		}
		return ports.VideoDetails{}, err
	}
	renditions, err := r.ListRenditions(ctx, videoID)
	if err != nil {
		return ports.VideoDetails{}, err
	}
	return ports.VideoDetails{VideoListItem: item, Renditions: renditions}, nil
}

// ListAlbums возвращает видео, сгруппированные по альбомам или вкладке interactions.
func (r *VideoRepository) ListAlbums(ctx context.Context, ownerID uuid.UUID, viewerID uuid.UUID, tab string, search string) ([]ports.VideoAlbumView, error) {
	const query = `
		SELECT video.id, video.media_id, video.album_id, source.uploaded_by,
			video.title, source.original_filename, video.status, video.duration, video.created_at,
			(SELECT COUNT(*) FROM media.video_views WHERE video_id = video.id),
			(SELECT COUNT(*) FROM media.video_likes WHERE video_id = video.id),
			EXISTS (SELECT 1 FROM media.video_likes WHERE video_id = video.id AND user_id = $2),
			EXISTS (SELECT 1 FROM media.video_bookmarks WHERE video_id = video.id AND user_id = $2),
			EXISTS (SELECT 1 FROM media.video_favorites WHERE video_id = video.id AND user_id = $2),
			album.id, album.title
		FROM media.video_assets AS video
		JOIN media.media AS source ON source.id = video.media_id AND source.deleted_at IS NULL
		LEFT JOIN media.video_albums AS album ON album.id = video.album_id
		WHERE (
			($3 = 'uploaded' AND source.uploaded_by = $1)
			OR ($3 = 'favorite' AND EXISTS (SELECT 1 FROM media.video_favorites WHERE video_id = video.id AND user_id = $2))
			OR ($3 = 'viewed' AND EXISTS (SELECT 1 FROM media.video_views WHERE video_id = video.id AND user_id = $2))
			OR ($3 = 'bookmarked' AND EXISTS (SELECT 1 FROM media.video_bookmarks WHERE video_id = video.id AND user_id = $2))
		)
		AND ($4 = '' OR video.title ILIKE '%' || $4 || '%' OR source.original_filename ILIKE '%' || $4 || '%')
		ORDER BY video.created_at DESC, video.id DESC`
	rows, err := r.pool.Query(ctx, query, ownerID, viewerID, tab, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := make([]ports.VideoAlbumView, 0)
	indexes := make(map[uuid.UUID]int)
	for rows.Next() {
		var item ports.VideoListItem
		var albumID *uuid.UUID
		var albumTitle *string
		if err := rows.Scan(
			&item.ID, &item.MediaID, &item.AlbumID, &item.OwnerID, &item.Title, &item.OriginalFilename,
			&item.Status, &item.Duration, &item.CreatedAt, &item.ViewsCount, &item.LikesCount,
			&item.LikedByViewer, &item.BookmarkedByViewer, &item.FavoritedByViewer,
			&albumID, &albumTitle,
		); err != nil {
			return nil, err
		}
		if tab != "uploaded" {
			albumID = nil
			name := map[string]string{"favorite": "Избранное", "viewed": "Просмотренные", "bookmarked": "Смотреть позже"}[tab]
			albumTitle = &name
		}
		key := uuid.Nil()
		if albumID != nil {
			key = *albumID
		}
		index, ok := indexes[key]
		if !ok {
			name := "Без альбома"
			if albumTitle != nil && *albumTitle != "" {
				name = *albumTitle
			}
			grouped = append(grouped, ports.VideoAlbumView{ID: albumID, Title: name, Videos: make([]ports.VideoListItem, 0, 1)})
			index = len(grouped) - 1
			indexes[key] = index
		}
		grouped[index].Videos = append(grouped[index].Videos, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return grouped, nil
}

// CreateAlbum сохраняет альбом видео.
func (r *VideoRepository) CreateAlbum(ctx context.Context, album domain.VideoAlbum) (domain.VideoAlbum, error) {
	const query = `
		INSERT INTO media.video_albums (id, user_id, title)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, title, created_at, updated_at`
	if err := r.pool.QueryRow(ctx, query, album.ID, album.UserID, album.Title).Scan(
		&album.ID, &album.UserID, &album.Title, &album.CreatedAt, &album.UpdatedAt,
	); err != nil {
		return domain.VideoAlbum{}, err
	}
	return album, nil
}

// AlbumBelongsToUser проверяет владельца альбома.
func (r *VideoRepository) AlbumBelongsToUser(ctx context.Context, albumID uuid.UUID, userID uuid.UUID) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM media.video_albums WHERE id = $1 AND user_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, albumID, userID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// DeleteAlbum удаляет альбом с принадлежащими ему видео и возвращает media для storage cleanup.
func (r *VideoRepository) DeleteAlbum(ctx context.Context, albumID uuid.UUID, userID uuid.UUID) ([]ports.VideoDeletion, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var owner uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT user_id FROM media.video_albums WHERE id = $1`, albumID).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if owner != userID {
		return nil, ports.ErrForbidden
	}
	rows, err := tx.Query(ctx, `SELECT id, media_id FROM media.video_assets WHERE album_id = $1`, albumID)
	if err != nil {
		return nil, err
	}
	deletions := make([]ports.VideoDeletion, 0)
	for rows.Next() {
		var deletion ports.VideoDeletion
		if err := rows.Scan(&deletion.VideoID, &deletion.MediaID); err != nil {
			rows.Close()
			return nil, err
		}
		deletions = append(deletions, deletion)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if _, err := tx.Exec(ctx, `DELETE FROM media.video_assets WHERE album_id = $1`, albumID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM media.video_albums WHERE id = $1`, albumID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return deletions, nil
}

// ListRenditions возвращает готовые варианты видео по возрастанию высоты.
func (r *VideoRepository) ListRenditions(ctx context.Context, videoID uuid.UUID) ([]domain.VideoRendition, error) {
	const query = `
		SELECT id, video_id, height, object_key, content_type, size, duration, created_at
		FROM media.video_renditions
		WHERE video_id = $1
		ORDER BY height`
	rows, err := r.pool.Query(ctx, query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	renditions := make([]domain.VideoRendition, 0)
	for rows.Next() {
		var rendition domain.VideoRendition
		if err := rows.Scan(&rendition.ID, &rendition.VideoID, &rendition.Height, &rendition.ObjectKey, &rendition.ContentType, &rendition.Size, &rendition.Duration, &rendition.CreatedAt); err != nil {
			return nil, err
		}
		renditions = append(renditions, rendition)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return renditions, nil
}

// Delete удаляет video-метаданные; renditions удаляются каскадно.
func (r *VideoRepository) Delete(ctx context.Context, videoID uuid.UUID) error {
	const query = `DELETE FROM media.video_assets WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, videoID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// MarkProcessing переводит видео в статус обработки.
func (r *VideoRepository) MarkProcessing(ctx context.Context, videoID uuid.UUID) error {
	const query = `UPDATE media.video_assets SET status = $2, updated_at = now() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, videoID, domain.VideoStatusProcessing)
	return err
}

// Complete сохраняет статус worker и полностью заменяет список renditions.
func (r *VideoRepository) Complete(ctx context.Context, videoID uuid.UUID, status string, duration float64, renditions []domain.VideoRendition, updatedAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const updateQuery = `
		UPDATE media.video_assets
		SET status = $2, duration = $3, updated_at = $4
		WHERE id = $1`
	result, err := tx.Exec(ctx, updateQuery, videoID, status, duration, updatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM media.video_renditions WHERE video_id = $1`, videoID); err != nil {
		return err
	}
	const insertQuery = `
		INSERT INTO media.video_renditions (id, video_id, height, object_key, content_type, size, duration)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for _, rendition := range renditions {
		if _, err := tx.Exec(ctx, insertQuery, rendition.ID, videoID, rendition.Height, rendition.ObjectKey, rendition.ContentType, rendition.Size, rendition.Duration); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RecordView фиксирует просмотр один раз для авторизованного пользователя и возвращает счётчик.
func (r *VideoRepository) RecordView(ctx context.Context, videoID uuid.UUID, userID *uuid.UUID) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM media.video_assets AS video
			JOIN media.media AS source ON source.id = video.media_id AND source.deleted_at IS NULL
			WHERE video.id = $1
		)`, videoID).Scan(&exists); err != nil {
		return 0, err
	}
	if !exists {
		return 0, ports.ErrNotFound
	}
	if userID == nil {
		_, err = tx.Exec(ctx, `INSERT INTO media.video_views (id, video_id) VALUES ($1, $2)`, uuid.New(), videoID)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO media.video_views (id, video_id, user_id) VALUES ($1, $2, $3)
			ON CONFLICT (video_id, user_id) WHERE user_id IS NOT NULL DO NOTHING`, uuid.New(), videoID, *userID)
	}
	if err != nil {
		return 0, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM media.video_views WHERE video_id = $1`, videoID).Scan(&count); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return count, nil
}

// ToggleLike переключает like и возвращает новое состояние и счётчик.
func (r *VideoRepository) ToggleLike(ctx context.Context, videoID uuid.UUID, userID uuid.UUID) (ports.InteractionResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ports.InteractionResult{}, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM media.video_assets AS video
			JOIN media.media AS source ON source.id = video.media_id AND source.deleted_at IS NULL
			WHERE video.id = $1
		)`, videoID).Scan(&exists); err != nil {
		return ports.InteractionResult{}, err
	}
	if !exists {
		return ports.InteractionResult{}, ports.ErrNotFound
	}
	var inserted bool
	if err := tx.QueryRow(ctx, `
		INSERT INTO media.video_likes (id, video_id, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (video_id, user_id) DO NOTHING
		RETURNING true`, uuid.New(), videoID, userID).Scan(&inserted); errors.Is(err, pgx.ErrNoRows) {
		inserted = false
	} else if err != nil {
		return ports.InteractionResult{}, err
	}
	if !inserted {
		if _, err := tx.Exec(ctx, `DELETE FROM media.video_likes WHERE video_id = $1 AND user_id = $2`, videoID, userID); err != nil {
			return ports.InteractionResult{}, err
		}
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM media.video_likes WHERE video_id = $1`, videoID).Scan(&count); err != nil {
		return ports.InteractionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.InteractionResult{}, err
	}
	return ports.InteractionResult{Liked: inserted, LikesCount: count}, nil
}

// SetBookmark добавляет или удаляет закладку видео.
func (r *VideoRepository) SetBookmark(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error) {
	return r.setInteraction(ctx, "video_bookmarks", videoID, userID, value)
}

// SetFavorite добавляет или удаляет видео из избранного.
func (r *VideoRepository) SetFavorite(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error) {
	return r.setInteraction(ctx, "video_favorites", videoID, userID, value)
}

func (r *VideoRepository) setInteraction(ctx context.Context, table string, videoID uuid.UUID, userID uuid.UUID, value bool) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM media.video_assets AS video
			JOIN media.media AS source ON source.id = video.media_id AND source.deleted_at IS NULL
			WHERE video.id = $1
		)`, videoID).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, ports.ErrNotFound
	}
	if value {
		_, err := r.pool.Exec(ctx, `INSERT INTO media.`+table+` (id, video_id, user_id) VALUES ($1, $2, $3) ON CONFLICT (video_id, user_id) DO NOTHING`, uuid.New(), videoID, userID)
		return err == nil, err
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM media.`+table+` WHERE video_id = $1 AND user_id = $2`, videoID, userID)
	return false, err
}

func (r *VideoRepository) scanVideo(ctx context.Context, query string, args ...any) (domain.VideoAsset, error) {
	var video domain.VideoAsset
	err := r.pool.QueryRow(ctx, query, args...).Scan(&video.ID, &video.MediaID, &video.AlbumID, &video.Title, &video.Status, &video.Duration, &video.CreatedAt, &video.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.VideoAsset{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.VideoAsset{}, err
	}
	return video, nil
}
