package httptransport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"uuid"

	"general-project/identity/internal/application"
	"github.com/stretchr/testify/require"
)

type fakeReadinessChecker struct{}

func (fakeReadinessChecker) Check(_ context.Context) error { return nil }

func TestHealth(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	NewHandler(nil, nil).Health(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
}

func TestReady(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	NewHandler(fakeReadinessChecker{}, nil).Ready(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"status":"ready"}`, recorder.Body.String())
}

func TestWriteAuthUsesGeneratedResponseModel(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("aa695c9d-1c93-4c64-a396-2b8504193359")
	recorder := httptest.NewRecorder()

	writeAuth(recorder, http.StatusOK, application.AuthDTO{
		User: application.UserDTO{
			ID:        userID,
			Email:     "user@example.com",
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		},
		AccessToken:      "access-token",
		TokenType:        "Bearer",
		ExpiresIn:        900,
		RefreshToken:     "refresh-token",
		RefreshExpiresIn: 86_400,
	})

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
		"user": {
			"id": "aa695c9d-1c93-4c64-a396-2b8504193359",
			"email": "user@example.com",
			"status": "active",
			"created_at": "2026-09-09T12:00:00Z",
			"updated_at": "2026-09-09T12:00:00Z"
		},
		"access_token": "access-token",
		"token_type": "Bearer",
		"expires_in": 900,
		"refresh_token": "refresh-token",
		"refresh_expires_in": 86400
	}`, recorder.Body.String())
}
