package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"uuid"

	"general-project/content/internal/domain"
	"general-project/content/internal/ports"
)

const maxCommentLength = 2000

// CreateCommentInput содержит входные данные создания комментария.
type CreateCommentInput struct {
	Body string
}

// CommentDTO представляет безопасный результат application-сценария комментария.
type CommentDTO struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	AuthorID  uuid.UUID
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LikeDTO представляет результат изменения лайка поста.
type LikeDTO struct {
	PostID uuid.UUID
	UserID uuid.UUID
	Liked  bool
}

// CommentService реализует сценарии комментариев к текстовым постам.
type CommentService struct {
	posts    ports.PostRepository
	comments ports.CommentRepository
	cache    ports.PostCache
}

// NewCommentService создаёт application-сервис комментариев.
func NewCommentService(posts ports.PostRepository, comments ports.CommentRepository, caches ...ports.PostCache) *CommentService {
	var cache ports.PostCache
	if len(caches) > 0 {
		cache = caches[0]
	}
	return &CommentService{posts: posts, comments: comments, cache: cache}
}

// CreateComment создаёт комментарий текущего пользователя к существующему посту.
func (s *CommentService) CreateComment(ctx context.Context, authorID uuid.UUID, postID uuid.UUID, input CreateCommentInput) (CommentDTO, error) {
	if _, err := s.posts.FindByID(ctx, postID); errors.Is(err, ports.ErrNotFound) {
		return CommentDTO{}, ports.ErrNotFound
	} else if err != nil {
		return CommentDTO{}, err
	}
	body := strings.TrimSpace(input.Body)
	if length := len([]rune(body)); length < 1 || length > maxCommentLength {
		return CommentDTO{}, ErrValidation
	}
	comment, err := s.comments.Create(ctx, domain.Comment{
		ID:       uuid.New(),
		PostID:   postID,
		AuthorID: authorID,
		Body:     body,
	})
	if err != nil {
		return CommentDTO{}, err
	}
	s.invalidatePostCaches(ctx, postID)
	return toCommentDTO(comment), nil
}

// ListComments возвращает комментарии поста в порядке публикации.
func (s *CommentService) ListComments(ctx context.Context, postID uuid.UUID, limit int) ([]CommentDTO, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrValidation
	}
	if _, err := s.posts.FindByID(ctx, postID); errors.Is(err, ports.ErrNotFound) {
		return nil, ports.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	comments, err := s.comments.ListByPost(ctx, postID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]CommentDTO, 0, len(comments))
	for _, comment := range comments {
		result = append(result, toCommentDTO(comment))
	}
	return result, nil
}

// DeleteComment удаляет комментарий только его автором.
func (s *CommentService) DeleteComment(ctx context.Context, actorID uuid.UUID, commentID uuid.UUID) error {
	comment, err := s.comments.FindByID(ctx, commentID)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !comment.CanBeManagedBy(actorID) {
		return ErrForbidden
	}
	if err := s.comments.Delete(ctx, commentID); err != nil {
		return err
	}
	s.invalidatePostCaches(ctx, comment.PostID)
	return nil
}

func (s *CommentService) invalidatePostCaches(ctx context.Context, postID uuid.UUID) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, postCacheKey(postID), feedCacheKey(10), feedCacheKey(20), feedCacheKey(50))
}

// LikePost добавляет лайк текущего пользователя к существующему посту.
func (s *LikeService) LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (LikeDTO, error) {
	if _, err := s.posts.FindByID(ctx, postID); errors.Is(err, ports.ErrNotFound) {
		return LikeDTO{}, ports.ErrNotFound
	} else if err != nil {
		return LikeDTO{}, err
	}
	if err := s.likes.Add(ctx, postID, userID); err != nil {
		return LikeDTO{}, err
	}
	return LikeDTO{PostID: postID, UserID: userID, Liked: true}, nil
}

// UnlikePost удаляет лайк текущего пользователя у поста.
func (s *LikeService) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (LikeDTO, error) {
	if _, err := s.posts.FindByID(ctx, postID); errors.Is(err, ports.ErrNotFound) {
		return LikeDTO{}, ports.ErrNotFound
	} else if err != nil {
		return LikeDTO{}, err
	}
	if _, err := s.likes.Remove(ctx, postID, userID); err != nil {
		return LikeDTO{}, err
	}
	return LikeDTO{PostID: postID, UserID: userID, Liked: false}, nil
}

// LikeService реализует сценарии лайков текстовых постов.
type LikeService struct {
	posts ports.PostRepository
	likes ports.LikeRepository
}

// NewLikeService создаёт application-сервис лайков.
func NewLikeService(posts ports.PostRepository, likes ports.LikeRepository) *LikeService {
	return &LikeService{posts: posts, likes: likes}
}

func toCommentDTO(comment domain.Comment) CommentDTO {
	return CommentDTO{
		ID:        comment.ID,
		PostID:    comment.PostID,
		AuthorID:  comment.AuthorID,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}
