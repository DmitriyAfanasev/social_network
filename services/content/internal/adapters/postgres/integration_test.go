package postgres

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

func TestPostgresContentRepositories(t *testing.T) {
	// Arrange
	dsn := os.Getenv("CONTENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CONTENT_TEST_DATABASE_URL не задан")
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

	posts := NewPostRepository(pool)
	comments := NewCommentRepository(pool)
	likes := NewLikeRepository(pool)
	author, viewer := uuid.New(), uuid.New()
	postID, mediaID := uuid.New(), uuid.New()

	// Act
	post, err := posts.Create(ctx, domain.Post{ID: postID, AuthorID: author, Body: "integration post", MediaIDs: []uuid.UUID{mediaID}})

	// Assert
	require.NoError(t, err)
	require.Equal(t, postID, post.ID)
	require.Equal(t, []uuid.UUID{mediaID}, post.MediaIDs)

	// Act
	loaded, err := posts.FindByID(ctx, postID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, post.MediaIDs, loaded.MediaIDs)

	// Act
	err = likes.Add(ctx, postID, viewer)

	// Assert
	require.NoError(t, err)
	require.NoError(t, likes.Add(ctx, postID, viewer))
	count, err := likes.Count(ctx, postID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Act
	comment, err := comments.Create(ctx, domain.Comment{ID: uuid.New(), PostID: postID, AuthorID: viewer, Body: "looks good"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, "looks good", comment.Body)
	listed, err := comments.ListByPost(ctx, postID, 10)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	// Act
	removed, err := likes.Remove(ctx, postID, viewer)

	// Assert
	require.NoError(t, err)
	require.True(t, removed)
	removed, err = likes.Remove(ctx, postID, viewer)
	require.NoError(t, err)
	require.False(t, removed)
	require.NoError(t, posts.Delete(ctx, postID))
	_, err = posts.FindByID(ctx, postID)
	require.ErrorIs(t, err, ports.ErrNotFound)
}
