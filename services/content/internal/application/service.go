package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"uuid"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
	"general-project/libs/platform/correlation"
)

var (
	// ErrValidation означает, что тело поста не соответствует правилам.
	ErrValidation = errors.New("content validation failed")
	// ErrForbidden означает, что пользователь не может изменить пост.
	ErrForbidden = errors.New("post management forbidden")
)

const (
	postCacheTTL = 30 * time.Second
	feedCacheTTL = 10 * time.Second
)

// CreatePostInput содержит входные данные создания поста.
type CreatePostInput struct {
	Body     string
	MediaIDs []uuid.UUID
}

// UpdatePostInput содержит входные данные изменения поста.
type UpdatePostInput struct {
	Body     string
	MediaIDs []uuid.UUID
}

// PostDTO представляет безопасный результат application-сценария поста.
type PostDTO struct {
	ID            uuid.UUID
	AuthorID      uuid.UUID
	Body          string
	MediaIDs      []uuid.UUID
	CommentsCount int
	LikesCount    int
	LikedByViewer bool
	LikedUserIDs  []uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContentService реализует сценарии текстовых постов.
type ContentService struct {
	posts  ports.PostRepository
	cache  ports.PostCache
	media  ports.MediaReferenceChecker
	outbox ports.OutboxRepository
	likes  ports.LikeStatsReader
}

// NewContentServiceWithLikes создаёт content-сервис с чтением статистики лайков.
func NewContentServiceWithLikes(posts ports.PostRepository, cache ports.PostCache, media ports.MediaReferenceChecker, outbox ports.OutboxRepository, likes ports.LikeStatsReader) *ContentService {
	service := NewContentService(posts, cache, media, outbox)
	service.likes = likes
	return service
}

// NewContentService создаёт application-сервис постов.
func NewContentService(posts ports.PostRepository, cache ports.PostCache, media ports.MediaReferenceChecker, outboxes ...ports.OutboxRepository) *ContentService {
	var outbox ports.OutboxRepository
	if len(outboxes) > 0 {
		outbox = outboxes[0]
	}
	return &ContentService{posts: posts, cache: cache, media: media, outbox: outbox}
}

// CreatePost создаёт текстовый пост от имени пользователя.
func (s *ContentService) CreatePost(ctx context.Context, authorID uuid.UUID, input CreatePostInput) (PostDTO, error) {
	body := normalizeBody(input.Body)
	if !validPost(body, input.MediaIDs) {
		return PostDTO{}, ErrValidation
	}
	if err := s.validateMedia(ctx, authorID, input.MediaIDs); err != nil {
		return PostDTO{}, err
	}
	post := domain.Post{ID: uuid.New(), AuthorID: authorID, Body: body, MediaIDs: uniqueMediaIDs(input.MediaIDs)}
	eventPayload, marshalErr := json.Marshal(struct {
		PostID   uuid.UUID   `json:"post_id"`
		AuthorID uuid.UUID   `json:"author_id"`
		Body     string      `json:"body"`
		MediaIDs []uuid.UUID `json:"media_ids"`
	}{PostID: post.ID, AuthorID: post.AuthorID, Body: post.Body, MediaIDs: post.MediaIDs})
	if marshalErr != nil {
		return PostDTO{}, marshalErr
	}
	var created domain.Post
	var err error
	if s.outbox != nil {
		now := time.Now().UTC()
		created, err = s.posts.CreateWithOutbox(ctx, post, ports.OutboxEvent{ID: uuid.New(), EventType: "content.post.created", AggregateID: &post.ID, Payload: eventPayload, CorrelationID: correlation.ID(ctx), AvailableAt: now, CreatedAt: now})
	} else {
		created, err = s.posts.Create(ctx, post)
	}
	if err != nil {
		return PostDTO{}, err
	}
	s.invalidateFeed(ctx)
	return toPostDTO(created), nil
}

// GetPost возвращает пост по UUID и использует cache read-модели.
func (s *ContentService) GetPost(ctx context.Context, postID uuid.UUID, viewerIDs ...uuid.UUID) (PostDTO, error) {
	cacheKey := postCacheKey(postID)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached PostDTO
			if json.Unmarshal(payload, &cached) == nil {
				if err := s.enrichLikeStats(ctx, &cached, firstViewer(viewerIDs)); err != nil {
					return PostDTO{}, err
				}
				return cached, nil
			}
		}
	}
	post, err := s.posts.FindByID(ctx, postID)
	if err != nil {
		return PostDTO{}, err
	}
	dto := toPostDTO(post)
	s.cachePost(ctx, cacheKey, dto)
	if err := s.enrichLikeStats(ctx, &dto, firstViewer(viewerIDs)); err != nil {
		return PostDTO{}, err
	}
	return dto, nil
}

// ListRecent возвращает последние посты с ограничением размера страницы.
func (s *ContentService) ListRecent(ctx context.Context, limit int, viewerIDs ...uuid.UUID) ([]PostDTO, error) {
	if limit < 1 || limit > 50 {
		return nil, ErrValidation
	}
	cacheKey := feedCacheKey(limit)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached []PostDTO
			if json.Unmarshal(payload, &cached) == nil {
				if err := s.enrichLikeStatsForPosts(ctx, cached, firstViewer(viewerIDs)); err != nil {
					return nil, err
				}
				return cached, nil
			}
		}
	}
	posts, err := s.posts.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := make([]PostDTO, 0, len(posts))
	for _, post := range posts {
		result = append(result, toPostDTO(post))
	}
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(result); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, feedCacheTTL) //nolint:errcheck // cache is best-effort.
		}
	}
	if err := s.enrichLikeStatsForPosts(ctx, result, firstViewer(viewerIDs)); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdatePost изменяет пост только его автором.
