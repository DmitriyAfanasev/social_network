package application

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/identity/internal/ports"
)

type identityOutboxFake struct {
	events    []ports.OutboxEvent
	claimErr  error
	published []uuid.UUID
	failed    []uuid.UUID
}

func (f *identityOutboxFake) Claim(context.Context, int, time.Time) ([]ports.OutboxEvent, error) {
	return f.events, f.claimErr
}

func (f *identityOutboxFake) MarkPublished(_ context.Context, id uuid.UUID, _ time.Time) error {
	f.published = append(f.published, id)
	return nil
}

func (f *identityOutboxFake) MarkFailed(_ context.Context, id uuid.UUID, _ time.Time, _ string) error {
	f.failed = append(f.failed, id)
	return nil
}

type identityEventPublisherFake struct{ err error }

func (f identityEventPublisherFake) Publish(context.Context, ports.OutboxEvent) error { return f.err }
func (identityEventPublisherFake) Close() error                                       { return nil }

func TestIdentityOutboxPublisherMarksPublishedEvents(t *testing.T) {
	t.Parallel()

	event := ports.OutboxEvent{ID: uuid.New()}
	outbox := &identityOutboxFake{events: []ports.OutboxEvent{event}}
	publisher := NewOutboxPublisher(outbox, identityEventPublisherFake{})

	err := publisher.RunOnce(context.Background())

	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{event.ID}, outbox.published)
	require.Empty(t, outbox.failed)
}

func TestIdentityOutboxPublisherMarksFailedEvents(t *testing.T) {
	t.Parallel()

	event := ports.OutboxEvent{ID: uuid.New()}
	outbox := &identityOutboxFake{events: []ports.OutboxEvent{event}}
	publisher := NewOutboxPublisher(outbox, identityEventPublisherFake{err: errors.New("broker unavailable")})

	err := publisher.RunOnce(context.Background())

	require.Error(t, err)
	require.Equal(t, []uuid.UUID{event.ID}, outbox.failed)
	require.Empty(t, outbox.published)
}

func TestIdentityOutboxPublisherReturnsClaimError(t *testing.T) {
	t.Parallel()

	claimErr := errors.New("database unavailable")
	publisher := NewOutboxPublisher(&identityOutboxFake{claimErr: claimErr}, identityEventPublisherFake{})

	require.ErrorIs(t, publisher.RunOnce(context.Background()), claimErr)
}
