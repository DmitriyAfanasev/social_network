package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"general-project/libs/platform/auth"
)

type fakeMessagingVerifier struct{ userID uuid.UUID }

func (f fakeMessagingVerifier) UserID(string) (uuid.UUID, error) { return f.userID, nil }

func TestHandlerUnauthorizedBranches(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "direct conversation", handler: handler.GetOrCreateDirect},
		{name: "list conversations", handler: handler.ListConversations},
		{name: "archive", handler: handler.ArchiveConversation},
		{name: "hide", handler: handler.HideConversation},
		{name: "mark unread", handler: handler.MarkConversationUnread},
		{name: "pin", handler: handler.PinConversation},
		{name: "mute", handler: handler.MuteConversation},
		{name: "clear history", handler: handler.ClearHistory},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, httptest.NewRequest(http.MethodPost, "/", nil))

			// Assert
			require.Equal(t, http.StatusUnauthorized, response.Code)
			require.Equal(t, "unauthorized", decodeMessagingErrorCode(t, response))
		})
	}
}

func TestHandlerInvalidConversationIDAfterAuthentication(t *testing.T) {
	// Arrange
	handler := &Handler{}
	request := httptest.NewRequest(http.MethodPost, "/conversations/not-uuid/archive", nil)
	request.Header.Set("Authorization", "Bearer token")
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("conversationID", "not-uuid")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	response := httptest.NewRecorder()
	secured := auth.Middleware(fakeMessagingVerifier{userID: uuid.New()})(http.HandlerFunc(handler.ArchiveConversation))

	// Act
	secured.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, "invalid_conversation_id", decodeMessagingErrorCode(t, response))
}

func decodeMessagingErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	code, ok := body["code"].(string)
	require.True(t, ok)
	return code
}
