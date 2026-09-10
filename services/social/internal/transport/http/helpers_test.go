package httptransport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"general-project/social/internal/application"

	"github.com/stretchr/testify/require"
)

func TestSocialErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"validation", application.ErrValidation, 422, "validation_error"},
		{"blocked", application.ErrBlocked, 409, "interaction_blocked"},
		{"not found", application.ErrFriendRequestNotFound, 404, "friend_request_not_found"},
		{"forbidden", application.ErrFriendRequestForbidden, 403, "friend_request_forbidden"},
		{"state", application.ErrFriendRequestState, 409, "friend_request_state_conflict"},
		{"unknown", errors.New("boom"), 500, "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeSocialError(rr, tc.err)
			require.Equal(t, tc.status, rr.Code)
			require.Contains(t, rr.Body.String(), tc.code)
		})
	}
}

func TestAuthenticatedUserRejectsMissingContext(t *testing.T) {
	rr := httptest.NewRecorder()
	user, ok := authenticatedUser(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, ok)
	require.Empty(t, user)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
