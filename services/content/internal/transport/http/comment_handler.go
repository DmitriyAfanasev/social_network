package httptransport

import (
	"net/http"
	"strconv"
	"uuid"

	"general-project/content/internal/application"
	"general-project/libs/platform/httpx"
	"github.com/go-chi/chi/v5"
)

// ListComments возвращает комментарии поста в порядке публикации.
// @Summary Получить комментарии поста
// @Tags content
// @Produce json
// @Param postID path string true "UUID поста"
// @Param limit query int false "Количество комментариев" default(50)
// @Success 200 {object} commentsResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/content/posts/{postID}/comments [get]
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	postID, ok := parsePostID(w, r)
	if !ok {
		return
	}
	limit := 50
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writePostError(w, application.ErrValidation)
			return
		}
		limit = parsed
	}
	comments, err := h.comments.ListComments(r.Context(), postID, limit)
	if err != nil {
		writePostError(w, err)
		return
	}
	writeComments(w, http.StatusOK, comments)
}

// CreateComment создаёт комментарий к посту от имени текущего пользователя.
// @Summary Создать комментарий
// @Tags content
// @Accept json
// @Produce json
// @Param postID path string true "UUID поста"
// @Param request body postRequest true "Текст комментария"
// @Success 201 {object} commentResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/content/posts/{postID}/comments [post]
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
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
	comment, err := h.comments.CreateComment(r.Context(), userID, postID, application.CreateCommentInput{Body: request.Body})
	if err != nil {
		writePostError(w, err)
		return
	}
	writeComment(w, http.StatusCreated, comment)
}

// DeleteComment удаляет комментарий текущего пользователя.
// @Summary Удалить комментарий
// @Tags content
// @Param commentID path string true "UUID комментария"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/content/comments/{commentID} [delete]
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	commentID, err := uuid.Parse(chi.URLParam(r, "commentID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_comment_id", "некорректный UUID комментария")
		return
	}
	if err := h.comments.DeleteComment(r.Context(), userID, commentID); err != nil {
		writePostError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
