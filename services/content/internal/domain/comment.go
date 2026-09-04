package domain

import (
	"time"
	"uuid"
)

// Comment описывает комментарий к посту без инфраструктурных зависимостей.
type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	AuthorID  uuid.UUID
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CanBeManagedBy проверяет, принадлежит ли комментарий указанному пользователю.
func (c Comment) CanBeManagedBy(userID uuid.UUID) bool {
	return c.AuthorID == userID
}
