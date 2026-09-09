package httptransport

import (
	"io"
	"net/http"
	"uuid"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/media/internal/application"
	"github.com/go-chi/chi/v5"
)

const maxMultipartMemory = 8 << 20
const maxMediaUploadSize int64 = 25 * 1024 * 1024

// Upload загружает файл в object storage и создаёт его метаданные.









func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	limitMultipartBody(w, r, maxMediaUploadSize)
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
	if header.Size > maxMediaUploadSize {
		writeMediaError(w, application.ErrValidation)
		return
	}
	content, err := readUpload(file, maxMediaUploadSize)
	if err != nil {
		writeMediaError(w, application.ErrValidation)
		return
	}
	media, err := h.media.Upload(r.Context(), userID, application.UploadInput{Filename: header.Filename, ContentType: header.Header.Get("Content-Type"), Content: content})
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeMedia(w, http.StatusCreated, media)
}

func limitMultipartBody(w http.ResponseWriter, r *http.Request, fileLimit int64) {
	r.Body = http.MaxBytesReader(w, r.Body, fileLimit+maxMultipartMemory)
}

// Delete удаляет медиаобъект текущего пользователя.








func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_media_id", "некорректный UUID медиаобъекта")
		return
	}
	if err := h.media.Delete(r.Context(), userID, mediaID); err != nil {
		writeMediaError(w, err)
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

func readUpload(file io.Reader, maxSize int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxSize {
		return nil, application.ErrValidation
	}
	return content, nil
}
