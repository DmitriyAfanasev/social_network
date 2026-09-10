package httptransport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"general-project/messaging/internal/application"
	"general-project/messaging/internal/ports"

	"github.com/stretchr/testify/require"
)

func TestPagination(t *testing.T) {
	for _, tc := range []struct {
		query         string
		offset, limit int
		ok            bool
	}{
		{"", 0, 50, true}, {"?offset=10&limit=25", 10, 25, true}, {"?offset=bad", 0, 0, false}, {"?limit=bad", 0, 0, false},
	} {
		// Arrange
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/"+tc.query, nil)

		// Act
		offset, limit, ok := pagination(rr, r)

		// Assert
		require.Equal(t, tc.offset, offset)
		require.Equal(t, tc.limit, limit)
		require.Equal(t, tc.ok, ok)
	}
}

func TestDecodeMessageRequest(t *testing.T) {
	// Arrange
	mediaID := uuid.New()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"body":"hello","media_id":"`+mediaID.String()+`"}`))

	// Act
	got, err := decodeMessageRequest(r)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "hello", got.Body)
	require.Equal(t, mediaID, *got.MediaID)

	// Arrange
	invalidRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"media_id":"bad"}`))

	// Act
	_, err = decodeMessageRequest(invalidRequest)

	// Assert
	require.Error(t, err)
}

func TestMessagingErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{application.ErrValidation, 422, "validation_error"}, {application.ErrInteractionForbidden, 403, "message_forbidden"}, {ports.ErrForbidden, 403, "forbidden"}, {ports.ErrNotFound, 404, "messaging_resource_not_found"},
	} {
		status, code, _ := messagingErrorStatus(tc.err)
		require.Equal(t, tc.status, status)
		require.Equal(t, tc.code, code)
	}
}

func TestNotificationEventNameFailsClosedForInvalidNames(t *testing.T) {
	for _, payload := range [][]byte{[]byte(`not-json`), []byte(`{"type":""}`), []byte(`{"type":"bad\nname"}`)} {
		require.Equal(t, "notification", notificationEventName(payload))
	}
	require.Equal(t, "message.created", notificationEventName([]byte(`{"type":"message.created"}`)))
}
