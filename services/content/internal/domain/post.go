// Package domain содержит сущности и правила content-сервиса.
package domain

import (
	"time"

	"uuid"
)

// Post описывает текстовый пост без инфраструктурных зависимостей.
type Post struct {
	ID       uuid.UUID
	AuthorID uuid.UUID
	Body     string
	MediaIDs []uuid.UUID
	// CommentsCount — число комментариев, связанных с постом.
	CommentsCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CanBeManagedBy проверяет, принадлежит ли пост указанному пользователю.
func (p Post) CanBeManagedBy(userID uuid.UUID) bool {
	return p.AuthorID == userID
}
