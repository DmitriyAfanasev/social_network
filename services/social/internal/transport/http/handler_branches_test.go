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

type fakeSocialVerifier struct{ userID uuid.UUID }

func (f fakeSocialVerifier) UserID(string) (uuid.UUID, error) { return f.userID, nil }

func TestHandlerUnauthorizedBranches(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "block", handler: handler.Block},
		{name: "unblock", handler: handler.Unblock},
		{name: "add friend", handler: handler.AddFriend},
		{name: "remove friend", handler: handler.RemoveFriend},
		{name: "list requests", handler: handler.ListFriendRequests},
		{name: "accept request", handler: handler.AcceptFriendRequest},
		{name: "relationship", handler: handler.GetRelationship},
		{name: "public friends", handler: handler.GetPublicFriends},
		{name: "relationships", handler: handler.GetRelationships},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, httptest.NewRequest(http.MethodGet, "/", nil))

			// Assert
			require.Equal(t, http.StatusUnauthorized, response.Code)
			require.Equal(t, "unauthorized", decodeSocialErrorCode(t, response))
		})
	}
}

func TestHandlerInvalidTargetIDAfterAuthentication(t *testing.T) {
	// Arrange
	handler := &Handler{}
	request := httptest.NewRequest(http.MethodPut, "/friendships/not-uuid", nil)
	request.Header.Set("Authorization", "Bearer token")
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("targetID", "not-uuid")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	response := httptest.NewRecorder()
	secured := auth.Middleware(fakeSocialVerifier{userID: uuid.New()})(http.HandlerFunc(handler.AddFriend))

	// Act
	secured.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, "validation_error", decodeSocialErrorCode(t, response))
}

func decodeSocialErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	code, ok := body["code"].(string)
	require.True(t, ok)
	return code
}
