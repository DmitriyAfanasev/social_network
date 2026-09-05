// Package access содержит HTTP-проверки политик доступа для messaging.
package access

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/messaging/internal/ports"
)

// Client читает политику сообщений из profiles и социальную близость из social.
type Client struct {
	profilesURL string
	socialURL   string
	client      *http.Client
}

// NewClient создаёт клиент проверки права на начало диалога.
func NewClient(profilesURL string, socialURL string) *Client {
	return &Client{
		profilesURL: strings.TrimRight(profilesURL, "/"),
		socialURL:   strings.TrimRight(socialURL, "/"),
		client:      &http.Client{Timeout: 2 * time.Second},
	}
}

// CanMessage проверяет policy целевого профиля с учётом социальной близости.
func (c *Client) CanMessage(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (bool, error) {
	if actorID == uuid.Nil() || targetID == uuid.Nil() || actorID == targetID {
		return false, nil
	}
	policy, err := c.messagePolicy(ctx, targetID)
	if err != nil {
		return false, err
	}
	switch policy {
	case "nobody":
		return false, nil
	case "everyone", "":
		return true, nil
	case "friends", "friends_of_friends":
		relationship, relationshipErr := c.relationship(ctx, targetID)
		if relationshipErr != nil {
			return false, relationshipErr
		}
		if relationship.IsBlocked {
			return false, nil
		}
		if policy == "friends" {
			return relationship.IsFriend, nil
		}
		return relationship.IsFriend || relationship.IsFriendOfFriend, nil
	default:
		return false, fmt.Errorf("unsupported message policy %q", policy)
	}
}

func (c *Client) messagePolicy(ctx context.Context, targetID uuid.UUID) (string, error) {
	if c.profilesURL == "" {
		return "everyone", nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/profiles/%s", c.profilesURL, targetID), nil)
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
			MessagePolicy string `json:"message_policy"`
		} `json:"privacy"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Privacy.MessagePolicy, nil
}

func (c *Client) relationship(ctx context.Context, targetID uuid.UUID) (relationshipResponse, error) {
	token, ok := auth.AccessTokenFromContext(ctx)
	if !ok || c.socialURL == "" {
		return relationshipResponse{}, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/social/relationships/%s", c.socialURL, targetID), nil)
	if err != nil {
		return relationshipResponse{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.client.Do(request)
	if err != nil {
		return relationshipResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return relationshipResponse{}, fmt.Errorf("social relationship request returned status %d", response.StatusCode)
	}
	var result relationshipResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return relationshipResponse{}, err
	}
	return result, nil
}

type relationshipResponse struct {
	IsFriend         bool `json:"is_friend"`
	IsFriendOfFriend bool `json:"is_friend_of_friend"`
	IsBlocked        bool `json:"is_blocked"`
}

var _ ports.MessagePolicyReader = (*Client)(nil)
