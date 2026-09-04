package postgres

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

// MusicRepository реализует операции музыкальных треков через pgx.
type MusicRepository struct {
	pool *pgxpool.Pool
}

// NewMusicRepository создаёт PostgreSQL-адаптер музыкальных треков.
func NewMusicRepository(pool *pgxpool.Pool) *MusicRepository {
	return &MusicRepository{pool: pool}
}

// Create сохраняет метаданные музыкального трека.
func (r *MusicRepository) Create(ctx context.Context, track domain.MusicTrack) (domain.MusicTrack, error) {
	const query = `
		INSERT INTO media.music_tracks (id, media_id, user_id, title, artist)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, media_id, user_id, title, artist, created_at`
	return r.scanCreated(ctx, query, track.ID, track.MediaID, track.UserID, track.Title, track.Artist)
}

// ListByUser возвращает активные музыкальные треки пользователя.
func (r *MusicRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.MusicTrack, error) {
	const query = `
		SELECT track.id, track.media_id, track.user_id, track.title, track.artist,
			media.duration, track.created_at
		FROM media.music_tracks AS track
		JOIN media.media AS media ON media.id = track.media_id AND media.deleted_at IS NULL
		WHERE track.user_id = $1
		ORDER BY track.created_at DESC, track.id DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := make([]domain.MusicTrack, 0)
	for rows.Next() {
		var track domain.MusicTrack
		if err := rows.Scan(&track.ID, &track.MediaID, &track.UserID, &track.Title, &track.Artist, &track.Duration, &track.CreatedAt); err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

// FindByID загружает трек только вместе с активным media-объектом.
func (r *MusicRepository) FindByID(ctx context.Context, trackID uuid.UUID) (domain.MusicTrack, error) {
	const query = `
		SELECT track.id, track.media_id, track.user_id, track.title, track.artist,
			media.duration, track.created_at
		FROM media.music_tracks AS track
		JOIN media.media AS media ON media.id = track.media_id AND media.deleted_at IS NULL
		WHERE track.id = $1`
	return r.scanOne(ctx, query, trackID)
}

// Delete удаляет метаданные музыкального трека.
func (r *MusicRepository) Delete(ctx context.Context, trackID uuid.UUID) error {
	const query = `DELETE FROM media.music_tracks WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, trackID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *MusicRepository) scanOne(ctx context.Context, query string, args ...any) (domain.MusicTrack, error) {
	var track domain.MusicTrack
	err := r.pool.QueryRow(ctx, query, args...).Scan(&track.ID, &track.MediaID, &track.UserID, &track.Title, &track.Artist, &track.Duration, &track.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MusicTrack{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.MusicTrack{}, err
	}
	return track, nil
}

func (r *MusicRepository) scanCreated(ctx context.Context, query string, args ...any) (domain.MusicTrack, error) {
	var track domain.MusicTrack
	err := r.pool.QueryRow(ctx, query, args...).Scan(&track.ID, &track.MediaID, &track.UserID, &track.Title, &track.Artist, &track.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MusicTrack{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.MusicTrack{}, err
	}
	return track, nil
}
