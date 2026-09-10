package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"general-project/libs/platform/auth"
)

type fakeContentVerifier struct{ userID uuid.UUID }

func (f fakeContentVerifier) UserID(string) (uuid.UUID, error) { return f.userID, nil }

func TestHandlerUnauthorizedBranches(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "create", handler: handler.Create},
		{name: "update", handler: handler.Update},
		{name: "delete", handler: handler.Delete},
		{name: "create comment", handler: handler.CreateComment},
		{name: "delete comment", handler: handler.DeleteComment},
		{name: "like", handler: handler.Like},
		{name: "unlike", handler: handler.Unlike},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, request)

			// Assert
			require.Equal(t, http.StatusUnauthorized, response.Code)
			require.Equal(t, "unauthorized", decodeErrorCode(t, response))
		})
	}
}

func TestHandlerAuthenticatedValidationBranches(t *testing.T) {
	userID := uuid.New()
	handler := &Handler{}
	withAuth := func(handlerFunc http.HandlerFunc) http.Handler {
		return auth.Middleware(fakeContentVerifier{userID: userID})(handlerFunc)
	}

	// Arrange
	createRequest := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader("{"))
	createRequest.Header.Set("Authorization", "Bearer test")
	createResponse := httptest.NewRecorder()

	// Act
	withAuth(handler.Create).ServeHTTP(createResponse, createRequest)

	// Assert
	require.Equal(t, http.StatusBadRequest, createResponse.Code)
	require.Equal(t, "invalid_json", decodeErrorCode(t, createResponse))

	// Arrange
	updateRequest := httptest.NewRequest(http.MethodPatch, "/posts/not-uuid", strings.NewReader("{}"))
	updateRequest.Header.Set("Authorization", "Bearer test")
	updateRequest = withPathParam(updateRequest, "postID", "not-uuid")
	updateResponse := httptest.NewRecorder()

	// Act
	withAuth(handler.Update).ServeHTTP(updateResponse, updateRequest)

	// Assert
	require.Equal(t, http.StatusBadRequest, updateResponse.Code)
	require.Equal(t, "invalid_post_id", decodeErrorCode(t, updateResponse))

	// Arrange
	commentsRequest := httptest.NewRequest(http.MethodGet, "/posts/not-uuid/comments", nil)
	commentsRequest = withPathParam(commentsRequest, "postID", "not-uuid")
	commentsResponse := httptest.NewRecorder()

	// Act
	handler.ListComments(commentsResponse, commentsRequest)

	// Assert
	require.Equal(t, http.StatusBadRequest, commentsResponse.Code)
	require.Equal(t, "invalid_post_id", decodeErrorCode(t, commentsResponse))
}

func withPathParam(request *http.Request, name string, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(name, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func decodeErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	code, ok := body["code"].(string)
	require.True(t, ok)
	return code
}
