package httptransport

import (
	"net/http"
	"uuid"

	"general-project/content/internal/application"
	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type postRequest = PostRequest

// Create создаёт текстовый пост от имени текущего пользователя.









func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	var request postRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	mediaIDs, err := parseMediaIDs(request.MediaIDs)
	if err != nil {
		writePostError(w, application.ErrValidation)
		return
	}
	post, err := h.content.CreatePost(r.Context(), userID, application.CreatePostInput{Body: request.Body, MediaIDs: mediaIDs})
	if err != nil {
		writePostError(w, err)
		return
	}
	writePost(w, http.StatusCreated, post)
}

// Update изменяет текст поста текущего пользователя.












func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	postID, ok := parsePostID(w, r)
	if !ok {
		return
	}
	var request postRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	mediaIDs, err := parseMediaIDs(request.MediaIDs)
	if err != nil {
		writePostError(w, application.ErrValidation)
		return
	}
	post, err := h.content.UpdatePost(r.Context(), userID, postID, application.UpdatePostInput{Body: request.Body, MediaIDs: mediaIDs})
	if err != nil {
		writePostError(w, err)
		return
	}
	writePost(w, http.StatusOK, post)
}

// Delete удаляет пост текущего пользователя.








func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	postID, ok := parsePostID(w, r)
	if !ok {
		return
	}
	if err := h.content.DeletePost(r.Context(), userID, postID); err != nil {
		writePostError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authenticatedUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return uuid.Nil(), false
	}
	return userID, true
}

func parsePostID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_post_id", "некорректный UUID поста")
		return uuid.Nil(), false
	}
	return postID, true
}

func parseMediaIDs(values *[]string) ([]uuid.UUID, error) {
	if values == nil {
		return nil, nil
	}
	mediaIDs := make([]uuid.UUID, 0, len(*values))
	for _, value := range *values {
		mediaID, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		mediaIDs = append(mediaIDs, mediaID)
	}
	return mediaIDs, nil
}
