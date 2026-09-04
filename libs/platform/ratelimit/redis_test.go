package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAllowRejectsInvalidConfigurationBeforeUsingRedis(t *testing.T) {
	limiter := &RedisFixedWindow{}

	allowed, err := limiter.Allow(context.Background(), "test", 0, time.Minute)

	require.False(t, allowed)
	require.Error(t, err)
}

func TestMiddlewareReturnsUnavailableForInvalidConfiguration(t *testing.T) {
	limiter := &RedisFixedWindow{}
	handler := Middleware(limiter, 0, time.Minute, func(*http.Request) string { return "test" })(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not be called for invalid rate-limit configuration")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
