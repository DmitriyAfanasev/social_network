package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestAuthLifecycle(t *testing.T) {
	// Arrange
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	mailpitURL := strings.TrimRight(os.Getenv("E2E_MAILPIT_URL"), "/")
	if baseURL == "" || mailpitURL == "" {
		t.Skip("E2E_BASE_URL и E2E_MAILPIT_URL не заданы")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	email := fmt.Sprintf("e2e-%s@example.test", uuid.New().String())
	password := "correct horse battery staple"

	// Act
	status, body := jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/register", map[string]string{
		"email": email, "password": password,
	})

	// Assert
	require.Equal(t, http.StatusAccepted, status)
	require.Equal(t, "confirmation email sent", body["message"])

	// Act
	confirmationToken := waitForConfirmationToken(t, client, mailpitURL, email)
	status, body = jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/confirm-registration", map[string]string{
		"token": confirmationToken,
	})

	// Assert
	require.Equal(t, http.StatusOK, status, "confirmation response: %#v", body)
	confirmedRefresh := requireString(t, body, "refresh_token")
	require.NotEmpty(t, requireString(t, body, "access_token"))

	// Act
	status, body = jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/login", map[string]string{
		"email": email, "password": password,
	})

	// Assert
	require.Equal(t, http.StatusOK, status)
	loginRefresh := requireString(t, body, "refresh_token")
	require.NotEqual(t, confirmedRefresh, loginRefresh)

	// Act
	status, body = jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/refresh", map[string]string{
		"refresh_token": loginRefresh,
	})

	// Assert
	require.Equal(t, http.StatusOK, status)
	rotatedRefresh := requireString(t, body, "refresh_token")
	require.NotEqual(t, loginRefresh, rotatedRefresh)

	// Act
	status, _ = jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/logout", map[string]string{
		"refresh_token": rotatedRefresh,
	})

	// Assert
	require.Equal(t, http.StatusNoContent, status)
	status, _ = jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/refresh", map[string]string{
		"refresh_token": rotatedRefresh,
	})
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestMessagingLifecycle(t *testing.T) {
	// Arrange
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	mailpitURL := strings.TrimRight(os.Getenv("E2E_MAILPIT_URL"), "/")
	if baseURL == "" || mailpitURL == "" {
		t.Skip("E2E_BASE_URL и E2E_MAILPIT_URL не заданы")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	_, firstAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	secondUserID, secondAccessToken := registerAndConfirm(t, client, baseURL, mailpitURL)
	waitForProfile(t, client, baseURL, secondUserID, secondAccessToken)

	// Act
	status, conversationBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/messaging/conversations/direct", firstAccessToken, map[string]string{
		"other_user_id": secondUserID,
	})

	// Assert
	require.Equal(t, http.StatusOK, status, "conversation response: %#v", conversationBody)
	conversationID := requireString(t, conversationBody, "id")
	participantIDs, ok := conversationBody["participant_ids"].([]any)
	require.True(t, ok)
	require.Len(t, participantIDs, 2)

	// Act
	status, messageBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/messaging/conversations/"+conversationID+"/messages", firstAccessToken, map[string]string{
		"body": "hello from e2e",
	})

	// Assert
	require.Equal(t, http.StatusCreated, status)
	messageID := requireString(t, messageBody, "id")
	require.Equal(t, "hello from e2e", messageBody["body"])

	// Act
	status, messageBody = authorizedJSONRequest(t, client, http.MethodPatch, baseURL+"/v1/messaging/messages/"+messageID, firstAccessToken, map[string]string{
		"body": "edited from e2e",
	})

	// Assert
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "edited from e2e", messageBody["body"])

	// Act
	status, _ = authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/messaging/conversations/"+conversationID+"/messages?limit=20", secondAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status)

	// Act
	status, _ = authorizedJSONRequest(t, client, http.MethodDelete, baseURL+"/v1/messaging/messages/"+messageID, firstAccessToken, nil)

	// Assert
	require.Equal(t, http.StatusNoContent, status)
}

