package httptransport

import (
	"net/http"
	"uuid"

	"general-project/libs/platform/httpx"
	"github.com/go-chi/chi/v5"
)

// GetByID возвращает активные метаданные медиаобъекта.
// @Summary Получить метаданные медиа
// @Tags media
// @Produce json
// @Param mediaID path string true "UUID медиаобъекта"
// @Success 200 {object} mediaResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/media/{mediaID} [get]
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
