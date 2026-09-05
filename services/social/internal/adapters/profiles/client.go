// Package profiles содержит HTTP-адаптер social к profiles-сервису.
package profiles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uuid"

	"general-project/social/internal/ports"
)

// Client читает политики профилей через profiles API.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient создаёт HTTP-клиент profiles-сервиса.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 2 * time.Second}}
}

// GetFriendRequestPolicy возвращает политику входящих заявок целевого профиля.
func (c *Client) GetFriendRequestPolicy(ctx context.Context, targetID uuid.UUID) (string, error) {
	if c.baseURL == "" {
		return "everyone", nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/profiles/%s", c.baseURL, targetID), nil)
	if err != nil {
		return "", err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("profile policy request returned status %d", response.StatusCode)
	}
	var payload struct {
		Privacy struct {
			FriendRequestPolicy string `json:"friend_request_policy"`
		} `json:"privacy"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Privacy.FriendRequestPolicy == "" {
		return "everyone", nil
	}
	return payload.Privacy.FriendRequestPolicy, nil
}

var _ ports.ProfilePolicyReader = (*Client)(nil)
