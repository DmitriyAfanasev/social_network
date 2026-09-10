package application

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/social/internal/ports"
)

type socialOutboxFake struct {
	events    []ports.OutboxEvent
	claimErr  error
	published []uuid.UUID
	failed    []uuid.UUID
}

func (f *socialOutboxFake) Claim(context.Context, int, time.Time) ([]ports.OutboxEvent, error) {
	return f.events, f.claimErr
}

func (f *socialOutboxFake) MarkPublished(_ context.Context, id uuid.UUID, _ time.Time) error {
	f.published = append(f.published, id)
	return nil
}

func (f *socialOutboxFake) MarkFailed(_ context.Context, id uuid.UUID, _ time.Time, _ string) error {
	f.failed = append(f.failed, id)
	return nil
}

type socialEventPublisherFake struct{ err error }

func (f socialEventPublisherFake) Publish(context.Context, ports.OutboxEvent) error { return f.err }
func (socialEventPublisherFake) Close() error                                       { return nil }

func TestSocialOutboxPublisherMarksPublishedEvents(t *testing.T) {
	t.Parallel()

	event := ports.OutboxEvent{ID: uuid.New()}
	outbox := &socialOutboxFake{events: []ports.OutboxEvent{event}}
	publisher := NewOutboxPublisher(outbox, socialEventPublisherFake{})

	err := publisher.RunOnce(context.Background())

	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{event.ID}, outbox.published)
	require.Empty(t, outbox.failed)
}

func TestSocialOutboxPublisherMarksFailedEvents(t *testing.T) {
	t.Parallel()

	event := ports.OutboxEvent{ID: uuid.New()}
	outbox := &socialOutboxFake{events: []ports.OutboxEvent{event}}
	publisher := NewOutboxPublisher(outbox, socialEventPublisherFake{err: errors.New("broker unavailable")})

	err := publisher.RunOnce(context.Background())

	require.Error(t, err)
	require.Equal(t, []uuid.UUID{event.ID}, outbox.failed)
	require.Empty(t, outbox.published)
}

func TestSocialOutboxPublisherReturnsClaimError(t *testing.T) {
	t.Parallel()

	claimErr := errors.New("database unavailable")
	publisher := NewOutboxPublisher(&socialOutboxFake{claimErr: claimErr}, socialEventPublisherFake{})

	require.ErrorIs(t, publisher.RunOnce(context.Background()), claimErr)
}
