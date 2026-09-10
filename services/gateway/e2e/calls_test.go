package e2e

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestCallSignalingLifecycle(t *testing.T) {
	// Arrange
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	mailpitURL := strings.TrimRight(os.Getenv("E2E_MAILPIT_URL"), "/")
	if baseURL == "" || mailpitURL == "" {
		t.Skip("E2E_BASE_URL и E2E_MAILPIT_URL не заданы")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	_, callerAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	calleeID, calleeAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	waitForProfile(t, client, baseURL, calleeID, calleeAccessToken)
	status, conversationBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/messaging/conversations/direct", callerAccessToken, map[string]string{
		"other_user_id": calleeID,
	})
	require.Equal(t, http.StatusOK, status, "conversation response: %#v", conversationBody)
	conversationID := requireString(t, conversationBody, "id")

	callerConn := dialCallWebSocket(t, baseURL, callerAccessToken)
	defer callerConn.Close()
	calleeConn := dialCallWebSocket(t, baseURL, calleeAccessToken)
	defer calleeConn.Close()

	// Act
	require.NoError(t, callerConn.WriteJSON(map[string]any{
		"type": "conversation.subscribe", "conversation_id": conversationID,
	}))
	require.NoError(t, calleeConn.WriteJSON(map[string]any{
		"type": "conversation.subscribe", "conversation_id": conversationID,
	}))

	// Assert
	require.Equal(t, "conversation.subscribed", readCallEvent(t, callerConn)["type"])
	require.Equal(t, "conversation.subscribed", readCallEvent(t, calleeConn)["type"])

	// Act
	require.NoError(t, callerConn.WriteJSON(map[string]any{
		"type": "call.start", "conversation_id": conversationID,
		"target_user_id": calleeID, "call_type": "video",
	}))

	// Assert
	inviteForCaller := readCallEvent(t, callerConn)
	inviteForCallee := readCallEvent(t, calleeConn)
	require.Equal(t, "call.invite", inviteForCaller["type"])
	require.Equal(t, "call.invite", inviteForCallee["type"])
	callID := requireEventString(t, inviteForCaller, "call_id")
	require.Equal(t, callID, inviteForCallee["call_id"])
	require.Equal(t, "ringing", inviteForCallee["status"])

	// Act
	require.NoError(t, calleeConn.WriteJSON(map[string]any{
		"type": "call.accept", "conversation_id": conversationID, "call_id": callID,
	}))

	// Assert
	acceptedForCaller := readCallEvent(t, callerConn)
	acceptedForCallee := readCallEvent(t, calleeConn)
	require.Equal(t, "call.accept", acceptedForCaller["type"])
	require.Equal(t, "call.accept", acceptedForCallee["type"])
	require.Equal(t, "active", acceptedForCaller["status"])

	// Act
	require.NoError(t, callerConn.WriteJSON(map[string]any{
		"type": "call.signal", "conversation_id": conversationID, "call_id": callID,
		"signal": map[string]any{"kind": "offer", "sdp": map[string]string{"type": "offer"}},
	}))

	// Assert
	signalForCaller := readCallEvent(t, callerConn)
	signalForCallee := readCallEvent(t, calleeConn)
	require.Equal(t, "call.signal", signalForCaller["type"])
	require.Equal(t, "call.signal", signalForCallee["type"])
	require.Equal(t, callID, signalForCallee["call_id"])
	signal, ok := signalForCallee["signal"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "offer", signal["kind"])

	// Act
	require.NoError(t, callerConn.WriteJSON(map[string]any{
		"type": "call.end", "conversation_id": conversationID, "call_id": callID,
	}))

	// Assert
	endedForCaller := readCallEvent(t, callerConn)
	endedForCallee := readCallEvent(t, calleeConn)
	require.Equal(t, "call.end", endedForCaller["type"])
	require.Equal(t, "call.end", endedForCallee["type"])
	require.Equal(t, "ended", endedForCallee["status"])
}

func dialCallWebSocket(t *testing.T, baseURL string, accessToken string) *websocket.Conn {
	t.Helper()
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/v1/calls/ws?access_token=" + url.QueryEscape(accessToken)
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if response != nil {
		defer response.Body.Close()
	}
	require.NoError(t, err)
	return conn
}

func readCallEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	var event map[string]any
	require.NoError(t, conn.ReadJSON(&event))
	return event
}

func requireEventString(t *testing.T, event map[string]any, key string) string {
	t.Helper()
	value, ok := event[key].(string)
	require.True(t, ok, "event field %q: %#v", key, event)
	require.NotEmpty(t, value)
	return value
}
