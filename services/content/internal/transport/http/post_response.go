package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"uuid"

	"general-project/content/internal/application"
	"general-project/content/internal/ports"
	"general-project/libs/platform/httpx"
)

type postResponse struct {
	ID            string   `json:"id"`
	AuthorID      string   `json:"author_id"`
	Body          string   `json:"body"`
	MediaIDs      []string `json:"media_ids,omitempty"`
	LikesCount    int      `json:"likes_count"`
	LikedByViewer bool     `json:"is_liked_by_current"`
	LikedUserIDs  []string `json:"liked_user_ids,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type postsResponse struct {
	Posts []postResponse `json:"posts"`
}

func mapPostResponse(post application.PostDTO) postResponse {
	return postResponse{
		ID:         post.ID.String(),
		AuthorID:   post.AuthorID.String(),
		Body:       post.Body,
		MediaIDs:   mediaIDStrings(post.MediaIDs),
		LikesCount: post.LikesCount, LikedByViewer: post.LikedByViewer,
		LikedUserIDs: userIDStrings(post.LikedUserIDs),
		CreatedAt:    post.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt:    post.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
	}
}

func mediaIDStrings(mediaIDs []uuid.UUID) []string {
	result := make([]string, 0, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		result = append(result, mediaID.String())
	}
	return result
}

func userIDStrings(userIDs []uuid.UUID) []string {
	result := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		result = append(result, userID.String())
	}
	return result
}

func writePost(w http.ResponseWriter, status int, post application.PostDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapPostResponse(post))
}

func writePosts(w http.ResponseWriter, status int, posts []application.PostDTO) {
	response := postsResponse{Posts: make([]postResponse, 0, len(posts))}
	for _, post := range posts {
		response.Posts = append(response.Posts, mapPostResponse(post))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func postErrorStatus(err error) (int, string, string) {
	switch {
	case errors.Is(err, application.ErrValidation):
		return http.StatusUnprocessableEntity, "validation_error", "текст поста некорректен"
	case errors.Is(err, application.ErrForbidden):
		return http.StatusForbidden, "forbidden", "пост может изменять только его автор"
	case errors.Is(err, ports.ErrMediaForbidden):
		return http.StatusForbidden, "media_forbidden", "вложение принадлежит другому пользователю"
	case errors.Is(err, ports.ErrMediaNotFound):
		return http.StatusUnprocessableEntity, "media_not_found", "вложение не найдено"
	case errors.Is(err, ports.ErrNotFound):
		return http.StatusNotFound, "post_not_found", "пост не найден"
	default:
		return http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера"
	}
}

func writePostError(w http.ResponseWriter, err error) {
	status, code, message := postErrorStatus(err)
	httpx.WriteError(w, status, code, message)
}
