// Package httptransport содержит HTTP transport analytics-сервиса.
package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/analytics/internal/application"
	"general-project/analytics/internal/domain"
	"general-project/analytics/internal/ports"
	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
)

// Handler содержит HTTP-обработчики аналитики видео.
type Handler struct {
	video     *application.VideoAnalyticsService
	readiness ports.ReadinessChecker
}

// NewHandler создаёт HTTP-обработчик аналитики.
func NewHandler(video *application.VideoAnalyticsService, readiness ports.ReadinessChecker) *Handler {
	return &Handler{video: video, readiness: readiness}
}

// Health возвращает состояние процесса без проверки внешних зависимостей.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

// Ready проверяет, что analytics-сервис может обратиться к PostgreSQL.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.readiness.Check(r.Context()); err != nil {
		writeStatus(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	writeStatus(w, http.StatusOK, "ready")
}

// GetVideoStats возвращает агрегированную статистику конкретного видео.
// @Summary Получить аналитику видео
// @Tags analytics
// @Produce json
// @Param videoID path string true "UUID видео"
// @Success 200 {object} videoStatsResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/analytics/videos/{videoID} [get]
func (h *Handler) GetVideoStats(w http.ResponseWriter, r *http.Request) {
	videoID, err := uuid.Parse(chi.URLParam(r, "videoID"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_video_id", "некорректный UUID видео")
		return
	}
	stats, err := h.video.GetVideoStats(r.Context(), videoID)
	if err != nil {
		writeAnalyticsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapVideoStats(stats))
}

// ListMyVideoStats возвращает историю видео, просмотренных текущим пользователем.
// @Summary Получить историю просмотров видео
// @Tags analytics
// @Produce json
// @Param limit query int false "Количество видео, от 1 до 100" default(20)
// @Success 200 {object} viewerVideoStatsListResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/analytics/me/videos [get]
func (h *Handler) ListMyVideoStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "limit должен быть целым числом от 1 до 100")
			return
		}
		limit = parsed
	}
	items, err := h.video.ListViewerVideoStats(r.Context(), userID, limit)
	if err != nil {
		writeAnalyticsError(w, err)
		return
	}
	response := viewerVideoStatsListResponse{Videos: make([]viewerVideoStatsResponse, 0, len(items))}
	for _, item := range items {
		response.Videos = append(response.Videos, mapViewerVideoStats(item))
	}
	writeJSON(w, http.StatusOK, response)
}

type videoStatsResponse struct {
	VideoID             string     `json:"video_id"`
	Views               int64      `json:"views"`
	UniqueViewers       int64      `json:"unique_viewers"`
	TotalWatchSeconds   float64    `json:"total_watch_seconds"`
	AverageWatchSeconds float64    `json:"average_watch_seconds"`
	CompletedViews      int64      `json:"completed_views"`
	LastViewedAt        *time.Time `json:"last_viewed_at,omitempty"`
}

type viewerVideoStatsResponse struct {
	VideoID             string    `json:"video_id"`
	Views               int64     `json:"views"`
	WatchSeconds        float64   `json:"watch_seconds"`
	AverageWatchSeconds float64   `json:"average_watch_seconds"`
	LastViewedAt        time.Time `json:"last_viewed_at"`
}

type viewerVideoStatsListResponse struct {
	Videos []viewerVideoStatsResponse `json:"videos"`
}

func mapVideoStats(stats domain.VideoStats) videoStatsResponse {
	return videoStatsResponse{
		VideoID:             stats.VideoID.String(),
		Views:               stats.Views,
		UniqueViewers:       stats.UniqueViewers,
		TotalWatchSeconds:   stats.TotalWatchSeconds,
		AverageWatchSeconds: stats.AverageWatchSeconds,
		CompletedViews:      stats.CompletedViews,
		LastViewedAt:        stats.LastViewedAt,
	}
}

func mapViewerVideoStats(stats domain.ViewerVideoStats) viewerVideoStatsResponse {
	return viewerVideoStatsResponse{
		VideoID: stats.VideoID.String(), Views: stats.Views, WatchSeconds: stats.WatchSeconds,
		AverageWatchSeconds: stats.AverageWatchSeconds, LastViewedAt: stats.LastViewedAt,
	}
}

func writeAnalyticsError(w http.ResponseWriter, err error) {
	if errors.Is(err, application.ErrVideoAnalyticsValidation) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "некорректные параметры запроса")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "analytics_unavailable", "не удалось получить аналитику")
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	writeJSON(w, status, map[string]string{"status": value})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
