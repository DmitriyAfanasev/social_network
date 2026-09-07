// Package access содержит HTTP-проверки видимости музыкальных треков.
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
	"general-project/media/internal/ports"
)

// Client читает политику профиля из profiles и близость из social.
type Client struct {
	profilesURL string
	socialURL   string
	client      *http.Client
}

// NewClient создаёт клиент проверки видимости музыки.
func NewClient(profilesURL string, socialURL string) *Client {
	return &Client{profilesURL: strings.TrimRight(profilesURL, "/"), socialURL: strings.TrimRight(socialURL, "/"), client: &http.Client{Timeout: 2 * time.Second}}
}

// CanViewMusic проверяет, доступна ли музыка владельца текущему пользователю.
func (c *Client) CanViewMusic(ctx context.Context, viewerID uuid.UUID, ownerID uuid.UUID) (bool, error) {
	if viewerID != uuid.Nil() && viewerID == ownerID {
		return true, nil
	}
	policy, err := c.musicPolicy(ctx, ownerID)
	if err != nil {
		return false, err
	}
	switch policy {
	case "nobody":
		return false, nil
	case "everyone", "":
		return true, nil
	case "friends", "friends_of_friends":
		relationship, relationshipErr := c.relationship(ctx, ownerID)
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
		return false, fmt.Errorf("unsupported music policy %q", policy)
	}
}

func (c *Client) musicPolicy(ctx context.Context, ownerID uuid.UUID) (string, error) {
	if c.profilesURL == "" {
		return "everyone", nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/profiles/%s", c.profilesURL, ownerID), nil)
	if err != nil {
		return "", err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("profile privacy request returned status %d", response.StatusCode)
	}
	var payload struct {
		Privacy struct {
			MusicVisibility string `json:"music_visibility"`
		} `json:"privacy"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Privacy.MusicVisibility, nil
}

func (c *Client) relationship(ctx context.Context, ownerID uuid.UUID) (relationshipResponse, error) {
	token, ok := auth.AccessTokenFromContext(ctx)
	if !ok || c.socialURL == "" {
		return relationshipResponse{}, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/social/relationships/%s", c.socialURL, ownerID), nil)
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

var _ ports.MusicVisibilityReader = (*Client)(nil)

// CanViewPhoto проверяет ограничения файла фотографии, передавая проверенный токен зрителя.
func (c *Client) CanViewPhoto(ctx context.Context, mediaID uuid.UUID) (bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.profilesURL+"/v1/profiles/photo-media/"+mediaID.String()+"/visibility", nil)
	if err != nil {
		return false, err
	}
	if token, ok := auth.AccessTokenFromContext(ctx); ok {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("photo visibility status %d", response.StatusCode)
	}
	var payload struct {
		Allowed bool `json:"allowed"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.Allowed, nil
}
