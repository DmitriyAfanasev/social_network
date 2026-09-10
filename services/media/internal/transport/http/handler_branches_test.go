package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlerWriteEndpointsRequireAuthentication(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{name: "upload", method: http.MethodPost, path: "/v1/media", handler: handler.Upload},
		{name: "delete", method: http.MethodDelete, path: "/v1/media/id", handler: handler.Delete},
		{name: "create video", method: http.MethodPost, path: "/v1/media/videos", handler: handler.CreateVideo},
		{name: "create album", method: http.MethodPost, path: "/v1/media/videos/albums", handler: handler.CreateAlbum},
		{name: "create music", method: http.MethodPost, path: "/v1/media/music", handler: handler.CreateMusic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, httptest.NewRequest(tt.method, tt.path, nil))

			// Assert
			require.Equal(t, http.StatusUnauthorized, response.Code)
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, "unauthorized", body["code"])
		})
	}
}

func TestHandlerReadEndpointsRejectInvalidIDs(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "media metadata", handler: handler.GetByID},
		{name: "media content", handler: handler.StreamContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			request := httptest.NewRequest(http.MethodGet, "/v1/media/not-uuid", nil)
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, request)

			// Assert
			require.Equal(t, http.StatusBadRequest, response.Code)
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, "invalid_media_id", body["code"])
		})
	}
}
