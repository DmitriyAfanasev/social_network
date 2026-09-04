// Package httptransport содержит HTTP transport social-сервиса.
package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/social/internal/application"
	"general-project/social/internal/ports"
)

// Handler содержит HTTP-обработчики social-сервиса.
type Handler struct {
	readiness ports.ReadinessChecker
	social    *application.SocialService
}

// NewHandler создаёт HTTP-обработчик social-сервиса.
func NewHandler(readiness ports.ReadinessChecker, social *application.SocialService) *Handler {
	return &Handler{readiness: readiness, social: social}
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
