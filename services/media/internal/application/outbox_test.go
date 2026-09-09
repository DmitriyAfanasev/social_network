package application

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/media/internal/ports"
)

type fakeImageJobPublisher struct {
	jobs []ports.ImageOptimizationJob
	err  error
}

func (p *fakeImageJobPublisher) Publish(_ context.Context, job ports.ImageOptimizationJob) error {
	p.jobs = append(p.jobs, job)
	return p.err
}

type fakeEventPublisher struct {
	events []ports.OutboxEvent
}

func (p *fakeEventPublisher) Publish(_ context.Context, event ports.OutboxEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakeEventPublisher) Close() error { return nil }

type publishingOutbox struct {
	events    []ports.OutboxEvent
	published []uuid.UUID
}

func (o *publishingOutbox) Claim(context.Context, int, time.Time) ([]ports.OutboxEvent, error) {
	return o.events, nil
}

func (o *publishingOutbox) MarkPublished(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	o.published = append(o.published, eventID)
	return nil
}

func (o *publishingOutbox) MarkFailed(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}

func TestOutboxPublisherQueuesImageOptimizationForNewImage(t *testing.T) {
	mediaID := uuid.New()
	event := ports.OutboxEvent{ID: uuid.New(), EventType: "media.media.created", Payload: []byte(`{"media_id":"` + mediaID.String() + `","media_type":"image"}`)}
	outbox := &publishingOutbox{events: []ports.OutboxEvent{event}}
	jobs := &fakeImageJobPublisher{}
	publisher := NewOutboxPublisher(outbox, &fakeEventPublisher{}, jobs)

	require.NoError(t, publisher.RunOnce(context.Background()))
	require.Equal(t, []ports.ImageOptimizationJob{{MediaID: mediaID}}, jobs.jobs)
	require.Equal(t, []uuid.UUID{event.ID}, outbox.published)
}
