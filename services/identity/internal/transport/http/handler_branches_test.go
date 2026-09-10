package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthHandlersRejectMalformedJSON(t *testing.T) {
	handler := &Handler{}
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "register", handler: handler.Register},
		{name: "request confirmation", handler: handler.RequestRegistrationConfirmation},
		{name: "confirm registration", handler: handler.ConfirmRegistration},
		{name: "password reset request", handler: handler.RequestPasswordReset},
		{name: "password reset", handler: handler.ResetPassword},
		{name: "login", handler: handler.Login},
		{name: "refresh", handler: handler.Refresh},
		{name: "logout", handler: handler.Logout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			response := httptest.NewRecorder()

			// Act
			tt.handler(response, request)

			// Assert
			require.Equal(t, http.StatusBadRequest, response.Code)
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, "invalid_json", body["code"])
		})
	}
}
