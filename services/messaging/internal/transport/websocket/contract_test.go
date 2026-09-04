package websocket

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func newWebSocketContractServer(t *testing.T, hub *Hub, userID uuid.UUID) *httptest.Server {
	t.Helper()
	handler := NewHandler(nil, fakeAccessTokenVerifier{userID: userID}, hub, nil)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("environment does not allow test listeners: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func dialContractWebSocket(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/?access_token=valid-token"
	connection, response, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
	t.Cleanup(func() { _ = connection.Close() })
	return connection
}

func waitForClient(t *testing.T, hub *Hub, userID uuid.UUID, expected int) {
	t.Helper()
	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients[userID]) == expected
	}, time.Second, 5*time.Millisecond)
}

func TestWebSocketContractRejectsUnauthenticatedHandshake(t *testing.T) {
	hub := NewHub()
	server := newWebSocketContractServer(t, hub, uuid.New())

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	_, response, err := websocket.DefaultDialer.Dial(url, nil)

	require.Error(t, err)
	require.NotNil(t, response)
	if response.Body != nil {
		defer response.Body.Close()
	}
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestWebSocketContractPreservesMessageOrdering(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	server := newWebSocketContractServer(t, hub, userID)
	connection := dialContractWebSocket(t, server)
	waitForClient(t, hub, userID, 1)

	for sequence := 1; sequence <= 3; sequence++ {
		payload, err := json.Marshal(map[string]int{"sequence": sequence})
		require.NoError(t, err)
		hub.Broadcast([]uuid.UUID{userID}, payload)
	}

	for sequence := 1; sequence <= 3; sequence++ {
		_, payload, err := connection.ReadMessage()
		require.NoError(t, err)
		require.JSONEq(t, `{"sequence":`+strconv.Itoa(sequence)+`}`, string(payload))
	}
}

func TestWebSocketContractReconnectRemovesPreviousConnection(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()
	server := newWebSocketContractServer(t, hub, userID)
	first := dialContractWebSocket(t, server)
	waitForClient(t, hub, userID, 1)

	require.NoError(t, first.Close())
	waitForClient(t, hub, userID, 0)

	second := dialContractWebSocket(t, server)
	waitForClient(t, hub, userID, 1)
	require.NoError(t, second.Close())
}
