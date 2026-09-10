package postgres

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"general-project/profiles/internal/ports"
)

func TestProfileRepositoryMapsProfileAndJSONPolicies(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewProfileRepository(db)
	userID := uuid.New()
	handle := "tester"
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.ExpectQuery("SELECT user_id, handle, bio").WithArgs(userID).WillReturnRows(
		pgxmock.NewRows([]string{"user_id", "handle", "bio", "avatar_url", "profile_details", "profile_privacy", "created_at", "updated_at"}).
			AddRow(userID, &handle, "bio", "/avatar", []byte(`{"first_name":"Ada"}`), []byte(`{"music_visibility":"friends"}`), now, now),
	)

	// Act
	profile, err := repository.FindByUserID(context.Background(), userID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, userID, profile.UserID)
	require.Equal(t, "Ada", profile.Details.FirstName)
	require.Equal(t, "friends", profile.Privacy.MusicVisibility)
	require.Equal(t, "everyone", profile.Privacy.ProfileVisibility)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestProfileRepositoryMapsNoRowsToNotFound(t *testing.T) {
	// Arrange
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repository := NewProfileRepository(db)
	userID := uuid.New()
	db.ExpectQuery("SELECT user_id, handle, bio").WithArgs(userID).WillReturnError(pgx.ErrNoRows)

	// Act
	_, err = repository.FindByUserID(context.Background(), userID)

	// Assert
	require.ErrorIs(t, err, ports.ErrNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}
