package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/messaging/internal/ports"
)

type fakeNotificationStream struct {
	events []ports.RealtimeEvent
}

func (f *fakeNotificationStream) Subscribe(uuid.UUID) (<-chan []byte, func()) {
	return nil, func() {}
}

func (f *fakeNotificationStream) Publish(event ports.RealtimeEvent) {
	f.events = append(f.events, event)
}

func TestNotificationEventHandlerPublishesFriendAccepted(t *testing.T) {
	t.Parallel()

	stream := &fakeNotificationStream{}
	handler := NewNotificationEventHandler(stream)
	actorID, targetID := uuid.New(), uuid.New()
	eventID := uuid.New()

	err := handler.Handle(context.Background(), ports.IntegrationEvent{
		ID:          eventID,
		EventType:   friendshipCreatedEvent,
		AggregateID: &targetID,
		Payload:     mustJSON(t, map[string]uuid.UUID{"actor_id": actorID, "target_id": targetID}),
		CreatedAt:   time.Now().UTC(),
	})

	require.NoError(t, err)
	require.Len(t, stream.events, 1)
	require.Equal(t, []uuid.UUID{targetID}, stream.events[0].RecipientIDs)

	var notification map[string]any
	require.NoError(t, json.Unmarshal(stream.events[0].Payload, &notification))
	require.Equal(t, friendAcceptedType, notification["type"])
	require.Equal(t, actorID.String(), notification["actor_id"])
	require.Equal(t, "Заявку в друзья приняли", notification["message"])
}

func TestNotificationEventHandlerDeduplicatesEvent(t *testing.T) {
	t.Parallel()

	stream := &fakeNotificationStream{}
	handler := NewNotificationEventHandler(stream)
	actorID, targetID := uuid.New(), uuid.New()
	event := ports.IntegrationEvent{
		ID:        uuid.New(),
		EventType: friendshipCreatedEvent,
		Payload:   mustJSON(t, map[string]uuid.UUID{"actor_id": actorID, "target_id": targetID}),
		CreatedAt: time.Now().UTC(),
	}

	require.NoError(t, handler.Handle(context.Background(), event))
	require.NoError(t, handler.Handle(context.Background(), event))
	require.Len(t, stream.events, 1)
}

func TestNotificationEventHandlerPublishesFriendRequested(t *testing.T) {
	t.Parallel()

	stream := &fakeNotificationStream{}
	handler := NewNotificationEventHandler(stream)
	actorID, targetID := uuid.New(), uuid.New()

	err := handler.Handle(context.Background(), ports.IntegrationEvent{
		ID:        uuid.New(),
		EventType: friendRequestCreatedEvent,
		Payload:   mustJSON(t, map[string]uuid.UUID{"actor_id": actorID, "target_id": targetID}),
		CreatedAt: time.Now().UTC(),
	})

	require.NoError(t, err)
	require.Len(t, stream.events, 1)
	var notification map[string]any
	require.NoError(t, json.Unmarshal(stream.events[0].Payload, &notification))
	require.Equal(t, friendRequestedType, notification["type"])
	require.Equal(t, actorID.String(), notification["actor_id"])
}

func TestNotificationEventHandlerIgnoresUnsupportedEvent(t *testing.T) {
	t.Parallel()

	stream := &fakeNotificationStream{}
	handler := NewNotificationEventHandler(stream)

	err := handler.Handle(context.Background(), ports.IntegrationEvent{ID: uuid.New(), EventType: "social.subscription.created", Payload: []byte(`{}`)})

	require.NoError(t, err)
	require.Empty(t, stream.events)
}

func TestNotificationEventHandlerSupportsLegacyFriendshipPayload(t *testing.T) {
	t.Parallel()

	stream := &fakeNotificationStream{}
	handler := NewNotificationEventHandler(stream)
	actorID, targetID := uuid.New(), uuid.New()

	err := handler.Handle(context.Background(), ports.IntegrationEvent{
		ID:          uuid.New(),
		EventType:   friendshipCreatedEvent,
		AggregateID: &targetID,
		Payload:     mustJSON(t, map[string]uuid.UUID{"user_id": actorID, "friend_id": targetID}),
		CreatedAt:   time.Now().UTC(),
	})

	require.NoError(t, err)
	require.Len(t, stream.events, 1)
	require.Equal(t, []uuid.UUID{targetID}, stream.events[0].RecipientIDs)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return payload
}
