package httptransport

import (
	"testing"
	"time"

	"uuid"

	"general-project/profiles/internal/application"
	"github.com/stretchr/testify/require"
)

func TestMapProfileResponseUsesGeneratedModel(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("a2cd0ddd-9b2d-49aa-ad61-85de2bf83a64")
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	response := mapProfileResponse(application.ProfileDTO{
		UserID: userID, Bio: "bio", AvatarURL: "https://example.com/avatar.jpg", CreatedAt: now, UpdatedAt: now,
		Details: application.ProfileDetailsDTO{FirstName: "Иван"},
		Privacy: application.ProfilePrivacyDTO{ProfileVisibility: "public"},
	})

	require.Equal(t, userID.String(), response.UserID)
	require.Equal(t, "Иван", response.FirstName)
	require.Equal(t, "public", response.Privacy.ProfileVisibility)
	require.Equal(t, "2026-09-09T12:00:00Z", response.CreatedAt)
}
