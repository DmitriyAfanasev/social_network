package application

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/analytics/internal/domain"
	"general-project/analytics/internal/ports"
)

type fakeOutboxRepository struct {
	events       []domain.Event
	published    []uuid.UUID
	failed       []uuid.UUID
	nextAttempt  time.Time
	failureCause string
	failedEvents []uuid.UUID
}

func (f *fakeOutboxRepository) Claim(_ context.Context, _ int, _ time.Time) ([]domain.Event, error) {
	return f.events, nil
}

func (f *fakeOutboxRepository) MarkPublished(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	f.published = append(f.published, eventID)
	return nil
}

func (f *fakeOutboxRepository) MarkFailed(_ context.Context, eventID uuid.UUID, nextAttempt time.Time, reason string) error {
	f.failed = append(f.failed, eventID)
	f.nextAttempt = nextAttempt
	f.failureCause = reason
	return nil
}

type fakeEventPublisher struct {
	events []domain.Event
	err    error
	dlq    []uuid.UUID
}

func (f *fakeEventPublisher) Publish(_ context.Context, event domain.Event) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, event)
	return nil
}

func (f *fakeEventPublisher) Close() error { return nil }

func (f *fakeEventPublisher) PublishDeadLetter(_ context.Context, event domain.Event, _ string, _ int) error {
	f.dlq = append(f.dlq, event.ID)
	return nil
}

func TestOutboxPublisherMarksSuccessAndSchedulesRetry(t *testing.T) {
	t.Parallel()

	successID := uuid.New()
	failureID := uuid.New()
	outbox := &fakeOutboxRepository{events: []domain.Event{
		{ID: successID, Attempts: 1},
		{ID: failureID, Attempts: 2},
	}}
	broker := &fakeEventPublisher{}
	publisher := NewOutboxPublisher(outbox, broker)

	err := publisher.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{successID, failureID}, outbox.published)

	broker.err = errors.New("kafka unavailable")
	err = publisher.RunOnce(context.Background())
	require.Error(t, err)
	require.Equal(t, []uuid.UUID{successID, failureID}, outbox.failed)
	require.NotEmpty(t, outbox.nextAttempt)
	require.EqualError(t, errors.New(outbox.failureCause), "kafka unavailable")
}

type fakeProcessedRepository struct {
	claimed   map[uuid.UUID]bool
	processed []uuid.UUID
	released  []uuid.UUID
}

func (f *fakeProcessedRepository) Claim(_ context.Context, eventID uuid.UUID, _ time.Time) (ports.ClaimResult, error) {
	if f.claimed[eventID] {
		return ports.ClaimResult{}, nil
	}
	f.claimed[eventID] = true
	return ports.ClaimResult{Claimed: true, Attempts: 1}, nil
}

func (f *fakeProcessedRepository) MarkFailed(_ context.Context, eventID uuid.UUID, _ string) error {
	f.released = append(f.released, eventID)
	delete(f.claimed, eventID)
	return nil
}

func (f *fakeProcessedRepository) MarkProcessed(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	f.processed = append(f.processed, eventID)
	return nil
}

type fakeEventHandler struct {
	count int
	err   error
}

func (f *fakeEventHandler) Handle(_ context.Context, _ domain.Event) error {
	f.count++
	return f.err
}

func TestEventConsumerIsIdempotentOnRedelivery(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	processed := &fakeProcessedRepository{claimed: map[uuid.UUID]bool{}}
	handler := &fakeEventHandler{}
	consumer := NewEventConsumer(processed, handler, &fakeEventPublisher{}, 3, 0)
	event := domain.Event{ID: eventID, EventType: "post.created", Payload: []byte(`{}`), CreatedAt: time.Now().UTC()}

	require.NoError(t, consumer.Handle(context.Background(), event))
	require.NoError(t, consumer.Handle(context.Background(), event))
	require.Equal(t, 1, handler.count)
	require.Equal(t, []uuid.UUID{eventID}, processed.processed)
}

func TestEventConsumerPublishesDeadLetterAfterLimit(t *testing.T) {
	eventID := uuid.New()
	processed := &fakeProcessedRepository{claimed: map[uuid.UUID]bool{}}
	broker := &fakeEventPublisher{}
	consumer := NewEventConsumer(processed, &fakeEventHandler{err: errors.New("invalid payload")}, broker, 1, 0)
	event := domain.Event{ID: eventID, EventType: "post.created", Payload: []byte(`{}`), CreatedAt: time.Now().UTC()}

	err := consumer.Handle(context.Background(), event)

	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{eventID}, broker.dlq)
	require.Equal(t, []uuid.UUID{eventID}, processed.processed)
}
