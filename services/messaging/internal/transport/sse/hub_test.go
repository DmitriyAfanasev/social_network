package sse

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/messaging/internal/ports"
)

func TestHubPublishesOnlyToRecipients(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	recipient, other := uuid.New(), uuid.New()
	channel, unsubscribe := hub.Subscribe(recipient)
	defer unsubscribe()
	otherChannel, otherUnsubscribe := hub.Subscribe(other)
	defer otherUnsubscribe()

	hub.Publish(ports.RealtimeEvent{RecipientIDs: []uuid.UUID{recipient}, Payload: []byte(`{"type":"message.new"}`)})

	require.Equal(t, []byte(`{"type":"message.new"}`), <-channel)
	select {
	case <-otherChannel:
		t.Fatal("event was published to a non-recipient")
	default:
	}
}

func TestHubDropsEventsForFullSubscriber(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	userID := uuid.New()
	channel, unsubscribe := hub.Subscribe(userID)
	defer unsubscribe()

	for index := 0; index < subscriberBuffer+5; index++ {
		hub.Publish(ports.RealtimeEvent{RecipientIDs: []uuid.UUID{userID}, Payload: []byte("event")})
	}

	count := 0
	for {
		select {
		case <-channel:
			count++
		default:
			require.Equal(t, subscriberBuffer, count)
			return
		}
	}
}
