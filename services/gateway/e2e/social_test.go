package e2e

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSocialFriendRequestAndPrivacy(t *testing.T) {
	// Arrange
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	mailpitURL := strings.TrimRight(os.Getenv("E2E_MAILPIT_URL"), "/")
	if baseURL == "" || mailpitURL == "" {
		t.Skip("E2E_BASE_URL и E2E_MAILPIT_URL не заданы")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	friendID, friendAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	targetID, targetAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	_, strangerAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	waitForProfile(t, client, baseURL, targetID, targetAccessToken)

	status, privacyBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/profiles/me/privacy", targetAccessToken, nil)
	require.Equal(t, http.StatusOK, status, "privacy response: %#v", privacyBody)
	privacyBody["friends_visibility"] = "friends"

	// Act
	status, updatedPrivacy := authorizedJSONRequest(t, client, http.MethodPut, baseURL+"/v1/profiles/me/privacy", targetAccessToken, privacyBody)

	// Assert
	require.Equal(t, http.StatusOK, status, "updated privacy response: %#v", updatedPrivacy)
	require.Equal(t, "friends", updatedPrivacy["friends_visibility"])

	// Act
	status, requestBody := authorizedJSONRequest(t, client, http.MethodPut, baseURL+"/v1/social/friendships/"+targetID, friendAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "friend request response: %#v", requestBody)
	require.Equal(t, "pending", requestBody["state"])
	requestID := requireString(t, requestBody, "request_id")

	// Act
	status, incomingBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/social/friend-requests?direction=incoming", targetAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "incoming requests response: %#v", incomingBody)
	incoming, ok := incomingBody["incoming"].([]any)
	require.True(t, ok)
	require.Len(t, incoming, 1)
	incomingRequest, ok := incoming[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, requestID, incomingRequest["id"])
	require.Equal(t, friendID, incomingRequest["sender_id"])
	require.Equal(t, targetID, incomingRequest["recipient_id"])

	// Act
	status, acceptedBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/social/friend-requests/"+requestID+"/accept", targetAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "accept response: %#v", acceptedBody)
	require.Equal(t, requestID, acceptedBody["request_id"])
	require.Equal(t, "accepted", acceptedBody["state"])

	// Act
	status, relationshipBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/social/relationships/"+targetID, friendAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "relationship response: %#v", relationshipBody)
	require.Equal(t, true, relationshipBody["is_friend"])

	// Act
	status, strangerFriendsBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/social/relationships/"+targetID+"/friends", strangerAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "stranger friends response: %#v", strangerFriendsBody)
	require.Equal(t, false, strangerFriendsBody["visible"])
	require.Empty(t, strangerFriendsBody["friends"])

	// Act
	status, friendFriendsBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/social/relationships/"+targetID+"/friends", friendAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "friend friends response: %#v", friendFriendsBody)
	require.Equal(t, true, friendFriendsBody["visible"])
	friends, ok := friendFriendsBody["friends"].([]any)
	require.True(t, ok)
	require.Contains(t, friends, friendID)

	// Act
	status, _ = authorizedJSONRequest(t, client, http.MethodDelete, baseURL+"/v1/social/friendships/"+targetID, friendAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status)
}
