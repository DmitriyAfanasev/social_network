package httptransport

import (
	"net/http"
	"uuid"

	"general-project/libs/platform/httpx"

	"github.com/go-chi/chi/v5"
)

// GetByID возвращает активные метаданные медиаобъекта.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_media_id", "некорректный UUID медиаобъекта")
		return
	}
	media, err := h.media.GetByID(r.Context(), mediaID)
	if err != nil {
		writeMediaError(w, err)
		return
	}
	writeMedia(w, http.StatusOK, media)
}
