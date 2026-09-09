// Package httptransport содержит HTTP transport profiles-сервиса.
package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/profiles/internal/application"
	"general-project/profiles/internal/ports"
)

// Handler содержит HTTP-обработчики profiles-сервиса.
type Handler struct {
	readiness ports.ReadinessChecker
	profiles  *application.ProfileService
	media     *application.ProfileMediaService
}

// NewHandler создаёт HTTP-обработчик profiles-сервиса.
func NewHandler(readiness ports.ReadinessChecker, profiles *application.ProfileService, media *application.ProfileMediaService) *Handler {
	return &Handler{readiness: readiness, profiles: profiles, media: media}
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
	_ = json.NewEncoder(w).Encode(StatusResponse{Status: StatusResponseStatus(value)})
}
func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
