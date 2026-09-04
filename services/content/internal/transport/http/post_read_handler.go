package httptransport

import (
	"net/http"
	"strconv"
	"uuid"

	"general-project/content/internal/application"
	"general-project/libs/platform/httpx"
	"github.com/go-chi/chi/v5"
)

// GetByID возвращает текстовый пост по UUID.
// @Summary Получить пост
// @Tags content
// @Produce json
// @Param postID path string true "UUID поста"
// @Success 200 {object} postResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/content/posts/{postID} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_post_id", "некорректный UUID поста")
		return
	}
	post, err := h.content.GetPost(r.Context(), postID)
	if err != nil {
		writePostError(w, err)
		return
	}
	writePost(w, http.StatusOK, post)
}

// Feed возвращает последние текстовые посты.
// @Summary Получить ленту постов
// @Tags content
// @Produce json
// @Param limit query int false "Количество постов" default(20)
// @Success 200 {object} postsResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/content/feed [get]
func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writePostError(w, application.ErrValidation)
			return
		}
		limit = parsed
	}
	posts, err := h.content.ListRecent(r.Context(), limit)
	if err != nil {
		writePostError(w, err)
		return
	}
	writePosts(w, http.StatusOK, posts)
}
