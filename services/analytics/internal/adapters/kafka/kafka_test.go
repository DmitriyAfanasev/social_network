package kafka

import (
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestDecodeEvent(t *testing.T) {
	// Arrange
	id := uuid.New()
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	payload := []byte(`{"event_id":"` + id.String() + `","event_type":"post.created","payload":{"post_id":"x"},"created_at":"` + createdAt.Format(time.RFC3339) + `"}`)

	// Act
	event, err := decodeEvent(payload)

	// Assert
	require.NoError(t, err)
	require.Equal(t, id, event.ID)
	require.Equal(t, "post.created", event.EventType)
	require.JSONEq(t, `{"post_id":"x"}`, string(event.Payload))
	require.Equal(t, createdAt, event.CreatedAt)
}

func TestDecodeEventRejectsInvalidEnvelope(t *testing.T) {
	for _, payload := range [][]byte{
		[]byte(`not-json`),
		[]byte(`{"event_type":"post.created","payload":{},"created_at":"2026-01-02T03:04:05Z"}`),
		[]byte(`{"event_id":"` + uuid.New().String() + `","event_type":"","payload":{},"created_at":"2026-01-02T03:04:05Z"}`),
	} {
		// Act
		_, err := decodeEvent(payload)

		// Assert
		require.Error(t, err)
	}
}
