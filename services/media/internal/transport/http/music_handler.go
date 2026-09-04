package httptransport

import (
	"net/http"
	"uuid"

	"general-project/libs/platform/httpx"
	"general-project/media/internal/application"
	"github.com/go-chi/chi/v5"
)

const maxMusicUploadSize int64 = 15 * 1024 * 1024

// ListMusic возвращает музыкальные треки текущего пользователя.
// @Summary Получить свою музыку
// @Tags media
// @Produce json
// @Success 200 {object} musicListResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/media/music [get]
func (h *Handler) ListMusic(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	tracks, err := h.music.ListMine(r.Context(), userID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeMusicList(w, http.StatusOK, tracks)
}

// CreateMusic загружает аудиофайл и создаёт музыкальный трек.
// @Summary Загрузить музыкальный трек
// @Tags media
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Аудиофайл"
// @Param title formData string false "Название"
// @Param artist formData string false "Исполнитель"
// @Success 201 {object} musicResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/media/music [post]
func (h *Handler) CreateMusic(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	limitMultipartBody(w, r, maxMusicUploadSize)
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
	content, err := readUpload(file, maxMusicUploadSize)
	if err != nil {
		writeMediaError(w, application.ErrValidation)
		return
	}
	track, err := h.music.Create(r.Context(), userID, application.CreateMusicInput{
		Title:  r.FormValue("title"),
		Artist: r.FormValue("artist"),
		File: application.UploadInput{
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			Content:     content,
		},
	})
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeMusic(w, http.StatusCreated, track)
}

// DeleteMusic удаляет музыкальный трек текущего пользователя.
// @Summary Удалить музыкальный трек
// @Tags media
// @Param trackID path string true "UUID трека"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/media/music/{trackID} [delete]
func (h *Handler) DeleteMusic(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	trackID, err := uuid.Parse(chi.URLParam(r, "trackID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_track_id", "некорректный UUID трека")
		return
	}
	if err := h.music.Delete(r.Context(), userID, trackID); err != nil {
		writeMediaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
