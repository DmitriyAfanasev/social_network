// Package postgres содержит PostgreSQL-адаптеры analytics-сервиса.
package postgres

import (
	"context"

	"general-project/analytics/internal/domain"
)

// SummaryRepository читает агрегаты из журнала analytics-событий.
type SummaryRepository struct {
	pool dbPool
}

// NewSummaryRepository создаёт адаптер analytics summary.
func NewSummaryRepository(pool dbPool) *SummaryRepository {
	return &SummaryRepository{pool: pool}
}

// GetSummary агрегирует события без зависимости от таблиц других схем.
func (r *SummaryRepository) GetSummary(ctx context.Context) (domain.Summary, error) {
	const query = `
		SELECT
			COUNT(*)::bigint,
			COUNT(DISTINCT NULLIF(COALESCE(
				payload->>'user_id', payload->>'author_id', payload->>'uploaded_by',
				payload->>'subscriber_id'
			), ''))::bigint,
			COUNT(*) FILTER (WHERE event_type = 'identity.user.registered')::bigint,
			COUNT(*) FILTER (WHERE event_type = 'content.post.created')::bigint,
			COUNT(*) FILTER (WHERE event_type = 'content.comment.created')::bigint,
			COUNT(*) FILTER (WHERE event_type IN ('content.post.like.created', 'content.comment.like.created'))::bigint,
			COUNT(*) FILTER (WHERE event_type IN ('content.post.like.deleted', 'content.comment.like.deleted'))::bigint,
			COUNT(*) FILTER (WHERE event_type = 'profiles.photo.deleted')::bigint
		FROM analytics.event_records`
	var summary domain.Summary
	err := r.pool.QueryRow(ctx, query).Scan(
		&summary.EventsCount,
		&summary.UniqueUsers,
		&summary.UsersRegistered,
		&summary.PostsCreated,
		&summary.CommentsCreated,
		&summary.LikesAdded,
		&summary.LikesRemoved,
		&summary.PhotosDeleted,
	)
	return summary, err
}
