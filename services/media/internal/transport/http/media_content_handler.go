package httptransport

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/libs/platform/httpx"
	"general-project/media/internal/application"
)

// StreamContent отдаёт бинарное содержимое активного медиаобъекта.
// @Summary Скачать или воспроизвести медиа
// @Tags media
// @Produce application/octet-stream
// @Param mediaID path string true "UUID медиаобъекта"
// @Success 200 {file} binary
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/media/{mediaID}/content [get]
func (h *Handler) StreamContent(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_media_id", "некорректный UUID медиаобъекта")
		return
	}
	if rangeValue := r.Header.Get("Range"); rangeValue != "" {
		media, getErr := h.media.GetByID(r.Context(), mediaID)
		if getErr != nil {
			writeMediaError(w, getErr)
			return
		}
		start, end, ok := parseByteRange(rangeValue, media.Size)
		if !ok {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", media.Size))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		content, _, openErr := h.media.OpenContentRange(r.Context(), mediaID, start, end)
		if openErr != nil {
			writeMediaError(w, openErr)
			return
		}
		defer content.Close()
		writeContentHeaders(w, media)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, media.Size))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.CopyN(w, content, end-start+1)
		return
	}
	content, media, err := h.media.OpenContent(r.Context(), mediaID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	defer content.Close()
	writeContentHeaders(w, media)
	_, _ = io.Copy(w, content)
}

func writeContentHeaders(w http.ResponseWriter, media application.MediaDTO) {
	contentType := media.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, strings.ReplaceAll(media.OriginalFilename, `"`, "")))
	if media.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(media.Size, 10))
	}
	if media.Size > 0 {
		w.Header().Set("Accept-Ranges", "bytes")
	}
}

func parseByteRange(value string, size int64) (int64, int64, bool) {
	if size <= 0 || !strings.HasPrefix(value, "bytes=") {
		return 0, 0, false
	}
	value = strings.TrimSpace(strings.TrimPrefix(value, "bytes="))
	if strings.Contains(value, ",") {
		return 0, 0, false
	}
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	left, right := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if left == "" {
		suffix, err := strconv.ParseInt(right, 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false
		}
		if suffix > size {
			suffix = size
		}
		return size - suffix, size - 1, true
	}
	start, err := strconv.ParseInt(left, 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, false
	}
	end := size - 1
	if right != "" {
		end, err = strconv.ParseInt(right, 10, 64)
		if err != nil || end < start {
			return 0, 0, false
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end, true
}
