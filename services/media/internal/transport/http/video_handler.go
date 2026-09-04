package httptransport

import (
	"net/http"
	"strconv"
	"strings"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/media/internal/application"
	"github.com/go-chi/chi/v5"
)

const maxVideoUploadSize int64 = 50 * 1024 * 1024

// CreateVideo загружает видео и ставит задачу транскодирования в Kafka.
// @Summary Загрузить видео
// @Tags media
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Видео"
// @Param title formData string false "Название видео"
// @Param album_id formData string false "UUID альбома"
// @Param heights formData string false "Высоты через запятую"
// @Success 202 {object} videoResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/media/videos [post]
func (h *Handler) CreateVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	limitMultipartBody(w, r, maxVideoUploadSize)
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_multipart", "некорректная multipart-форма")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "file_required", "требуется поле file")
		return
	}
	defer file.Close()
	content, err := readUpload(file, maxVideoUploadSize)
	if err != nil {
		writeMediaError(w, application.ErrValidation)
		return
	}
	heights, err := parseHeights(r.FormValue("heights"))
	if err != nil {
		writeMediaError(w, application.ErrValidation)
		return
	}
	var albumID *uuid.UUID
	if value := strings.TrimSpace(r.FormValue("album_id")); value != "" {
		parsed, parseErr := uuid.Parse(value)
		if parseErr != nil {
			writeMediaError(w, application.ErrValidation)
			return
		}
		albumID = &parsed
	}
	video, err := h.video.Create(r.Context(), userID, application.CreateVideoInput{
		Title:            r.FormValue("title"),
		AlbumID:          albumID,
		File:             application.UploadInput{Filename: header.Filename, ContentType: header.Header.Get("Content-Type"), Content: content},
		RequestedHeights: heights,
	})
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeVideo(w, http.StatusAccepted, video)
}

// ListVideos возвращает видео текущего владельца, сгруппированные по альбомам.
// @Summary Получить видео по альбомам
// @Tags media
// @Produce json
// @Param owner_id query string false "UUID владельца"
// @Param tab query string false "uploaded, favorite, viewed или bookmarked"
// @Param q query string false "Поиск по названию"
// @Success 200 {object} videoAlbumListResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/media/videos [get]
func (h *Handler) ListVideos(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	ownerID := viewerID
	if value := strings.TrimSpace(r.URL.Query().Get("owner_id")); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_owner_id", "некорректный UUID владельца")
			return
		}
		ownerID = parsed
	}
	albums, err := h.video.ListAlbums(r.Context(), ownerID, viewerID, r.URL.Query().Get("tab"), r.URL.Query().Get("q"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeVideoAlbums(w, http.StatusOK, albums)
}

// CreateAlbum создаёт альбом видео текущего пользователя.
// @Summary Создать альбом видео
// @Tags media
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param title formData string true "Название альбома"
// @Success 201 {object} videoAlbumResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/media/videos/albums [post]
func (h *Handler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeMediaError(w, application.ErrValidation)
		return
	}
	album, err := h.video.CreateAlbum(r.Context(), userID, r.FormValue("title"))
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeVideoAlbum(w, http.StatusCreated, album.ID, album.Title)
}

