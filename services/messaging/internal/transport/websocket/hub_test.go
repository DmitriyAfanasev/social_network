package websocket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientQueuePreservesOrderAndRejectsOverflow(t *testing.T) {
	client := &client{send: make(chan []byte, 2), closed: make(chan struct{})}

	require.True(t, client.enqueue([]byte(`{"sequence":1}`)))
	require.True(t, client.enqueue([]byte(`{"sequence":2}`)))
	require.False(t, client.enqueue([]byte(`{"sequence":3}`)))
	require.Equal(t, []byte(`{"sequence":1}`), <-client.send)
	require.Equal(t, []byte(`{"sequence":2}`), <-client.send)
}
