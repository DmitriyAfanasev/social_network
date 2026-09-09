package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

type fakePostRepository struct {
	posts        map[uuid.UUID]domain.Post
	createCalls  int
	outboxEvents []ports.OutboxEvent
}

func (f *fakePostRepository) Create(_ context.Context, post domain.Post) (domain.Post, error) {
	post.CreatedAt = time.Now().UTC()
	post.UpdatedAt = post.CreatedAt
	f.posts[post.ID] = post
	f.createCalls++
	return post, nil
}

func (f *fakePostRepository) CreateWithOutbox(ctx context.Context, post domain.Post, event ports.OutboxEvent) (domain.Post, error) {
	f.outboxEvents = append(f.outboxEvents, event)
	return f.Create(ctx, post)
}

type fakeContentOutbox struct{}

func (f *fakeContentOutbox) Claim(_ context.Context, _ int, _ time.Time) ([]ports.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeContentOutbox) MarkPublished(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (f *fakeContentOutbox) MarkFailed(_ context.Context, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

func (f *fakePostRepository) FindByID(_ context.Context, postID uuid.UUID) (domain.Post, error) {
	post, ok := f.posts[postID]
	if !ok {
		return domain.Post{}, ports.ErrNotFound
	}
	return post, nil
}

func (f *fakePostRepository) ListRecent(_ context.Context, limit int) ([]domain.Post, error) {
	result := make([]domain.Post, 0, limit)
	for _, post := range f.posts {
		if len(result) == limit {
			break
		}
		result = append(result, post)
	}
	return result, nil
}

func (f *fakePostRepository) Update(ctx context.Context, postID uuid.UUID, body string, mediaIDs []uuid.UUID) (domain.Post, error) {
	post, err := f.FindByID(ctx, postID)
	if err != nil {
		return domain.Post{}, err
	}
	post.Body = body
	post.MediaIDs = mediaIDs
	post.UpdatedAt = time.Now().UTC()
	f.posts[postID] = post
	return post, nil
}

func (f *fakePostRepository) Delete(_ context.Context, postID uuid.UUID) error {
	if _, ok := f.posts[postID]; !ok {
		return ports.ErrNotFound
	}
	delete(f.posts, postID)
	return nil
}

type fakePostCache struct {
	values map[string][]byte
}

type fakeMediaReferenceChecker struct {
	validated []uuid.UUID
}

func (f *fakeMediaReferenceChecker) ValidateOwned(_ context.Context, _ uuid.UUID, mediaIDs []uuid.UUID) error {
	f.validated = mediaIDs
	return nil
}

type fakeCommentRepository struct {
	comments map[uuid.UUID]domain.Comment
}

func (f *fakeCommentRepository) Create(_ context.Context, comment domain.Comment) (domain.Comment, error) {
	comment.CreatedAt = time.Now().UTC()
	comment.UpdatedAt = comment.CreatedAt
	f.comments[comment.ID] = comment
	return comment, nil
}

func (f *fakeCommentRepository) FindByID(_ context.Context, commentID uuid.UUID) (domain.Comment, error) {
	comment, ok := f.comments[commentID]
	if !ok {
		return domain.Comment{}, ports.ErrNotFound
	}
	return comment, nil
}

func (f *fakeCommentRepository) ListByPost(_ context.Context, postID uuid.UUID, limit int) ([]domain.Comment, error) {
	result := make([]domain.Comment, 0, limit)
	for _, comment := range f.comments {
		if comment.PostID == postID {
			result = append(result, comment)
		}
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func (f *fakeCommentRepository) Delete(_ context.Context, commentID uuid.UUID) error {
	if _, ok := f.comments[commentID]; !ok {
		return ports.ErrNotFound
	}
	delete(f.comments, commentID)
	return nil
}

type fakeLikeRepository struct {
	liked bool
}

func (f *fakeLikeRepository) Add(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	f.liked = true
	return nil
}

func (f *fakeLikeRepository) Remove(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	f.liked = false
	return true, nil
}

func (f *fakeLikeRepository) Count(context.Context, uuid.UUID) (int, error) {
	if f.liked {
		return 1, nil
	}
	return 0, nil
}

func (f *fakeLikeRepository) Has(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.liked, nil
}

func (f *fakeLikeRepository) ListUserIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	if !f.liked {
		return nil, nil
	}
	return []uuid.UUID{uuid.New()}, nil
}

func (f *fakePostCache) Get(_ context.Context, key string) ([]byte, error) {
	return f.values[key], nil
}

func (f *fakePostCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.values[key] = value
	return nil
}

func (f *fakePostCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func TestContentServiceCreatesNormalizedPost(t *testing.T) {
	t.Parallel()

	repository := &fakePostRepository{posts: map[uuid.UUID]domain.Post{}}
	service := NewContentService(repository, nil, nil)

	post, err := service.CreatePost(context.Background(), uuid.New(), CreatePostInput{Body: "  привет  "})

	require.NoError(t, err)
	require.Equal(t, "привет", post.Body)
	require.NotEqual(t, uuid.Nil(), post.ID)
	require.Equal(t, 1, repository.createCalls)
}

func TestContentServiceWritesPostAndOutboxEventTogether(t *testing.T) {
	t.Parallel()

	repository := &fakePostRepository{posts: map[uuid.UUID]domain.Post{}}
	service := NewContentService(repository, nil, nil, &fakeContentOutbox{})

	post, err := service.CreatePost(context.Background(), uuid.New(), CreatePostInput{Body: "пост с событием"})

	require.NoError(t, err)
	require.Len(t, repository.outboxEvents, 1)
	require.Equal(t, "content.post.created", repository.outboxEvents[0].EventType)
	require.Equal(t, post.ID, *repository.outboxEvents[0].AggregateID)
	require.NotEmpty(t, repository.outboxEvents[0].Payload)
}

func TestContentServiceAllowsMediaOnlyPost(t *testing.T) {
	t.Parallel()

	mediaID := uuid.New()
	repository := &fakePostRepository{posts: map[uuid.UUID]domain.Post{}}
	checker := &fakeMediaReferenceChecker{}
	service := NewContentService(repository, nil, checker)

	post, err := service.CreatePost(context.Background(), uuid.New(), CreatePostInput{MediaIDs: []uuid.UUID{mediaID}})

	require.NoError(t, err)
	require.Empty(t, post.Body)
	require.Equal(t, []uuid.UUID{mediaID}, post.MediaIDs)
	require.Equal(t, []uuid.UUID{mediaID}, checker.validated)
}

func TestContentServiceRejectsDuplicateMediaReference(t *testing.T) {
	t.Parallel()

	mediaID := uuid.New()
	service := NewContentService(&fakePostRepository{posts: map[uuid.UUID]domain.Post{}}, nil, nil)

	_, err := service.CreatePost(context.Background(), uuid.New(), CreatePostInput{Body: "пост", MediaIDs: []uuid.UUID{mediaID, mediaID}})

	require.ErrorIs(t, err, ErrValidation)
}

func TestContentServiceRejectsInvalidPost(t *testing.T) {
	t.Parallel()

	service := NewContentService(&fakePostRepository{posts: map[uuid.UUID]domain.Post{}}, nil, nil)

	_, err := service.CreatePost(context.Background(), uuid.New(), CreatePostInput{Body: "   "})

	require.ErrorIs(t, err, ErrValidation)
}

func TestContentServiceGetPostUsesCache(t *testing.T) {
	t.Parallel()

	postID := uuid.New()
	authorID := uuid.New()
	cached := PostDTO{ID: postID, AuthorID: authorID, Body: "из cache"}
	encoded, err := json.Marshal(cached)
	require.NoError(t, err)

	repository := &fakePostRepository{posts: map[uuid.UUID]domain.Post{}}
	postCache := &fakePostCache{values: map[string][]byte{postCacheKey(postID): encoded}}
	service := NewContentService(repository, postCache, nil)

	result, err := service.GetPost(context.Background(), postID)

	require.NoError(t, err)
	require.Equal(t, cached.Body, result.Body)
}

func TestContentServiceRejectsUpdateByAnotherUser(t *testing.T) {
	t.Parallel()

	postID := uuid.New()
	repository := &fakePostRepository{posts: map[uuid.UUID]domain.Post{
		postID: {ID: postID, AuthorID: uuid.New(), Body: "текст"},
	}}
	service := NewContentService(repository, nil, nil)

	_, err := service.UpdatePost(context.Background(), uuid.New(), postID, UpdatePostInput{Body: "новый текст"})

	require.ErrorIs(t, err, ErrForbidden)
}

func TestContentServiceInvalidatesPostCacheOnUpdate(t *testing.T) {
	t.Parallel()

	postID := uuid.New()
	authorID := uuid.New()
	repository := &fakePostRepository{posts: map[uuid.UUID]domain.Post{
		postID: {ID: postID, AuthorID: authorID, Body: "старый текст"},
	}}
	postCache := &fakePostCache{values: map[string][]byte{
		postCacheKey(postID): []byte(`stale`),
		feedCacheKey(20):     []byte(`stale`),
	}}
	service := NewContentService(repository, postCache, nil)

	_, err := service.UpdatePost(context.Background(), authorID, postID, UpdatePostInput{Body: "новый текст"})

	require.NoError(t, err)
	require.Nil(t, postCache.values[postCacheKey(postID)])
	require.Nil(t, postCache.values[feedCacheKey(20)])
}

func TestCommentServiceCreatesCommentForExistingPost(t *testing.T) {
	t.Parallel()

	postID := uuid.New()
	posts := &fakePostRepository{posts: map[uuid.UUID]domain.Post{
		postID: {ID: postID, AuthorID: uuid.New(), Body: "пост"},
	}}
	postCache := &fakePostCache{values: map[string][]byte{
		postCacheKey(postID): []byte("stale"),
		feedCacheKey(20):     []byte("stale"),
	}}
	service := NewCommentService(posts, &fakeCommentRepository{comments: map[uuid.UUID]domain.Comment{}}, postCache)

	comment, err := service.CreateComment(context.Background(), uuid.New(), postID, CreateCommentInput{Body: "  комментарий  "})

	require.NoError(t, err)
	require.Equal(t, "комментарий", comment.Body)
	require.Equal(t, postID, comment.PostID)
	require.Nil(t, postCache.values[postCacheKey(postID)])
	require.Nil(t, postCache.values[feedCacheKey(20)])
}

func TestCommentServiceRejectsMissingPost(t *testing.T) {
	t.Parallel()

	service := NewCommentService(&fakePostRepository{posts: map[uuid.UUID]domain.Post{}}, &fakeCommentRepository{comments: map[uuid.UUID]domain.Comment{}})

	_, err := service.CreateComment(context.Background(), uuid.New(), uuid.New(), CreateCommentInput{Body: "комментарий"})

	require.ErrorIs(t, err, ports.ErrNotFound)
}

func TestCommentServiceRejectsDeleteByAnotherUser(t *testing.T) {
	t.Parallel()

	commentID := uuid.New()
	comments := &fakeCommentRepository{comments: map[uuid.UUID]domain.Comment{
		commentID: {ID: commentID, AuthorID: uuid.New(), Body: "комментарий"},
	}}
	service := NewCommentService(&fakePostRepository{posts: map[uuid.UUID]domain.Post{}}, comments)

	err := service.DeleteComment(context.Background(), uuid.New(), commentID)

	require.ErrorIs(t, err, ErrForbidden)
}

func TestLikeServiceIsIdempotentAtApplicationBoundary(t *testing.T) {
	t.Parallel()

	postID := uuid.New()
	posts := &fakePostRepository{posts: map[uuid.UUID]domain.Post{
		postID: {ID: postID, AuthorID: uuid.New(), Body: "пост"},
	}}
	likes := &fakeLikeRepository{}
	service := NewLikeService(posts, likes)

	first, err := service.LikePost(context.Background(), uuid.New(), postID)
	require.NoError(t, err)
	require.True(t, first.Liked)
	require.True(t, likes.liked)

	second, err := service.UnlikePost(context.Background(), uuid.New(), postID)
	require.NoError(t, err)
	require.False(t, second.Liked)
	require.False(t, likes.liked)
}
