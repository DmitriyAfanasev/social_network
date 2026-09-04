package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"general-project/gateway/internal/config"
)

func TestNewRouterRoutesAndRewritesRequests(t *testing.T) {
	t.Parallel()

	var receivedPath string
	var receivedRequestID string
	backend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.Path
		receivedRequestID = request.Header.Get("X-Request-ID")
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	cfg := testConfig(backend.URL)
	router, err := NewRouter(cfg, slog.Default(), nil)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/v1/messaging/ws?access_token=test", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "/ws", receivedPath)
	require.Equal(t, "request-123", receivedRequestID)
}

func TestNewRouterProxiesSSEPathWithoutRewrite(t *testing.T) {
	t.Parallel()

	var receivedPath string
	backend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.Path
		_, _ = io.WriteString(writer, "data: ready\n\n")
	}))
	defer backend.Close()

	router, err := NewRouter(testConfig(backend.URL), slog.Default(), nil)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/v1/notifications/stream", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "/v1/notifications/stream", receivedPath)
	require.Equal(t, "data: ready\n\n", response.Body.String())
}

func TestReadinessRequiresAllUpstreams(t *testing.T) {
	t.Parallel()

	readyBackend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/readyz", request.URL.Path)
		writer.WriteHeader(http.StatusOK)
	}))
	defer readyBackend.Close()

	readyConfig := testConfig(readyBackend.URL)
	readyConfig.CallsURL = "http://127.0.0.1:1"
	router, err := NewRouter(readyConfig, slog.Default(), nil)
	require.NoError(t, err)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "dependencies_not_ready")
}

func TestCORSPreflight(t *testing.T) {
	t.Parallel()

	cfg := testConfig("http://127.0.0.1:1")
	router, err := NewRouter(cfg, slog.Default(), nil)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodOptions, "/v1/auth/login", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "http://localhost:5173", response.Header().Get("Access-Control-Allow-Origin"))
}

func testConfig(backendURL string) config.Config {
	return config.Config{
		IdentityURL:      backendURL,
		ProfilesURL:      backendURL,
		SocialURL:        backendURL,
		ContentURL:       backendURL,
		MediaURL:         backendURL,
		MessagingURL:     backendURL,
		CallsURL:         backendURL,
		AllowedOrigins:   []string{"http://localhost:5173"},
		RateLimit:        600,
		RateLimitWindow:  time.Minute,
		ReadinessTimeout: time.Second,
	}
}

func TestNewRouterRejectsInvalidUpstream(t *testing.T) {
	t.Parallel()

	cfg := testConfig("://invalid")
	_, err := NewRouter(cfg, slog.Default(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid identity upstream URL")
}
