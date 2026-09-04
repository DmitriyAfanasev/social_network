package ports

import (
	"context"
	"uuid"

	"general-project/content/internal/domain"
)

// CommentRepository предоставляет application-слою доступ к комментариям.
type CommentRepository interface {
	Create(ctx context.Context, comment domain.Comment) (domain.Comment, error)
	FindByID(ctx context.Context, commentID uuid.UUID) (domain.Comment, error)
	ListByPost(ctx context.Context, postID uuid.UUID, limit int) ([]domain.Comment, error)
	Delete(ctx context.Context, commentID uuid.UUID) error
}

// LikeRepository предоставляет application-слою операции с лайками постов.
type LikeRepository interface {
	Add(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error
	Remove(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error)
}
