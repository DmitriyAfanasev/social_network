// Package social содержит HTTP-адаптер profiles к social-сервису.
package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/profiles/internal/ports"
)

// Client получает признаки социальных отношений через HTTP API social-сервиса.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient создаёт HTTP-клиент social-сервиса.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 2 * time.Second}}
}

// GetRelationship проверяет доступ viewer к целевому профилю.
func (c *Client) GetRelationship(ctx context.Context, viewerID uuid.UUID, targetID uuid.UUID) (ports.RelationshipAccess, error) {
	if viewerID == uuid.Nil() || viewerID == targetID {
		return ports.RelationshipAccess{}, nil
	}
	token, ok := auth.AccessTokenFromContext(ctx)
	if !ok || c.baseURL == "" {
		return ports.RelationshipAccess{}, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/social/relationships/%s", c.baseURL, targetID), nil)
	if err != nil {
		return ports.RelationshipAccess{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.client.Do(request)
	if err != nil {
		return ports.RelationshipAccess{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ports.RelationshipAccess{}, fmt.Errorf("social relationship request returned status %d", response.StatusCode)
	}
	var payload struct {
		IsFriend         bool `json:"is_friend"`
		IsFriendOfFriend bool `json:"is_friend_of_friend"`
		IsBlocked        bool `json:"is_blocked"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ports.RelationshipAccess{}, err
	}
	return ports.RelationshipAccess{IsFriend: payload.IsFriend, IsFriendOfFriend: payload.IsFriendOfFriend, IsBlocked: payload.IsBlocked}, nil
}
