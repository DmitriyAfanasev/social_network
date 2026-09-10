package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/libs/platform/auth"
)

type fakeProfilesVerifier struct{ userID uuid.UUID }

func (f fakeProfilesVerifier) UserID(string) (uuid.UUID, error) { return f.userID, nil }

func TestProfileSettingsUnauthorizedBranches(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "set handle", handler: handler.SetHandle},
		{name: "update profile", handler: handler.UpdatePublicProfile},
		{name: "get privacy", handler: handler.GetPrivacy},
		{name: "update privacy", handler: handler.UpdatePrivacy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, httptest.NewRequest(http.MethodGet, "/", nil))

			// Assert
			require.Equal(t, http.StatusUnauthorized, response.Code)
			require.Equal(t, "unauthorized", decodeProfilesErrorCode(t, response))
		})
	}
}

func TestProfileSettingsRejectMalformedJSONAfterAuthentication(t *testing.T) {
	handler := &Handler{}
	secured := func(handler http.HandlerFunc) http.Handler {
		return auth.Middleware(fakeProfilesVerifier{userID: uuid.New()})(handler)
	}
	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "set handle", handler: handler.SetHandle},
		{name: "update profile", handler: handler.UpdatePublicProfile},
		{name: "update privacy", handler: handler.UpdatePrivacy},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			request := httptest.NewRequest(http.MethodPut, "/", strings.NewReader("{"))
			request.Header.Set("Authorization", "Bearer token")
			response := httptest.NewRecorder()

			// Act
			secured(tt.handler).ServeHTTP(response, request)

			// Assert
			require.Equal(t, http.StatusBadRequest, response.Code)
			require.Equal(t, "invalid_json", decodeProfilesErrorCode(t, response))
		})
	}
}

func decodeProfilesErrorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	code, ok := body["code"].(string)
	require.True(t, ok)
	return code
}
