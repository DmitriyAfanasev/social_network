package domain

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestMediaOwnership(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	require.True(t, (Media{UploadedBy: owner}).CanBeManagedBy(owner))
	require.False(t, (Media{UploadedBy: owner}).CanBeManagedBy(other))
	require.True(t, (MusicTrack{UserID: owner}).CanBeManagedBy(owner))
	require.False(t, (MusicTrack{UserID: owner}).CanBeManagedBy(other))
}
