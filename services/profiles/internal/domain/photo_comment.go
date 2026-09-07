package domain

import (
	"time"
	"uuid"
)

// PhotoComment описывает комментарий к фотографии.
type PhotoComment struct {
	ID        uuid.UUID
	PhotoID   uuid.UUID
	UserID    uuid.UUID
	Body      string
	CreatedAt time.Time
}
