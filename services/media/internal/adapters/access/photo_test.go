package access

import (
	"context"
	"fmt"
	"general-project/libs/platform/auth"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"
)

func TestPhotoAccessForwardsVerifiedTokenAndFailsClosed(t *testing.T) {
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer verified-test-token", r.Header.Get("Authorization"))
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, `{"allowed":false}`)
	}))
	defer server.Close()
	client := NewClient(server.URL, "")
	ctx := auth.ContextWithAccessToken(context.Background(), "verified-test-token")
	allowed, err := client.CanViewPhoto(ctx, uuid.New())
	require.NoError(t, err)
	require.False(t, allowed)
	status = http.StatusServiceUnavailable
	allowed, err = client.CanViewPhoto(ctx, uuid.New())
	require.Error(t, err)
	require.False(t, allowed)
}
