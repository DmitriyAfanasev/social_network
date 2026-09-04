package httptransport

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"general-project/profiles/internal/application"
)

// Search выполняет prefix-поиск публичных профилей по handle.
// @Summary Поиск профилей по handle
// @Tags profiles
// @Produce json
// @Param q query string true "Начало handle"
// @Param limit query int false "Максимум результатов" default(20)
// @Success 200 {object} profilesResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/profiles/search [get]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writeProfileError(w, application.ErrValidation)
			return
		}
		limit = parsed
	}
	profiles, err := h.profiles.SearchByHandle(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	response := profilesResponse{Profiles: make([]profileResponse, 0, len(profiles))}
	for _, profile := range profiles {
		response.Profiles = append(response.Profiles, mapProfileResponse(profile))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// GetByHandle возвращает публичный профиль по его URL-handle.
// @Summary Получить профиль по handle
// @Tags profiles
// @Produce json
// @Param handle path string true "Публичный handle"
// @Success 200 {object} profileResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/profiles/{handle} [get]
func (h *Handler) GetByHandle(w http.ResponseWriter, r *http.Request) {
	profile, err := h.profiles.GetByHandle(r.Context(), chi.URLParam(r, "handle"))
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeProfile(w, http.StatusOK, profile)
}
