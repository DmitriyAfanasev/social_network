// Package httptransport содержит HTTP transport content-сервиса.
package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/content/internal/application"
	"general-project/content/internal/ports"
)

// Handler содержит HTTP-обработчики content-сервиса.
type Handler struct {
	readiness ports.ReadinessChecker
	content   *application.ContentService
	comments  *application.CommentService
	likes     *application.LikeService
}

// NewHandler создаёт HTTP-обработчик content-сервиса.
func NewHandler(readiness ports.ReadinessChecker, content *application.ContentService, comments *application.CommentService, likes *application.LikeService) *Handler {
	return &Handler{readiness: readiness, content: content, comments: comments, likes: likes}
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

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
