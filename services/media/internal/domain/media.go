package domain

import (
	"time"
	"uuid"
)

// Media описывает метаданные объекта в S3-совместимом хранилище.
type Media struct {
	ID               uuid.UUID
	ObjectKey        string
	Bucket           string
	OriginalFilename string
	ContentType      string
	MediaType        string
	Size             int64
	Checksum         string
	Width            *int
	Height           *int
	Duration         *float64
	UploadedBy       uuid.UUID
	CreatedAt        time.Time
	DeletedAt        *time.Time
}

// CanBeManagedBy проверяет, может ли пользователь управлять метаданными файла.
func (m Media) CanBeManagedBy(userID uuid.UUID) bool {
	return m.UploadedBy == userID
}