func (s *ContentService) UpdatePost(ctx context.Context, actorID uuid.UUID, postID uuid.UUID, input UpdatePostInput) (PostDTO, error) {
	post, err := s.posts.FindByID(ctx, postID)
	if errors.Is(err, ports.ErrNotFound) {
		return PostDTO{}, ports.ErrNotFound
	}
	if err != nil {
		return PostDTO{}, err
	}
	if !post.CanBeManagedBy(actorID) {
		return PostDTO{}, ErrForbidden
	}
	body := normalizeBody(input.Body)
	if !validPost(body, input.MediaIDs) {
		return PostDTO{}, ErrValidation
	}
	if err := s.validateMedia(ctx, actorID, input.MediaIDs); err != nil {
		return PostDTO{}, err
	}
	updated, err := s.posts.Update(ctx, postID, body, uniqueMediaIDs(input.MediaIDs))
	if err != nil {
		return PostDTO{}, err
	}
	s.invalidatePost(ctx, postID)
	s.invalidateFeed(ctx)
	return toPostDTO(updated), nil
}

// DeletePost удаляет пост только его автором.
func (s *ContentService) DeletePost(ctx context.Context, actorID uuid.UUID, postID uuid.UUID) error {
	post, err := s.posts.FindByID(ctx, postID)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !post.CanBeManagedBy(actorID) {
		return ErrForbidden
	}
	if err := s.posts.Delete(ctx, postID); err != nil {
		return err
	}
	s.invalidatePost(ctx, postID)
	s.invalidateFeed(ctx)
	return nil
}

func normalizeBody(body string) string {
	return strings.TrimSpace(body)
}

func validBody(body string) bool {
	length := len([]rune(body))
	return length >= 1 && length <= 5000
}

func validPost(body string, mediaIDs []uuid.UUID) bool {
	if len(mediaIDs) > 10 {
		return false
	}
	if len(mediaIDs) == 0 {
		return validBody(body)
	}
	return len([]rune(body)) <= 5000 && len(uniqueMediaIDs(mediaIDs)) == len(mediaIDs)
}

func uniqueMediaIDs(mediaIDs []uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(mediaIDs))
	seen := make(map[uuid.UUID]struct{}, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		if _, ok := seen[mediaID]; ok {
			continue
		}
		seen[mediaID] = struct{}{}
		result = append(result, mediaID)
	}
	return result
}

func (s *ContentService) validateMedia(ctx context.Context, userID uuid.UUID, mediaIDs []uuid.UUID) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	if s.media == nil {
		return ErrValidation
	}
	return s.media.ValidateOwned(ctx, userID, mediaIDs)
}

func toPostDTO(post domain.Post) PostDTO {
	return PostDTO{ID: post.ID, AuthorID: post.AuthorID, Body: post.Body, MediaIDs: append([]uuid.UUID(nil), post.MediaIDs...), CommentsCount: post.CommentsCount, CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt}
}

func firstViewer(viewerIDs []uuid.UUID) uuid.UUID {
	if len(viewerIDs) > 0 {
		return viewerIDs[0]
	}
	return uuid.Nil()
}

func (s *ContentService) enrichLikeStatsForPosts(ctx context.Context, posts []PostDTO, viewerID uuid.UUID) error {
	for index := range posts {
		if err := s.enrichLikeStats(ctx, &posts[index], viewerID); err != nil {
			return err
		}
	}
	return nil
}

func (s *ContentService) enrichLikeStats(ctx context.Context, post *PostDTO, viewerID uuid.UUID) error {
	if s.likes == nil {
		return nil
	}
	count, err := s.likes.Count(ctx, post.ID)
	if err != nil {
		return err
	}
	post.LikesCount = count
	post.LikedUserIDs, err = s.likes.ListUserIDs(ctx, post.ID)
	if err != nil {
		return err
	}
	if viewerID != uuid.Nil() {
		post.LikedByViewer, err = s.likes.Has(ctx, post.ID, viewerID)
		if err != nil {
			return err
		}
	}
	return nil
}

func postCacheKey(postID uuid.UUID) string {
	return fmt.Sprintf("content:v2:post:%s", postID)
}

func feedCacheKey(limit int) string {
	return fmt.Sprintf("content:v2:feed:%d", limit)
}

func (s *ContentService) cachePost(ctx context.Context, key string, dto PostDTO) {
	if s.cache == nil {
		return
	}
	if payload, err := json.Marshal(dto); err == nil {
		_ = s.cache.Set(ctx, key, payload, postCacheTTL) //nolint:errcheck // cache is best-effort.
	}
}

func (s *ContentService) invalidatePost(ctx context.Context, postID uuid.UUID) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, postCacheKey(postID)) //nolint:errcheck // cache is best-effort.
	}
}

func (s *ContentService) invalidateFeed(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, feedCacheKey(10), feedCacheKey(20), feedCacheKey(50)) //nolint:errcheck // cache is best-effort.
	}
}
