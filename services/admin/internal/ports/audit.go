// Package ports содержит интерфейсы, которыми владеет application-слой.
package ports

import (
	"context"
	"uuid"

	"general-project/admin/internal/domain"
)

// AuditRepository хранит и читает записи модерационного аудита.
type AuditRepository interface {
	Record(ctx context.Context, log domain.AuditLog) error
	List(ctx context.Context, limit int) ([]domain.AuditLog, error)
}

// EventIDGenerator генерирует идентификаторы аудита при ручной записи.
type EventIDGenerator interface {
	New() uuid.UUID
}
