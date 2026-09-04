package domain

import (
	"time"
	"uuid"
)

// MusicTrack описывает музыкальный трек, связанный с audio media-объектом.
type MusicTrack struct {
	ID        uuid.UUID
	MediaID   uuid.UUID
	UserID    uuid.UUID
	Title     string
	Artist    string
	Duration  *float64
	CreatedAt time.Time
}

// CanBeManagedBy проверяет, принадлежит ли трек указанному пользователю.
func (t MusicTrack) CanBeManagedBy(userID uuid.UUID) bool {
	return t.UserID == userID
}
