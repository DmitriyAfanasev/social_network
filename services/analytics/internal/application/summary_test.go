package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"general-project/analytics/internal/domain"
)

type fakeSummaryRepository struct {
	summary domain.Summary
}

func (f fakeSummaryRepository) GetSummary(context.Context) (domain.Summary, error) {
	return f.summary, nil
}

func TestSummaryServiceReturnsRepositorySummary(t *testing.T) {
	wanted := domain.Summary{EventsCount: 4, UniqueUsers: 3, PostsCreated: 2}
	service := NewSummaryService(fakeSummaryRepository{summary: wanted})

	result, err := service.GetSummary(context.Background())

	require.NoError(t, err)
	require.Equal(t, wanted, result)
}
