package events

import (
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestDecodeEvent(t *testing.T) {
	t.Parallel()

	eventID, aggregateID := uuid.New(), uuid.New()
	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	payload := []byte(`{"actor_id":"` + eventID.String() + `"}`)
	value := []byte(`{"event_id":"` + eventID.String() + `","event_type":"social.friendship.created","aggregate_id":"` + aggregateID.String() + `","payload":` + string(payload) + `,"created_at":"` + createdAt.Format(time.RFC3339Nano) + `"}`)

	event, err := decodeEvent(value)

	require.NoError(t, err)
	require.Equal(t, eventID, event.ID)
	require.Equal(t, "social.friendship.created", event.EventType)
	require.Equal(t, aggregateID, *event.AggregateID)
	require.JSONEq(t, string(payload), string(event.Payload))
	require.Equal(t, createdAt, event.CreatedAt)
}

func TestDecodeEventRejectsIncompleteEnvelope(t *testing.T) {
	t.Parallel()

	_, err := decodeEvent([]byte(`{"event_type":"social.friendship.created","payload":{}}`))

	require.Error(t, err)
}
