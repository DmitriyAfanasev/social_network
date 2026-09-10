package httpx

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientIP(t *testing.T) {
	for _, tc := range []struct{ remote, want string }{{"127.0.0.1:1234", "127.0.0.1"}, {"unknown", "unknown"}, {"", "unknown"}} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = tc.remote
		require.Equal(t, tc.want, ClientIP(r))
	}
}

func TestRequestIDPreservesHeaderAndAddsContext(t *testing.T) {
	for _, supplied := range []string{"client-id", ""} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Request-ID", supplied)
		var got string
		h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = RequestIDFromContext(r.Context()) }))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, r)
		require.NotEmpty(t, rr.Header().Get("X-Request-ID"))
		if supplied != "" {
			require.Equal(t, supplied, got)
		}
	}
}

func TestRecoveryAndWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	var logs strings.Builder
	h := Recovery(slog.New(slog.NewTextHandler(&logs, nil)))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Contains(t, rr.Body.String(), `"internal_error"`)
}
