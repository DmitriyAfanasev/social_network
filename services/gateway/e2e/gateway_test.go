package e2e

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayRoutingAndErrorEnvelope(t *testing.T) {
	// Arrange
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	if baseURL == "" {
		t.Skip("E2E_BASE_URL не задан")
	}
	client := &http.Client{Timeout: 5 * time.Second}

	// Act
	status, notFoundBody := jsonRequest(t, client, http.MethodGet, baseURL+"/v1/route-that-does-not-exist", map[string]string{})

	// Assert
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, "not_found", notFoundBody["code"])
	require.NotEmpty(t, notFoundBody["message"])

	// Act
	status, methodBody := jsonRequest(t, client, http.MethodPost, baseURL+"/healthz", map[string]string{})

	// Assert
	require.Equal(t, http.StatusMethodNotAllowed, status)
	require.Equal(t, "method_not_allowed", methodBody["code"])
	require.NotEmpty(t, methodBody["message"])

	// Act
	status, unauthorizedBody := jsonRequest(t, client, http.MethodPost, baseURL+"/v1/content/posts", map[string]string{"body": "unauthorized"})

	// Assert
	require.Equal(t, http.StatusUnauthorized, status)
	require.Equal(t, "unauthorized", unauthorizedBody["code"])
	require.NotEmpty(t, unauthorizedBody["message"])
}
