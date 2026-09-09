// Package httptransport содержит HTTP transport identity-сервиса.
package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/identity/internal/application"
	"general-project/identity/internal/ports"
)

// Handler содержит HTTP-обработчики identity-сервиса.
type Handler struct {
	readiness ports.ReadinessChecker
	auth      application.Authenticator
}

// NewHandler создаёт HTTP-обработчик identity-сервиса.
func NewHandler(readiness ports.ReadinessChecker, auth application.Authenticator) *Handler {
	return &Handler{readiness: readiness, auth: auth}
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
