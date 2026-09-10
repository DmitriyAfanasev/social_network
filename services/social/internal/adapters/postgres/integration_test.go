package postgres

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"general-project/social/internal/domain"
	"general-project/social/internal/ports"
)

func TestPostgresSocialRepositories(t *testing.T) {
	// Arrange
	dsn := os.Getenv("SOCIAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SOCIAL_TEST_DATABASE_URL не задан")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	friendships := NewFriendshipRepository(pool)
	blocks := NewBlockRepository(pool)
	first, second := uuid.New(), uuid.New()
	left, right := domain.Pair(first, second)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM social.user_blocks WHERE blocker_id IN ($1, $2) OR blocked_id IN ($1, $2)", first, second)
		_, _ = pool.Exec(context.Background(), "DELETE FROM social.friendships WHERE user_id IN ($1, $2) OR friend_id IN ($1, $2)", first, second)
	})

	// Act
	err = friendships.CreateFriendship(ctx, left, right)

	// Assert
	require.NoError(t, err)
	require.NoError(t, friendships.CreateFriendship(ctx, left, right))
	friend, err := friendships.IsFriend(ctx, first, second)
	require.NoError(t, err)
	require.True(t, friend)
	friends, err := friendships.ListFriends(ctx, first)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{second}, friends)

	// Act
	blocked, err := blocks.IsBlocked(ctx, first, second)

	// Assert
	require.NoError(t, err)
	require.False(t, blocked)
	require.NoError(t, blocks.Block(ctx, first, second))
	blocked, err = blocks.IsBlocked(ctx, second, first)
	require.NoError(t, err)
	require.True(t, blocked)
	removed, err := blocks.Unblock(ctx, first, second)
	require.NoError(t, err)
	require.True(t, removed)
	removed, err = blocks.Unblock(ctx, first, second)
	require.NoError(t, err)
	require.False(t, removed)

	// Act
	removedFriendship, err := friendships.RemoveFriendship(ctx, left, right)

	// Assert
	require.NoError(t, err)
	require.True(t, removedFriendship)
	removedFriendship, err = friendships.RemoveFriendship(ctx, left, right)
	require.NoError(t, err)
	require.False(t, removedFriendship)
	_, err = friendships.GetFriendRequest(ctx, uuid.New())
	require.ErrorIs(t, err, ports.ErrFriendRequestNotFound)
}
