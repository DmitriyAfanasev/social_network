package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentLifecycle(t *testing.T) {
	// Arrange
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	mailpitURL := strings.TrimRight(os.Getenv("E2E_MAILPIT_URL"), "/")
	if baseURL == "" || mailpitURL == "" {
		t.Skip("E2E_BASE_URL и E2E_MAILPIT_URL не заданы")
	}
	client := &http.Client{}
	_, accessToken := registerAndConfirm(t, client, baseURL, mailpitURL)

	// Act
	status, mediaBody := multipartUploadRequest(t, client, baseURL+"/v1/media", accessToken, "photo.txt", "text/plain", []byte("content e2e media"))

	// Assert
	require.Equal(t, http.StatusCreated, status, "media response: %#v", mediaBody)
	mediaID := requireString(t, mediaBody, "id")
	require.Equal(t, "photo.txt", mediaBody["original_filename"])
	require.Equal(t, float64(len("content e2e media")), mediaBody["size"])

	// Act
	status, postBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/content/posts", accessToken, map[string]any{
		"body":      "content e2e post",
		"media_ids": []string{mediaID},
	})

	// Assert
	require.Equal(t, http.StatusCreated, status, "post response: %#v", postBody)
	postID := requireString(t, postBody, "id")
	require.Equal(t, "content e2e post", postBody["body"])
	require.Equal(t, []any{mediaID}, postBody["media_ids"])

	// Act
	status, feedBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/content/feed?limit=20", accessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "feed response: %#v", feedBody)
	feedPosts, ok := feedBody["posts"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, feedPosts)

	// Act
	status, likeBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/content/posts/"+postID+"/like", accessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "like response: %#v", likeBody)
	require.Equal(t, true, likeBody["liked"])
	require.Equal(t, postID, likeBody["post_id"])

	// Act
	status, commentBody := authorizedJSONRequest(t, client, http.MethodPost, baseURL+"/v1/content/posts/"+postID+"/comments", accessToken, map[string]string{
		"body": "content e2e comment",
	})

	// Assert
	require.Equal(t, http.StatusCreated, status, "comment response: %#v", commentBody)
	commentID := requireString(t, commentBody, "id")
	require.Equal(t, "content e2e comment", commentBody["body"])

	// Act
	status, commentsBody := authorizedJSONRequest(t, client, http.MethodGet, baseURL+"/v1/content/posts/"+postID+"/comments?limit=20", accessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "comments response: %#v", commentsBody)
	comments, ok := commentsBody["comments"].([]any)
	require.True(t, ok)
	require.Len(t, comments, 1)

	// Act
	status, unlikeBody := authorizedJSONRequest(t, client, http.MethodDelete, baseURL+"/v1/content/posts/"+postID+"/like", accessToken, nil)

	// Assert
	require.Equal(t, http.StatusOK, status, "unlike response: %#v", unlikeBody)
	require.Equal(t, false, unlikeBody["liked"])

	// Act
	status, _ = authorizedJSONRequest(t, client, http.MethodDelete, baseURL+"/v1/content/comments/"+commentID, accessToken, nil)

	// Assert
	require.Equal(t, http.StatusNoContent, status)

	// Act
	status, _ = authorizedJSONRequest(t, client, http.MethodDelete, baseURL+"/v1/content/posts/"+postID, accessToken, nil)

	// Assert
	require.Equal(t, http.StatusNoContent, status)
}

func multipartUploadRequest(t *testing.T, client *http.Client, endpoint string, accessToken string, filename string, contentType string, content []byte) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, &body)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if len(bytes.TrimSpace(data)) == 0 {
		return response.StatusCode, nil
	}
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result), string(data))
	return response.StatusCode, result
}