func registerAndConfirm(t *testing.T, client *http.Client, baseURL string, mailpitURL string) (string, string) {
	t.Helper()
	email := fmt.Sprintf("e2e-%s@example.test", uuid.New().String())
	password := "correct horse battery staple"
	status, body := jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/register", map[string]string{
		"email": email, "password": password,
	})
	require.Equal(t, http.StatusAccepted, status)
	confirmationToken := waitForConfirmationToken(t, client, mailpitURL, email)
	status, body = jsonRequest(t, client, http.MethodPost, baseURL+"/v1/auth/confirm-registration", map[string]string{
		"token": confirmationToken,
	})
	require.Equal(t, http.StatusOK, status, "confirmation response: %#v", body)
	user, ok := body["user"].(map[string]any)
	require.True(t, ok)
	return requireString(t, user, "id"), requireString(t, body, "access_token")
}

func waitForProfile(t *testing.T, client *http.Client, baseURL string, userID string, accessToken string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, _ := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/profiles/"+userID, accessToken, nil)
		if status == http.StatusOK {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("profile %s was not available within 30 seconds", userID)
}

func authorizedJSONRequest(t *testing.T, client *http.Client, method string, endpoint string, accessToken string, payload any) (int, map[string]any) {
	t.Helper()
	encoded := []byte(nil)
	if payload != nil {
		var err error
		encoded, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, endpoint, bytes.NewReader(encoded))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if len(bytes.TrimSpace(data)) == 0 {
		return response.StatusCode, nil
	}
	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body), string(data))
	return response.StatusCode, body
}

func jsonRequest(t *testing.T, client *http.Client, method string, endpoint string, payload any) (int, map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(context.Background(), method, endpoint, bytes.NewReader(encoded))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if len(bytes.TrimSpace(data)) == 0 {
		return response.StatusCode, nil
	}
	var body map[string]any
	require.NoError(t, json.Unmarshal(data, &body), string(data))
	return response.StatusCode, body
}

func waitForConfirmationToken(t *testing.T, client *http.Client, mailpitURL string, email string) string {
	t.Helper()
	pattern := regexp.MustCompile(`token=([^&\s"<>]+)`)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, mailpitURL+"/api/v1/messages", nil)
		require.NoError(t, err)
		response, err := client.Do(request)
		if err == nil {
			data, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr == nil && response.StatusCode == http.StatusOK {
				var listing struct {
					Messages []struct {
						ID string `json:"ID"`
						To []struct {
							Address string `json:"Address"`
						} `json:"To"`
					} `json:"messages"`
				}
				if json.Unmarshal(data, &listing) == nil {
					for _, message := range listing.Messages {
						matchesEmail := false
						for _, recipient := range message.To {
							if strings.EqualFold(recipient.Address, email) {
								matchesEmail = true
								break
							}
						}
						if !matchesEmail {
							continue
						}
						token := messageTextToken(t, client, mailpitURL, message.ID, pattern)
						if token != "" {
							return token
						}
					}
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("confirmation email for %s was not received", email)
	return ""
}

func messageTextToken(t *testing.T, client *http.Client, mailpitURL string, messageID string, pattern *regexp.Regexp) string {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, mailpitURL+"/api/v1/message/"+url.PathEscape(messageID), nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK {
		return ""
	}
	var detail struct {
		Text string `json:"Text"`
		HTML string `json:"HTML"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return ""
	}
	messageText := detail.Text + "\n" + detail.HTML
	matches := pattern.FindStringSubmatch(messageText)
	if len(matches) != 2 {
		return ""
	}
	token, err := url.QueryUnescape(string(matches[1]))
	if err != nil {
		return ""
	}
	return token
}

func requireString(t *testing.T, body map[string]any, key string) string {
	t.Helper()
	value, ok := body[key].(string)
	require.Truef(t, ok, "response field %q is missing or not a string", key)
	require.NotEmpty(t, value)
	return value
}
