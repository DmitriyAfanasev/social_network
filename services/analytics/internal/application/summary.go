package application

import (
	"context"

	"general-project/analytics/internal/domain"
	"general-project/analytics/internal/ports"
)

// SummaryService возвращает агрегированный analytics summary.
type SummaryService struct {
	repository ports.SummaryRepository
}

// NewSummaryService создаёт сервис чтения analytics summary.
func NewSummaryService(repository ports.SummaryRepository) *SummaryService {
	return &SummaryService{repository: repository}
}

// GetSummary возвращает текущие показатели из analytics-схемы.
func (s *SummaryService) GetSummary(ctx context.Context) (domain.Summary, error) {
	return s.repository.GetSummary(ctx)
}
