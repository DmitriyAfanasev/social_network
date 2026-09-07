// Package media проверяет метаданные фотографий через HTTP-контракт media.
package media

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"uuid"
)

// Client читает владельца и тип медиаобъекта.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient создаёт клиент с ограниченным временем ожидания.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 3 * time.Second}}
}

// CanAttachPhoto проверяет, что активный файл является изображением текущего пользователя.
func (c *Client) CanAttachPhoto(ctx context.Context, userID, mediaID uuid.UUID) (bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/media/"+mediaID.String(), nil)
	if err != nil {
		return false, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("media metadata status %d", response.StatusCode)
	}
	var payload struct {
		UploadedBy string `json:"uploaded_by"`
		MediaType  string `json:"media_type"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.UploadedBy == userID.String() && payload.MediaType == "image", nil
}
