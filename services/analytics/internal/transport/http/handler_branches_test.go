package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"general-project/analytics/internal/application"
	"general-project/analytics/internal/domain"
	"general-project/libs/platform/auth"
)

type fakeVideoAnalyticsRepository struct {
	videoStats       domain.VideoStats
	videoStatsError  error
	viewerStats      []domain.ViewerVideoStats
	viewerStatsError error
}

func (f *fakeVideoAnalyticsRepository) GetVideoStats(context.Context, uuid.UUID) (domain.VideoStats, error) {
	return f.videoStats, f.videoStatsError
}

func (f *fakeVideoAnalyticsRepository) ListViewerVideoStats(context.Context, uuid.UUID, int) ([]domain.ViewerVideoStats, error) {
	return f.viewerStats, f.viewerStatsError
}

type fakeAnalyticsReadiness struct{ err error }

func (f fakeAnalyticsReadiness) Check(context.Context) error { return f.err }

type fakeAnalyticsVerifier struct{ userID uuid.UUID }

func (f fakeAnalyticsVerifier) UserID(string) (uuid.UUID, error) { return f.userID, nil }

func TestHandlerHealthAndReadyBranches(t *testing.T) {
	tests := []struct {
		name       string
		readiness  fakeAnalyticsReadiness
		wantStatus int
		wantValue  string
	}{
		{name: "ready", wantStatus: http.StatusOK, wantValue: "ready"},
		{name: "database unavailable", readiness: fakeAnalyticsReadiness{err: errors.New("database down")}, wantStatus: http.StatusServiceUnavailable, wantValue: "database_unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewHandler(application.NewVideoAnalyticsService(&fakeVideoAnalyticsRepository{}), tt.readiness)
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			response := httptest.NewRecorder()

			// Act
			handler.Ready(response, request)

			// Assert
			require.Equal(t, tt.wantStatus, response.Code)
			var body StatusResponse
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, tt.wantValue, string(body.Status))
		})
	}

	// Arrange
	handler := NewHandler(application.NewVideoAnalyticsService(&fakeVideoAnalyticsRepository{}), fakeAnalyticsReadiness{})
	response := httptest.NewRecorder()

	// Act
	handler.Health(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
}

func TestHandlerGetVideoStatsBranches(t *testing.T) {
	videoID := uuid.New()
	tests := []struct {
		name       string
		path       string
		repository fakeVideoAnalyticsRepository
		wantStatus int
		wantCode   string
	}{
		{name: "invalid id", path: "/videos/not-uuid", wantStatus: http.StatusBadRequest, wantCode: "invalid_video_id"},
		{name: "repository error", path: "/videos/" + videoID.String(), repository: fakeVideoAnalyticsRepository{videoStatsError: errors.New("query failed")}, wantStatus: http.StatusInternalServerError, wantCode: "analytics_unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler := NewHandler(application.NewVideoAnalyticsService(&tt.repository), fakeAnalyticsReadiness{})
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			request = withVideoIDParam(request, pathVideoID(request.URL.Path))
			response := httptest.NewRecorder()

			// Act
			handler.GetVideoStats(response, request)

			// Assert
			require.Equal(t, tt.wantStatus, response.Code)
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, tt.wantCode, body["code"])
		})
	}

	// Arrange
	handler := NewHandler(application.NewVideoAnalyticsService(&fakeVideoAnalyticsRepository{videoStats: domain.VideoStats{VideoID: videoID, Views: 12}}), fakeAnalyticsReadiness{})
	request := httptest.NewRequest(http.MethodGet, "/videos/"+videoID.String(), nil)
	request = withVideoIDParam(request, videoID.String())
	response := httptest.NewRecorder()

	// Act
	handler.GetVideoStats(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	var body VideoStatsResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, videoID.String(), body.VideoId)
	require.Equal(t, int64(12), body.Views)
}

func TestHandlerListMyVideoStatsBranches(t *testing.T) {
	userID := uuid.New()
	videoID := uuid.New()
	repository := &fakeVideoAnalyticsRepository{viewerStats: []domain.ViewerVideoStats{{VideoID: videoID, Views: 3}}}
	handler := NewHandler(application.NewVideoAnalyticsService(repository), fakeAnalyticsReadiness{})
	withAuth := auth.Middleware(fakeAnalyticsVerifier{userID: userID})(http.HandlerFunc(handler.ListMyVideoStats))

	// Arrange
	unauthorized := httptest.NewRecorder()

	// Act
	handler.ListMyVideoStats(unauthorized, httptest.NewRequest(http.MethodGet, "/stats", nil))

	// Assert
	require.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	// Arrange
	invalidLimit := httptest.NewRequest(http.MethodGet, "/stats?limit=nope", nil)
	invalidLimit.Header.Set("Authorization", "Bearer token")
	invalidLimitResponse := httptest.NewRecorder()

	// Act
	withAuth.ServeHTTP(invalidLimitResponse, invalidLimit)

	// Assert
	require.Equal(t, http.StatusBadRequest, invalidLimitResponse.Code)

	// Arrange
	request := httptest.NewRequest(http.MethodGet, "/stats?limit=10", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	// Act
	withAuth.ServeHTTP(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	var body ViewerVideoStatsListResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Len(t, body.Videos, 1)
	require.Equal(t, videoID.String(), body.Videos[0].VideoId)
}

func pathVideoID(path string) string {
	for index := len(path) - 1; index >= 0; index-- {
		if path[index] == '/' {
			return path[index+1:]
		}
	}
	return path
}

func withVideoIDParam(request *http.Request, videoID string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("videoID", videoID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}
