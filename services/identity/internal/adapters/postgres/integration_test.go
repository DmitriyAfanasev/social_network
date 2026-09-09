package postgres

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"general-project/identity/internal/domain"
	"general-project/identity/internal/ports"
)

func TestPostgresIdentityRepositories(t *testing.T) {
	databaseURL := os.Getenv("IDENTITY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("IDENTITY_TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	userRepository := NewUserRepository(pool)
	refreshTokens := NewRefreshTokenStore(pool)
	verificationTokens := NewVerificationTokenStore(pool)
	userID := uuid.New()
	user := domain.User{ID: userID, Email: "identity-integration-" + userID.String() + "@example.test", Status: domain.UserStatusPending}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM identity.users WHERE id = $1`, userID); err != nil {
			t.Logf("cleanup identity user: %v", err)
		}
	})

	created, err := userRepository.Create(ctx, user, "bcrypt-hash", nil)
	require.NoError(t, err)
	require.Equal(t, userID, created.ID)
	require.NotZero(t, created.CreatedAt)

	loaded, err := userRepository.FindByID(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, domain.UserStatusPending, loaded.Status)
	require.Equal(t, user.Email, loaded.Email)

	loadedAuth, err := userRepository.FindByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, "bcrypt-hash", loadedAuth.PasswordHash)

	active, err := userRepository.Activate(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, domain.UserStatusActive, active.Status)
	require.ErrorIs(t, func() error {
		_, activateErr := userRepository.Activate(ctx, userID)
		return activateErr
	}(), ports.ErrNotFound)

	require.NoError(t, userRepository.UpdatePassword(ctx, userID, "new-bcrypt-hash"))
	loadedAuth, err = userRepository.FindByEmail(ctx, user.Email)
	require.NoError(t, err)
	require.Equal(t, "new-bcrypt-hash", loadedAuth.PasswordHash)

	expiresAt := time.Now().UTC().Add(time.Hour)
	require.NoError(t, refreshTokens.Save(ctx, userID, "refresh-hash-old", expiresAt))
	rotatedUserID, err := refreshTokens.Rotate(ctx, "refresh-hash-old", "refresh-hash-new", expiresAt)
	require.NoError(t, err)
	require.Equal(t, userID, rotatedUserID)
	_, err = refreshTokens.Rotate(ctx, "refresh-hash-old", "refresh-hash-again", expiresAt)
	require.ErrorIs(t, err, ports.ErrNotFound)

	require.NoError(t, verificationTokens.Save(ctx, userID, "email_confirmation", "verification-hash", expiresAt))
	consumedUserID, err := verificationTokens.Consume(ctx, "email_confirmation", "verification-hash")
	require.NoError(t, err)
	require.Equal(t, userID, consumedUserID)
	_, err = verificationTokens.Consume(ctx, "email_confirmation", "verification-hash")
	require.ErrorIs(t, err, ports.ErrNotFound)
}
