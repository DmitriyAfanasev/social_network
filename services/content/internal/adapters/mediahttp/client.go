// Package mediahttp содержит HTTP-адаптер проверки вложений через media-сервис.
package mediahttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uuid"

	"general-project/content/internal/ports"
)

// Client проверяет ownership media-объектов через публичный metadata endpoint.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient создаёт HTTP-клиент media-сервиса.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 3 * time.Second}}
}

// ValidateOwned проверяет, что все вложения существуют и принадлежат пользователю.
func (c *Client) ValidateOwned(ctx context.Context, userID uuid.UUID, mediaIDs []uuid.UUID) error {
	for _, mediaID := range mediaIDs {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/media/"+mediaID.String(), nil)
		if err != nil {
			return err
		}
		response, err := c.http.Do(request)
		if err != nil {
			return err
		}
		var payload mediaResponse
		decodeErr := json.NewDecoder(response.Body).Decode(&payload)
		response.Body.Close()
		if response.StatusCode == http.StatusNotFound {
			return ports.ErrMediaNotFound
		}
		if response.StatusCode != http.StatusOK || decodeErr != nil {
			return fmt.Errorf("media service returned status %d", response.StatusCode)
		}
		ownerID, err := uuid.Parse(payload.UploadedBy)
		if err != nil || ownerID != userID {
			return ports.ErrMediaForbidden
		}
	}
	return nil
}

type mediaResponse struct {
	UploadedBy string `json:"uploaded_by"`
}
