package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

func TestContentServiceValidatesMediaCountAndOwnershipDependency(t *testing.T) {
	t.Parallel()

	mediaIDs := make([]uuid.UUID, 10)
	for index := range mediaIDs {
		mediaIDs[index] = uuid.New()
	}
	posts := &fakePostRepository{posts: map[uuid.UUID]domain.Post{}}
	service := NewContentService(posts, nil, &fakeMediaReferenceChecker{})

	post, err := service.CreatePost(context.Background(), uuid.New(), CreatePostInput{MediaIDs: mediaIDs})
	require.NoError(t, err)
	require.Len(t, post.MediaIDs, 10)

	tooMany := append(append([]uuid.UUID{}, mediaIDs...), uuid.New())
	_, err = service.CreatePost(context.Background(), uuid.New(), CreatePostInput{MediaIDs: tooMany})
	require.ErrorIs(t, err, ErrValidation)

	withoutChecker := NewContentService(posts, nil, nil)
	_, err = withoutChecker.CreatePost(context.Background(), uuid.New(), CreatePostInput{Body: "post", MediaIDs: []uuid.UUID{uuid.New()}})
	require.ErrorIs(t, err, ErrValidation)
}

func TestCommentServiceValidatesBodyBoundariesAndDeletesOwnedComment(t *testing.T) {
	t.Parallel()

	postID, authorID := uuid.New(), uuid.New()
	commentID := uuid.New()
	posts := &fakePostRepository{posts: map[uuid.UUID]domain.Post{postID: {ID: postID, Body: "post"}}}
	comments := &fakeCommentRepository{comments: map[uuid.UUID]domain.Comment{
		commentID: {ID: commentID, PostID: postID, AuthorID: authorID, Body: "comment"},
	}}
	service := NewCommentService(posts, comments)

	_, err := service.CreateComment(context.Background(), authorID, postID, CreateCommentInput{Body: string(make([]rune, 2001))})
	require.ErrorIs(t, err, ErrValidation)

	comment, err := service.CreateComment(context.Background(), authorID, postID, CreateCommentInput{Body: string(make([]rune, 2000))})
	require.NoError(t, err)
	require.Len(t, []rune(comment.Body), 2000)

	err = service.DeleteComment(context.Background(), authorID, commentID)
	require.NoError(t, err)
	_, err = comments.FindByID(context.Background(), commentID)
	require.ErrorIs(t, err, ports.ErrNotFound)
}

func TestLikeServiceRejectsMissingPost(t *testing.T) {
	t.Parallel()

	service := NewLikeService(&fakePostRepository{posts: map[uuid.UUID]domain.Post{}}, &fakeLikeRepository{})

	_, err := service.LikePost(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, ports.ErrNotFound)

	_, err = service.UnlikePost(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, ports.ErrNotFound)
}
