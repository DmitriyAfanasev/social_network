// Package application содержит сценарии admin-сервиса.
package application

import (
	"context"
	"errors"
	"uuid"

	"general-project/admin/internal/domain"
	"general-project/admin/internal/ports"
)

var (
	// ErrValidation означает, что параметры audit-записи некорректны.
	ErrValidation = errors.New("admin audit validation failed")
)

// AuditService реализует сценарии записи и чтения moderation audit.
type AuditService struct {
	repository ports.AuditRepository
}

// NewAuditService создаёт application-сервис moderation audit.
func NewAuditService(repository ports.AuditRepository) *AuditService {
	return &AuditService{repository: repository}
}

// Record сохраняет действие модерации с заданным event ID для идемпотентности.
func (s *AuditService) Record(ctx context.Context, log domain.AuditLog) error {
	if log.EventID == uuid.Nil() || log.Action == "" || log.TargetType == "" {
		return ErrValidation
	}
	return s.repository.Record(ctx, log)
}

// List возвращает последние записи аудита с ограничением размера выборки.
func (s *AuditService) List(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrValidation
	}
	return s.repository.List(ctx, limit)
}
