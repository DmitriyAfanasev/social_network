package access

import (
	"context"
	"fmt"
	"general-project/libs/platform/auth"
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestPhotoAccessForwardsVerifiedTokenAndFailsClosed(t *testing.T) {
	status := http.StatusOK
	client := NewClient("https://profiles.test", "")
	client.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer verified-test-token", r.Header.Get("Authorization"))
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"allowed":false}`)), Header: make(http.Header), Request: r}, nil
	})
	ctx := auth.ContextWithAccessToken(context.Background(), "verified-test-token")
	allowed, err := client.CanViewPhoto(ctx, uuid.New())
	require.NoError(t, err)
	require.False(t, allowed)
	status = http.StatusServiceUnavailable
	allowed, err = client.CanViewPhoto(ctx, uuid.New())
	require.Error(t, err)
	require.False(t, allowed)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCanViewMusicPolicies(t *testing.T) {
	owner, viewer := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name    string
		policy  string
		rel     relationshipResponse
		want    bool
		wantErr bool
	}{
		{"owner always allowed", "nobody", relationshipResponse{}, true, false},
		{"nobody", "nobody", relationshipResponse{}, false, false},
		{"everyone", "everyone", relationshipResponse{}, true, false},
		{"friend", "friends", relationshipResponse{IsFriend: true}, true, false},
		{"not friend", "friends", relationshipResponse{}, false, false},
		{"friend of friend", "friends_of_friends", relationshipResponse{IsFriendOfFriend: true}, true, false},
		{"blocked", "friends_of_friends", relationshipResponse{IsFriend: true, IsBlocked: true}, false, false},
		{"unsupported", "secret", relationshipResponse{}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient("https://profiles.test", "https://social.test")
			client.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"privacy":{"music_visibility":"` + tc.policy + `"}}`
				if strings.Contains(r.URL.Host, "social") {
					body = fmt.Sprintf(`{"is_friend":%t,"is_friend_of_friend":%t,"is_blocked":%t}`, tc.rel.IsFriend, tc.rel.IsFriendOfFriend, tc.rel.IsBlocked)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
			})
			ctx := auth.ContextWithAccessToken(context.Background(), "token")
			got, err := client.CanViewMusic(ctx, viewer, owner)
			if tc.name == "owner always allowed" {
				got, err = client.CanViewMusic(ctx, owner, owner)
			}
			require.Equal(t, tc.want, got)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
