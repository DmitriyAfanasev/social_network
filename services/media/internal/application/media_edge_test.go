package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

func TestVideoServiceRejectsInvalidInputBoundaries(t *testing.T) {
	t.Parallel()

	service := &VideoService{}
	tests := []struct {
		name  string
		input CreateVideoInput
	}{
		{name: "not video", input: CreateVideoInput{File: UploadInput{ContentType: "image/png"}, RequestedHeights: []int{360}}},
		{name: "no heights", input: CreateVideoInput{File: UploadInput{ContentType: "video/mp4"}}},
		{name: "duplicate heights", input: CreateVideoInput{File: UploadInput{ContentType: "video/mp4"}, RequestedHeights: []int{360, 360}}},
		{name: "height too small", input: CreateVideoInput{File: UploadInput{ContentType: "video/mp4"}, RequestedHeights: []int{143}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := service.Create(context.Background(), uuid.New(), tt.input)

			require.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestVideoServiceValidatesAlbumTabAndCompletion(t *testing.T) {
	t.Parallel()

	repository := &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}}
	service := &VideoService{videos: repository}

	_, err := service.ListAlbums(context.Background(), uuid.New(), uuid.New(), "unknown", "")
	require.ErrorIs(t, err, ErrValidation)

	err = service.Complete(context.Background(), ports.TranscodeCompletion{VideoID: uuid.New(), Status: "processing"})
	require.ErrorIs(t, err, ErrValidation)

	err = service.Complete(context.Background(), ports.TranscodeCompletion{
		VideoID: uuid.New(), Status: domain.VideoStatusReady,
		Renditions: []ports.TranscodeRendition{{Height: 0, ObjectKey: "rendition.mp4", Size: 1}},
	})
	require.ErrorIs(t, err, ErrValidation)
}

func TestVideoServiceRejectsInvalidPlaybackMetrics(t *testing.T) {
	t.Parallel()

	service := &VideoService{videos: &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}}}

	_, err := service.RecordViewWithMetrics(context.Background(), uuid.New(), nil, RecordViewInput{SessionID: string(make([]rune, 129))})

	require.ErrorIs(t, err, ErrValidation)
}

type fakeMusicVisibilityReader struct {
	allowed bool
	err     error
}

func (f fakeMusicVisibilityReader) CanViewMusic(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.allowed, f.err
}

func TestMusicServiceAppliesVisibilityPolicy(t *testing.T) {
	t.Parallel()

	ownerID, viewerID, trackID := uuid.New(), uuid.New(), uuid.New()
	repository := &fakeMusicRepository{
		tracks: map[uuid.UUID]domain.MusicTrack{trackID: {ID: trackID, UserID: ownerID, Title: "track"}},
		saved:  make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	service := NewMusicService(nil, repository, nil, fakeMusicVisibilityReader{allowed: false})

	_, err := service.ListForViewer(context.Background(), ownerID, viewerID)
	require.ErrorIs(t, err, ErrForbidden)

	err = service.AddToLibrary(context.Background(), viewerID, trackID)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestMusicServiceDoesNotSaveOwnTrackAndListsForOwner(t *testing.T) {
	t.Parallel()

	userID, trackID := uuid.New(), uuid.New()
	repository := &fakeMusicRepository{
		tracks: map[uuid.UUID]domain.MusicTrack{trackID: {ID: trackID, UserID: userID, Title: "own"}},
		saved:  make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	service := NewMusicService(nil, repository, nil)

	err := service.AddToLibrary(context.Background(), userID, trackID)
	require.NoError(t, err)
	require.Empty(t, repository.saved)

	tracks, err := service.ListForViewer(context.Background(), userID, userID)
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	require.Equal(t, "own", tracks[0].Title)
}
