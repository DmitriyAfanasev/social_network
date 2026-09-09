package httptransport

import (
	"net/http"
	"strings"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/media/internal/application"
	"github.com/go-chi/chi/v5"
)

const maxMusicUploadSize int64 = 15 * 1024 * 1024

// ListMusic возвращает музыкальные треки владельца с учётом приватности.







func (h *Handler) ListMusic(w http.ResponseWriter, r *http.Request) {
	viewerID, _ := auth.UserIDFromContext(r.Context())
	ownerID := viewerID
	if value := strings.TrimSpace(r.URL.Query().Get("owner_id")); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_owner_id", "некорректный UUID владельца")
			return
		}
		ownerID = parsed
	}
	if ownerID == uuid.Nil() {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен или owner_id")
		return
	}
	tracks, err := h.music.ListForViewer(r.Context(), ownerID, viewerID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeMusicList(w, http.StatusOK, tracks)
}

// CreateMusic загружает аудиофайл и создаёт музыкальный трек.











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

// AddMusicToLibrary добавляет доступный трек в личную аудиотеку пользователя.









func (h *Handler) AddMusicToLibrary(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	trackID, err := uuid.Parse(chi.URLParam(r, "trackID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_track_id", "некорректный UUID трека")
		return
	}
	if err := h.music.AddToLibrary(r.Context(), userID, trackID); err != nil {
		writeMediaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveMusicFromLibrary удаляет чужой трек из личной аудиотеки пользователя.






func (h *Handler) RemoveMusicFromLibrary(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	trackID, err := uuid.Parse(chi.URLParam(r, "trackID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_track_id", "некорректный UUID трека")
		return
	}
	if err := h.music.RemoveFromLibrary(r.Context(), userID, trackID); err != nil {
		writeMediaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
