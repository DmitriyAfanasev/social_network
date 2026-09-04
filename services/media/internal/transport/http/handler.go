// Package httptransport содержит HTTP transport media-сервиса.
package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/media/internal/application"
	"general-project/media/internal/ports"
)

// Handler содержит HTTP-обработчики media-сервиса.
type Handler struct {
	readiness ports.ReadinessChecker
	media     *application.MediaService
	video     *application.VideoService
	music     *application.MusicService
}

// NewHandler создаёт HTTP-обработчик media-сервиса.
func NewHandler(readiness ports.ReadinessChecker, media *application.MediaService, video *application.VideoService, music *application.MusicService) *Handler {
	return &Handler{readiness: readiness, media: media, video: video, music: music}
}

// Health возвращает состояние процесса без проверки внешних зависимостей.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

// Ready проверяет доступность PostgreSQL перед приёмом трафика.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.readiness.Check(r.Context()); err != nil {
		writeStatus(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	writeStatus(w, http.StatusOK, "ready")
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": value})
}
