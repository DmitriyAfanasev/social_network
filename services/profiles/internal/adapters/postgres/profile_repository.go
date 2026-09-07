// Package postgres содержит PostgreSQL-адаптеры profiles-сервиса.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
)

// ProfileRepository реализует операции профиля через pgx.
type ProfileRepository struct {
	pool *pgxpool.Pool
}

const profileColumns = `user_id, handle, bio, avatar_url, profile_details, profile_privacy, created_at, updated_at`

// NewProfileRepository создаёт PostgreSQL-адаптер профилей.
func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

// FindByHandle загружает публичный профиль по URL-handle.
func (r *ProfileRepository) FindByHandle(ctx context.Context, handle string) (domain.Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM profiles.profiles WHERE lower(handle) = lower($1)`
	return r.findOne(ctx, query, handle)
}

// SearchByHandle ищет профили по началу handle с ограничением результата.
func (r *ProfileRepository) SearchByHandle(ctx context.Context, query string, limit int) ([]domain.Profile, error) {
	const statement = `SELECT ` + profileColumns + ` FROM profiles.profiles
		WHERE lower(handle) LIKE lower($1) || '%'
		ORDER BY lower(handle)
		LIMIT $2`

	rows, err := r.pool.Query(ctx, statement, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]domain.Profile, 0, limit)
	for rows.Next() {
		var profile domain.Profile
		if err := scanProfileRow(rows, &profile); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

// FindByUserID загружает профиль владельца по UUID пользователя.
func (r *ProfileRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM profiles.profiles WHERE user_id = $1`
	return r.findOne(ctx, query, userID)
}

// EnsureByUserID создаёт пустой профиль, если он ещё не существует.
func (r *ProfileRepository) EnsureByUserID(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	const query = `
		INSERT INTO profiles.profiles (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING ` + profileColumns
	return r.scanProfile(ctx, query, userID)
}

// SetHandle создаёт профиль при первом назначении handle или обновляет его.
func (r *ProfileRepository) SetHandle(ctx context.Context, userID uuid.UUID, handle string) (domain.Profile, error) {
	const query = `
		INSERT INTO profiles.profiles (user_id, handle)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET handle = EXCLUDED.handle, updated_at = now()
		RETURNING ` + profileColumns

	var profile domain.Profile
	err := scanProfileRow(r.pool.QueryRow(ctx, query, userID, handle), &profile)
	if isUniqueViolation(err) {
		return domain.Profile{}, ports.ErrAlreadyExists
	}
	if err != nil {
		return domain.Profile{}, err
	}
	return profile, nil
}

// UpdatePublicProfile обновляет описание существующего профиля.
func (r *ProfileRepository) UpdatePublicProfile(ctx context.Context, userID uuid.UUID, bio string) (domain.Profile, error) {
	profile, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Profile{}, err
	}
	profile.Bio = bio
	return r.update(ctx, userID, profile)
}

// UpdateProfileDetails обновляет дополнительные сведения профиля.
func (r *ProfileRepository) UpdateProfileDetails(ctx context.Context, userID uuid.UUID, details domain.ProfileDetails) (domain.Profile, error) {
	profile, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Profile{}, err
	}
	profile.Details = details
	return r.update(ctx, userID, profile)
}

// UpdateProfilePrivacy обновляет политики видимости и взаимодействия профиля.
func (r *ProfileRepository) UpdateProfilePrivacy(ctx context.Context, userID uuid.UUID, privacy domain.ProfilePrivacy) (domain.Profile, error) {
	profile, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Profile{}, err
	}
	profile.Privacy = privacy
	return r.update(ctx, userID, profile)
}

// UpdateAvatar сохраняет URL текущего аватара пользователя.
func (r *ProfileRepository) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (domain.Profile, error) {
	profile, err := r.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Profile{}, err
	}
	profile.AvatarURL = avatarURL
	return r.update(ctx, userID, profile)
}

func (r *ProfileRepository) findOne(ctx context.Context, query string, arg any) (domain.Profile, error) {
	return r.scanProfile(ctx, query, arg)
}

func (r *ProfileRepository) scanProfile(ctx context.Context, query string, args ...any) (domain.Profile, error) {
	var profile domain.Profile
	err := scanProfileRow(r.pool.QueryRow(ctx, query, args...), &profile)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Profile{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.Profile{}, err
	}
	return profile, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfileRow(row rowScanner, profile *domain.Profile) error {
	var detailsJSON []byte
	var privacyJSON []byte
	err := row.Scan(
		&profile.UserID,
		&profile.Handle,
		&profile.Bio,
		&profile.AvatarURL,
		&detailsJSON,
		&privacyJSON,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if len(detailsJSON) > 0 && string(detailsJSON) != "null" {
		if err := json.Unmarshal(detailsJSON, &profile.Details); err != nil {
			return err
		}
	}
	if len(privacyJSON) > 0 && string(privacyJSON) != "null" {
		if err := json.Unmarshal(privacyJSON, &profile.Privacy); err != nil {
			return err
		}
	}
	setPrivacyDefaults(&profile.Privacy)
	return nil
}

func (r *ProfileRepository) update(ctx context.Context, userID uuid.UUID, profile domain.Profile) (domain.Profile, error) {
	detailsJSON, err := json.Marshal(profile.Details)
	if err != nil {
		return domain.Profile{}, err
	}
	privacyJSON, err := json.Marshal(profile.Privacy)
	if err != nil {
		return domain.Profile{}, err
	}
	const query = `
		UPDATE profiles.profiles
		SET bio = $2, avatar_url = $3, profile_details = $4::jsonb, profile_privacy = $5::jsonb,
		    message_policy = $6, updated_at = now()
		WHERE user_id = $1
		RETURNING ` + profileColumns
	return r.scanProfile(ctx, query, userID, profile.Bio, profile.AvatarURL, detailsJSON, privacyJSON, profile.Privacy.MessagePolicy)
}

func setPrivacyDefaults(privacy *domain.ProfilePrivacy) {
	if privacy.ProfileVisibility == "" {
		privacy.ProfileVisibility = "everyone"
	}
	if privacy.FriendRequestPolicy == "" {
		privacy.FriendRequestPolicy = "everyone"
	}
	if privacy.MessagePolicy == "" {
		privacy.MessagePolicy = "everyone"
	}
	if privacy.PhoneVisibility == "" {
		privacy.PhoneVisibility = "everyone"
	}
	if privacy.BirthDateVisibility == "" {
		privacy.BirthDateVisibility = "everyone"
	}
	if privacy.GenderVisibility == "" {
		privacy.GenderVisibility = "everyone"
	}
	if privacy.LocationVisibility == "" {
		privacy.LocationVisibility = "everyone"
	}
	if privacy.StatusVisibility == "" {
		privacy.StatusVisibility = "everyone"
	}
	if privacy.FriendsVisibility == "" {
		privacy.FriendsVisibility = "friends"
	}
	if privacy.PostsVisibility == "" {
		privacy.PostsVisibility = "everyone"
	}
	if privacy.MusicVisibility == "" {
		privacy.MusicVisibility = "friends_of_friends"
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
