package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"

	"general-project/libs/platform/httpx"
	"general-project/media/internal/application"
	"general-project/media/internal/ports"
)

type mediaResponse struct {
	ID               string `json:"id"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type,omitempty"`
	MediaType        string `json:"media_type"`
	Size             int64  `json:"size"`
	Checksum         string `json:"checksum"`
	UploadedBy       string `json:"uploaded_by"`
	CreatedAt        string `json:"created_at"`
}

func mapMediaResponse(media application.MediaDTO) mediaResponse {
	return mediaResponse{
		ID:               media.ID.String(),
		OriginalFilename: media.OriginalFilename,
		ContentType:      media.ContentType,
		MediaType:        media.MediaType,
		Size:             media.Size,
		Checksum:         media.Checksum,
		UploadedBy:       media.UploadedBy.String(),
		CreatedAt:        media.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
	}
}
func writeMedia(w http.ResponseWriter, status int, media application.MediaDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapMediaResponse(media))
}
func writeMediaError(w http.ResponseWriter, err error) {
	status, code, message := mediaErrorStatus(err)
	httpx.WriteError(w, status, code, message)
}
func mediaErrorStatus(err error) (int, string, string) {
	switch {
	case errors.Is(err, application.ErrValidation):
		return http.StatusUnprocessableEntity, "validation_error", "файл не соответствует требованиям"
	case errors.Is(err, application.ErrForbidden), errors.Is(err, ports.ErrForbidden):
		return http.StatusForbidden, "forbidden", "нет доступа к медиаобъекту"
	case errors.Is(err, ports.ErrNotFound):
		return http.StatusNotFound, "media_not_found", "медиаобъект не найден"
	case errors.Is(err, ports.ErrAlreadyExists):
		return http.StatusConflict, "media_conflict", "медиаобъект уже существует"
	default:
		return http.StatusInternalServerError, "internal_error", "внутренняя ошибка сервера"
	}
}
