package websocket

import (
	"errors"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

type fakeAccessTokenVerifier struct {
	userID uuid.UUID
}

func (f fakeAccessTokenVerifier) UserID(token string) (uuid.UUID, error) {
	if token != "valid-token" {
		return uuid.Nil(), errors.New("invalid token")
	}
	return f.userID, nil
}

func TestHandlerAuthenticatesBearerToken(t *testing.T) {
	userID := uuid.New()
	handler := &Handler{verifier: fakeAccessTokenVerifier{userID: userID}}
	request := httptest.NewRequest("GET", "http://example.test/v1/messaging/ws", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	result, err := handler.userID(request)

	require.NoError(t, err)
	require.Equal(t, userID, result)
}

func TestHandlerRejectsMissingWebSocketToken(t *testing.T) {
	handler := &Handler{verifier: fakeAccessTokenVerifier{userID: uuid.New()}}
	request := httptest.NewRequest("GET", "http://example.test/v1/messaging/ws", nil)

	_, err := handler.userID(request)

	require.Error(t, err)
}
