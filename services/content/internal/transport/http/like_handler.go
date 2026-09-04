package httptransport

import (
	"context"
	"net/http"
	"uuid"

	"general-project/content/internal/application"
)

// Like добавляет лайк текущего пользователя к посту.
// @Summary Поставить лайк посту
// @Tags content
// @Produce json
// @Param postID path string true "UUID поста"
// @Success 200 {object} likeResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/content/posts/{postID}/like [post]
func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	h.handleLike(w, r, h.likes.LikePost)
}

// Unlike убирает лайк текущего пользователя у поста.
// @Summary Убрать лайк с поста
// @Tags content
// @Produce json
// @Param postID path string true "UUID поста"
// @Success 200 {object} likeResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/content/posts/{postID}/like [delete]
func (h *Handler) Unlike(w http.ResponseWriter, r *http.Request) {
	h.handleLike(w, r, h.likes.UnlikePost)
}

func (h *Handler) handleLike(w http.ResponseWriter, r *http.Request, action func(context.Context, uuid.UUID, uuid.UUID) (application.LikeDTO, error)) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	postID, ok := parsePostID(w, r)
	if !ok {
		return
	}
	like, err := action(r.Context(), userID, postID)
	if err != nil {
		writePostError(w, err)
		return
	}
	writeLike(w, http.StatusOK, like)
}
