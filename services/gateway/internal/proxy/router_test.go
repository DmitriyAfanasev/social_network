package proxy

import (
	"io"
	"log/slog"
	"net"
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
	backend := newIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.Path
		receivedRequestID = request.Header.Get("X-Request-ID")
		writer.WriteHeader(http.StatusNoContent)
	}))

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
	backend := newIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.Path
		if _, err := io.WriteString(writer, "data: ready\n\n"); err != nil {
			t.Errorf("write SSE response: %v", err)
		}
	}))

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

	readyBackend := newIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/readyz" {
			t.Errorf("path = %q, want /readyz", request.URL.Path)
		}
		writer.WriteHeader(http.StatusOK)
	}))

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

func newIPv4Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("environment does not allow test listeners: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server
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
		AnalyticsURL:     backendURL,
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
