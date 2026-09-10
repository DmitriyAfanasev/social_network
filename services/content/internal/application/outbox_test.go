package application

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/content/internal/ports"
)

type contentOutboxFake struct {
	events      []ports.OutboxEvent
	claimErr    error
	published   []uuid.UUID
	failed      []uuid.UUID
	failedError error
}

func (f *contentOutboxFake) Claim(context.Context, int, time.Time) ([]ports.OutboxEvent, error) {
	return f.events, f.claimErr
}

func (f *contentOutboxFake) MarkPublished(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	f.published = append(f.published, eventID)
	return nil
}

func (f *contentOutboxFake) MarkFailed(_ context.Context, eventID uuid.UUID, _ time.Time, reason string) error {
	f.failed = append(f.failed, eventID)
	f.failedError = errors.New(reason)
	return nil
}

type contentEventPublisherFake struct {
	events []ports.OutboxEvent
	err    error
}

func (f *contentEventPublisherFake) Publish(_ context.Context, event ports.OutboxEvent) error {
	f.events = append(f.events, event)
	return f.err
}

func (*contentEventPublisherFake) Close() error { return nil }

func TestContentOutboxPublisherMarksPublishedEvents(t *testing.T) {
	t.Parallel()

	event := ports.OutboxEvent{ID: uuid.New(), EventType: "content.post.created"}
	outbox := &contentOutboxFake{events: []ports.OutboxEvent{event}}
	publisher := NewOutboxPublisher(outbox, &contentEventPublisherFake{})

	err := publisher.RunOnce(context.Background())

	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{event.ID}, outbox.published)
	require.Empty(t, outbox.failed)
}

func TestContentOutboxPublisherMarksFailedAndContinues(t *testing.T) {
	t.Parallel()

	first, second := ports.OutboxEvent{ID: uuid.New()}, ports.OutboxEvent{ID: uuid.New()}
	outbox := &contentOutboxFake{events: []ports.OutboxEvent{first, second}}
	broker := &contentEventPublisherFake{err: errors.New("broker unavailable")}
	publisher := NewOutboxPublisher(outbox, broker)

	err := publisher.RunOnce(context.Background())

	require.Error(t, err)
	require.ElementsMatch(t, []uuid.UUID{first.ID, second.ID}, outbox.failed)
	require.Empty(t, outbox.published)
}

func TestContentOutboxPublisherReturnsClaimError(t *testing.T) {
	t.Parallel()

	claimErr := errors.New("database unavailable")
	publisher := NewOutboxPublisher(&contentOutboxFake{claimErr: claimErr}, &contentEventPublisherFake{})

	err := publisher.RunOnce(context.Background())

	require.ErrorIs(t, err, claimErr)
}