// DeleteAlbum удаляет альбом и видео текущего пользователя.
// @Summary Удалить альбом видео
// @Tags media
// @Param albumID path string true "UUID альбома"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/media/videos/albums/{albumID} [delete]
func (h *Handler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	albumID, err := uuid.Parse(chi.URLParam(r, "albumID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_album_id", "некорректный UUID альбома")
		return
	}
	if err := h.video.DeleteAlbum(r.Context(), userID, albumID); err != nil {
		writeMediaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RecordView фиксирует просмотр видео, допускается анонимный просмотр.
// @Summary Зафиксировать просмотр видео
// @Tags media
// @Produce json
// @Param videoID path string true "UUID видео"
// @Success 200 {object} videoViewResponse
// @Router /v1/media/videos/{videoID}/view [post]
func (h *Handler) RecordView(w http.ResponseWriter, r *http.Request) {
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_video_id", "некорректный UUID видео")
		return
	}
	var userID *uuid.UUID
	if current, ok := auth.UserIDFromContext(r.Context()); ok {
		userID = &current
	}
	count, err := h.video.RecordView(r.Context(), videoID, userID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeVideoView(w, http.StatusOK, count)
}

// ToggleLike переключает like текущего пользователя.
// @Summary Поставить или убрать like видео
// @Tags media
// @Produce json
// @Param videoID path string true "UUID видео"
// @Success 200 {object} videoLikeResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/media/videos/{videoID}/like [post]
func (h *Handler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_video_id", "некорректный UUID видео")
		return
	}
	result, err := h.video.ToggleLike(r.Context(), videoID, userID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeVideoLike(w, http.StatusOK, result)
}

// SetBookmark изменяет состояние закладки видео.
// @Summary Изменить закладку видео
// @Tags media
// @Produce json
// @Param videoID path string true "UUID видео"
// @Success 200 {object} videoFlagResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/media/videos/{videoID}/bookmark [post]
// @Router /v1/media/videos/{videoID}/bookmark [delete]
func (h *Handler) SetBookmark(w http.ResponseWriter, r *http.Request) {
	h.setVideoFlag(w, r, true)
}

// SetFavorite изменяет состояние избранного видео.
// @Summary Изменить избранное видео
// @Tags media
// @Produce json
// @Param videoID path string true "UUID видео"
// @Success 200 {object} videoFlagResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/media/videos/{videoID}/favorite [post]
// @Router /v1/media/videos/{videoID}/favorite [delete]
func (h *Handler) SetFavorite(w http.ResponseWriter, r *http.Request) {
	h.setVideoFlag(w, r, false)
}

func (h *Handler) setVideoFlag(w http.ResponseWriter, r *http.Request, bookmark bool) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_video_id", "некорректный UUID видео")
		return
	}
	value := r.Method == http.MethodPost
	if bookmark {
		result, serviceErr := h.video.SetBookmark(r.Context(), videoID, userID, value)
		if serviceErr != nil {
			writeMediaError(w, serviceErr)
			return
		}
		writeVideoBookmark(w, http.StatusOK, result)
		return
	}
	result, serviceErr := h.video.SetFavorite(r.Context(), videoID, userID, value)
	if serviceErr != nil {
		writeMediaError(w, serviceErr)
		return
	}
	writeVideoFavorite(w, http.StatusOK, result)
}

// GetVideo возвращает состояние видео и готовые варианты.
// @Summary Получить видео
// @Tags media
// @Produce json
// @Param videoID path string true "UUID видео"
// @Success 200 {object} videoResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/media/videos/{videoID} [get]
func (h *Handler) GetVideo(w http.ResponseWriter, r *http.Request) {
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_video_id", "некорректный UUID видео")
		return
	}
	var viewerID *uuid.UUID
	if current, ok := auth.UserIDFromContext(r.Context()); ok {
		viewerID = &current
	}
	video, err := h.video.GetByID(r.Context(), videoID, viewerID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeVideo(w, http.StatusOK, video)
}

// DeleteVideo удаляет видео текущего пользователя.
// @Summary Удалить видео
// @Tags media
// @Param videoID path string true "UUID видео"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/media/videos/{videoID} [delete]
func (h *Handler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_video_id", "некорректный UUID видео")
		return
	}
	if err := h.video.Delete(r.Context(), userID, videoID); err != nil {
		writeMediaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseHeights(value string) ([]int, error) {
	if strings.TrimSpace(value) == "" {
		return []int{360, 720}, nil
	}
	parts := strings.Split(value, ",")
	heights := make([]int, 0, len(parts))
	for _, part := range parts {
		height, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		heights = append(heights, height)
	}
	return heights, nil
}
