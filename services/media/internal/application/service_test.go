package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/media/internal/domain"
	"general-project/media/internal/ports"
)

type fakeMediaRepository struct {
	media       map[uuid.UUID]domain.Media
	createError error
	deleted     bool
	outboxEvent ports.OutboxEvent
}

func (f *fakeMediaRepository) Create(_ context.Context, media domain.Media) (domain.Media, error) {
	if f.createError != nil {
		return domain.Media{}, f.createError
	}
	media.CreatedAt = time.Now().UTC()
	f.media[media.ID] = media
	return media, nil
}

func (f *fakeMediaRepository) CreateWithOutbox(ctx context.Context, media domain.Media, event ports.OutboxEvent) (domain.Media, error) {
	f.outboxEvent = event
	return f.Create(ctx, media)
}

func (f *fakeMediaRepository) FindByID(_ context.Context, mediaID uuid.UUID) (domain.Media, error) {
	media, ok := f.media[mediaID]
	if !ok || media.DeletedAt != nil {
		return domain.Media{}, ports.ErrNotFound
	}
	return media, nil
}

func (f *fakeMediaRepository) UpdateProcessedImage(_ context.Context, mediaID uuid.UUID, contentType string, size int64, checksum string, width int, height int) error {
	media, ok := f.media[mediaID]
	if !ok || media.DeletedAt != nil {
		return ports.ErrNotFound
	}
	media.ContentType = contentType
	media.Size = size
	media.Checksum = checksum
	media.Width = &width
	media.Height = &height
	f.media[mediaID] = media
	return nil
}

func (f *fakeMediaRepository) Delete(_ context.Context, mediaID uuid.UUID, deletedAt time.Time) error {
	media, ok := f.media[mediaID]
	if !ok {
		return ports.ErrNotFound
	}
	media.DeletedAt = &deletedAt
	f.media[mediaID] = media
	f.deleted = true
	return nil
}

type fakeObjectStorage struct {
	objects     map[string][]byte
	putError    error
	deletedKeys []string
}

func (f *fakeObjectStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	value, ok := f.objects[key]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(value)), nil
}

func (f *fakeObjectStorage) GetRange(_ context.Context, key string, start int64, end int64) (io.ReadCloser, error) {
	value, ok := f.objects[key]
	if !ok {
		return nil, ports.ErrNotFound
	}
	if start < 0 || end < start || end >= int64(len(value)) {
		return nil, ports.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(value[start : end+1])), nil
}

type fakeMediaOutbox struct{}

func (fakeMediaOutbox) Claim(context.Context, int, time.Time) ([]ports.OutboxEvent, error) {
	return nil, nil
}

func (fakeMediaOutbox) MarkPublished(context.Context, uuid.UUID, time.Time) error { return nil }

func (fakeMediaOutbox) MarkFailed(context.Context, uuid.UUID, time.Time, string) error { return nil }

type fakeVideoRepository struct {
	videos      map[uuid.UUID]domain.VideoAsset
	renditions  []domain.VideoRendition
	completed   bool
	deleted     bool
	albumOwner  bool
	listTab     string
	outboxEvent *ports.OutboxEvent
}

func (f *fakeVideoRepository) Create(_ context.Context, video domain.VideoAsset) (domain.VideoAsset, error) {
	video.CreatedAt = time.Now().UTC()
	video.UpdatedAt = video.CreatedAt
	f.videos[video.ID] = video
	return video, nil
}

func (f *fakeVideoRepository) FindByID(_ context.Context, videoID uuid.UUID) (domain.VideoAsset, error) {
	video, ok := f.videos[videoID]
	if !ok {
		return domain.VideoAsset{}, ports.ErrNotFound
	}
	return video, nil
}

func (f *fakeVideoRepository) ListRenditions(_ context.Context, _ uuid.UUID) ([]domain.VideoRendition, error) {
	return f.renditions, nil
}

func (f *fakeVideoRepository) FindDetails(_ context.Context, videoID uuid.UUID, _ *uuid.UUID) (ports.VideoDetails, error) {
	video, err := f.FindByID(context.Background(), videoID)
	if err != nil {
		return ports.VideoDetails{}, err
	}
	return ports.VideoDetails{VideoListItem: ports.VideoListItem{ID: video.ID, MediaID: video.MediaID, AlbumID: video.AlbumID, Title: video.Title, Status: video.Status, CreatedAt: video.CreatedAt, UpdatedAt: video.UpdatedAt}, Renditions: f.renditions}, nil
}

func (f *fakeVideoRepository) ListAlbums(_ context.Context, _ uuid.UUID, _ uuid.UUID, tab string, _ string) ([]ports.VideoAlbumView, error) {
	f.listTab = tab
	return nil, nil
}

func (f *fakeVideoRepository) CreateAlbum(_ context.Context, album domain.VideoAlbum) (domain.VideoAlbum, error) {
	return album, nil
}

func (f *fakeVideoRepository) AlbumBelongsToUser(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.albumOwner, nil
}

func (f *fakeVideoRepository) DeleteAlbum(context.Context, uuid.UUID, uuid.UUID) ([]ports.VideoDeletion, error) {
	return nil, nil
}

func (f *fakeVideoRepository) Delete(_ context.Context, videoID uuid.UUID) error {
	if _, ok := f.videos[videoID]; !ok {
		return ports.ErrNotFound
	}
	delete(f.videos, videoID)
	f.deleted = true
	return nil
}

func (f *fakeVideoRepository) MarkProcessing(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (f *fakeVideoRepository) Complete(_ context.Context, videoID uuid.UUID, status string, duration float64, renditions []domain.VideoRendition, updatedAt time.Time) error {
	video, ok := f.videos[videoID]
	if !ok {
		return ports.ErrNotFound
	}
	video.Status = status
	video.Duration = &duration
	video.UpdatedAt = updatedAt
	f.videos[videoID] = video
	f.renditions = renditions
	f.completed = true
	return nil
}

func (f *fakeVideoRepository) RecordView(context.Context, uuid.UUID, *uuid.UUID) (int, error) {
	return 0, nil
}

func (f *fakeVideoRepository) RecordViewWithOutbox(ctx context.Context, videoID uuid.UUID, userID *uuid.UUID, event ports.OutboxEvent) (int, error) {
	f.outboxEvent = &event
	return f.RecordView(ctx, videoID, userID)
}

func (f *fakeVideoRepository) ToggleLike(context.Context, uuid.UUID, uuid.UUID) (ports.InteractionResult, error) {
	return ports.InteractionResult{}, nil
}

func (f *fakeVideoRepository) SetBookmark(context.Context, uuid.UUID, uuid.UUID, bool) (bool, error) {
	return false, nil
}

func (f *fakeVideoRepository) SetFavorite(context.Context, uuid.UUID, uuid.UUID, bool) (bool, error) {
	return false, nil
}

type fakeTranscodePublisher struct {
	request ports.TranscodeRequest
}

type fakeMusicRepository struct {
	tracks map[uuid.UUID]domain.MusicTrack
	saved  map[uuid.UUID]map[uuid.UUID]bool
}

func (f *fakeMusicRepository) Create(_ context.Context, track domain.MusicTrack) (domain.MusicTrack, error) {
	track.CreatedAt = time.Now().UTC()
	f.tracks[track.ID] = track
	return track, nil
}

func (f *fakeMusicRepository) ListByUser(_ context.Context, userID uuid.UUID) ([]domain.MusicTrack, error) {
	tracks := make([]domain.MusicTrack, 0)
	for _, track := range f.tracks {
		if track.UserID == userID {
			tracks = append(tracks, track)
			continue
		}
		if f.saved[userID][track.ID] {
			track.Saved = true
			tracks = append(tracks, track)
		}
	}
	return tracks, nil
}

func (f *fakeMusicRepository) ListByOwner(_ context.Context, ownerID uuid.UUID, viewerID uuid.UUID) ([]domain.MusicTrack, error) {
	tracks := make([]domain.MusicTrack, 0)
	for _, track := range f.tracks {
		if track.UserID != ownerID {
			continue
		}
		track.Saved = f.saved[viewerID][track.ID]
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func (f *fakeMusicRepository) FindByID(_ context.Context, trackID uuid.UUID) (domain.MusicTrack, error) {
	track, ok := f.tracks[trackID]
	if !ok {
		return domain.MusicTrack{}, ports.ErrNotFound
	}
	return track, nil
}

func (f *fakeMusicRepository) AddToLibrary(_ context.Context, userID uuid.UUID, trackID uuid.UUID) error {
	if f.saved[userID] == nil {
		f.saved[userID] = make(map[uuid.UUID]bool)
	}
	f.saved[userID][trackID] = true
	return nil
}

func (f *fakeMusicRepository) RemoveFromLibrary(_ context.Context, userID uuid.UUID, trackID uuid.UUID) error {
	delete(f.saved[userID], trackID)
	return nil
}

func (f *fakeMusicRepository) Delete(_ context.Context, trackID uuid.UUID) error {
	if _, ok := f.tracks[trackID]; !ok {
		return ports.ErrNotFound
	}
	delete(f.tracks, trackID)
	return nil
}

func (f *fakeTranscodePublisher) Publish(_ context.Context, request ports.TranscodeRequest) error {
	f.request = request
	return nil
}

type fakeMediaCache struct {
	values map[string][]byte
}

func (f *fakeMediaCache) Get(_ context.Context, key string) ([]byte, error) {
	return f.values[key], nil
}

func (f *fakeMediaCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.values[key] = value
	return nil
}

func (f *fakeMediaCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func (f *fakeObjectStorage) Put(_ context.Context, key string, _ string, content io.Reader, _ int64) error {
	if f.putError != nil {
		return f.putError
	}
	value, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	f.objects[key] = value
	return nil
}

func (f *fakeObjectStorage) Delete(_ context.Context, key string) error {
	delete(f.objects, key)
	f.deletedKeys = append(f.deletedKeys, key)
	return nil
}

func TestMediaServiceUploadsAndMapsMetadata(t *testing.T) {
	t.Parallel()

	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}
	storage := &fakeObjectStorage{objects: map[string][]byte{}}
	service := NewMediaService(repository, nil, storage, "general-project")

	media, err := service.Upload(context.Background(), uuid.New(), UploadInput{
		Filename:    "../photo.png",
		ContentType: "image/png",
		Content:     []byte("image-data"),
	})

	require.NoError(t, err)
	require.Equal(t, "photo.png", media.OriginalFilename)
	require.Equal(t, "image", media.MediaType)
	require.NotEmpty(t, media.Checksum)
	require.Len(t, storage.objects, 1)
}

func TestMediaServiceCleansObjectWhenMetadataCreateFails(t *testing.T) {
	t.Parallel()

	storage := &fakeObjectStorage{objects: map[string][]byte{}}
	service := NewMediaService(&fakeMediaRepository{
		media:       map[uuid.UUID]domain.Media{},
		createError: errors.New("database failed"),
	}, nil, storage, "general-project")

	_, err := service.Upload(context.Background(), uuid.New(), UploadInput{Filename: "file.txt", Content: []byte("data")})

	require.EqualError(t, err, "database failed")
	require.Empty(t, storage.objects)
	require.Len(t, storage.deletedKeys, 1)
}

func TestMediaServiceWritesOutboxEvent(t *testing.T) {
	t.Parallel()

	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}
	service := NewMediaService(repository, nil, &fakeObjectStorage{objects: map[string][]byte{}}, "general-project", fakeMediaOutbox{})
	userID := uuid.New()

	media, err := service.Upload(context.Background(), userID, UploadInput{Filename: "photo.png", ContentType: "image/png", Content: []byte("image-data")})

	require.NoError(t, err)
	require.Equal(t, "media.media.created", repository.outboxEvent.EventType)
	require.Equal(t, media.ID, *repository.outboxEvent.AggregateID)
	require.JSONEq(t, `{"media_id":"`+media.ID.String()+`","uploaded_by":"`+userID.String()+`","media_type":"image","size":10}`, string(repository.outboxEvent.Payload))
}

func TestMediaServiceRejectsInvalidUpload(t *testing.T) {
	t.Parallel()

	storage := &fakeObjectStorage{objects: map[string][]byte{}}
	service := NewMediaService(&fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}, nil, storage, "general-project")

	_, err := service.Upload(context.Background(), uuid.New(), UploadInput{Filename: "../", Content: []byte("data")})

	require.ErrorIs(t, err, ErrValidation)
}

func TestMediaServiceRejectsDeleteByAnotherUser(t *testing.T) {
	t.Parallel()

	mediaID := uuid.New()
	ownerID := uuid.New()
	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{
		mediaID: {ID: mediaID, UploadedBy: ownerID, ObjectKey: "objects/file"},
	}}
	service := NewMediaService(repository, nil, &fakeObjectStorage{objects: map[string][]byte{}}, "general-project")

	err := service.Delete(context.Background(), uuid.New(), mediaID)

	require.ErrorIs(t, err, ErrForbidden)
	require.False(t, repository.deleted)
}

func TestMediaServiceGetByIDUsesCache(t *testing.T) {
	t.Parallel()

	mediaID := uuid.New()
	cached := MediaDTO{ID: mediaID, OriginalFilename: "cached.txt", MediaType: "document"}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)

	service := NewMediaService(&fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}, &fakeMediaCache{values: map[string][]byte{mediaCacheKey(mediaID): payload}}, &fakeObjectStorage{objects: map[string][]byte{}}, "general-project")

	result, err := service.GetByID(context.Background(), mediaID)

	require.NoError(t, err)
	require.Equal(t, cached.OriginalFilename, result.OriginalFilename)
}

func TestMediaServiceOpensContentFromStorage(t *testing.T) {
	t.Parallel()

	mediaID := uuid.New()
	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{
		mediaID: {ID: mediaID, ObjectKey: "objects/file", ContentType: "image/png", Size: 4},
	}}
	storage := &fakeObjectStorage{objects: map[string][]byte{"objects/file": []byte("data")}}
	service := NewMediaService(repository, nil, storage, "general-project")

	content, media, err := service.OpenContent(context.Background(), mediaID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = content.Close() })

	value, err := io.ReadAll(content)
	require.NoError(t, err)
	require.Equal(t, []byte("data"), value)
	require.Equal(t, "image/png", media.ContentType)
}

func TestMediaServiceOpensContentRangeFromStorage(t *testing.T) {
	t.Parallel()

	mediaID := uuid.New()
	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{
		mediaID: {ID: mediaID, ObjectKey: "objects/file", ContentType: "audio/mpeg", Size: 6},
	}}
	service := NewMediaService(repository, nil, &fakeObjectStorage{objects: map[string][]byte{"objects/file": []byte("stream")}}, "general-project")

	content, media, err := service.OpenContentRange(context.Background(), mediaID, 1, 3)
	require.NoError(t, err)
	t.Cleanup(func() { _ = content.Close() })

	value, err := io.ReadAll(content)
	require.NoError(t, err)
	require.Equal(t, []byte("tre"), value)
	require.Equal(t, int64(6), media.Size)
}

func TestVideoServiceCreatesAssetAndPublishesTranscodeRequest(t *testing.T) {
	t.Parallel()

	mediaRepository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}
	storage := &fakeObjectStorage{objects: map[string][]byte{}}
	mediaService := NewMediaService(mediaRepository, nil, storage, "general-project")
	videoRepository := &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}}
	publisher := &fakeTranscodePublisher{}
	service := NewVideoService(mediaService, videoRepository, publisher, nil)

	video, err := service.Create(context.Background(), uuid.New(), CreateVideoInput{
		Title:            "Демо",
		File:             UploadInput{Filename: "demo.mp4", ContentType: "video/mp4", Content: []byte("video")},
		RequestedHeights: []int{360, 720},
	})

	require.NoError(t, err)
	require.Equal(t, "Демо", video.Title)
	require.Equal(t, domain.VideoStatusProcessing, video.Status)
	require.Equal(t, video.ID, publisher.request.VideoID)
	require.Equal(t, video.MediaID, publisher.request.MediaID)
	require.Equal(t, []int{360, 720}, publisher.request.RequestedHeights)
}

func TestVideoServiceRejectsVideoForAnotherUsersAlbum(t *testing.T) {
	t.Parallel()

	mediaService := NewMediaService(&fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}, nil, &fakeObjectStorage{objects: map[string][]byte{}}, "general-project")
	videoRepository := &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}, albumOwner: false}
	service := NewVideoService(mediaService, videoRepository, &fakeTranscodePublisher{}, nil)
	albumID := uuid.New()

	_, err := service.Create(context.Background(), uuid.New(), CreateVideoInput{
		AlbumID:          &albumID,
		File:             UploadInput{Filename: "demo.mp4", ContentType: "video/mp4", Content: []byte("video")},
		RequestedHeights: []int{360},
	})

	require.ErrorIs(t, err, ports.ErrNotFound)
	require.Empty(t, mediaService.media.(*fakeMediaRepository).media)
}

func TestVideoServiceUsesUploadedTabByDefault(t *testing.T) {
	t.Parallel()

	repository := &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}}
	service := &VideoService{videos: repository}

	_, err := service.ListAlbums(context.Background(), uuid.New(), uuid.New(), "", "")

	require.NoError(t, err)
	require.Equal(t, "uploaded", repository.listTab)
}

func TestVideoServiceMapsWorkerCompletionToRenditions(t *testing.T) {
	t.Parallel()

	videoID := uuid.New()
	videoRepository := &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{
		videoID: {ID: videoID, Status: domain.VideoStatusProcessing},
	}}
	service := &VideoService{videos: videoRepository}

	err := service.Complete(context.Background(), ports.TranscodeCompletion{
		VideoID:    videoID,
		Status:     domain.VideoStatusReady,
		Duration:   12.5,
		Renditions: []ports.TranscodeRendition{{Height: 720, ObjectKey: "renditions/720p.mp4", ContentType: "video/mp4", Size: 100, Duration: 12.5}},
	})

	require.NoError(t, err)
	require.True(t, videoRepository.completed)
	require.Len(t, videoRepository.renditions, 1)
	require.Equal(t, 720, videoRepository.renditions[0].Height)
}

func TestVideoServiceGetByIDUsesCache(t *testing.T) {
	t.Parallel()

	videoID := uuid.New()
	cached := VideoDTO{ID: videoID, MediaID: uuid.New(), Title: "из cache", Status: domain.VideoStatusReady}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)

	service := &VideoService{
		videos: &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}},
		cache:  &fakeMediaCache{values: map[string][]byte{videoCacheKey(videoID): payload}},
	}

	result, err := service.GetByID(context.Background(), videoID)

	require.NoError(t, err)
	require.Equal(t, cached.Title, result.Title)
}

func TestVideoServiceUsesPlaybackEventForProgressTelemetry(t *testing.T) {
	t.Parallel()

	videoRepository := &fakeVideoRepository{videos: map[uuid.UUID]domain.VideoAsset{}}
	service := NewVideoService(nil, videoRepository, nil, nil, fakeMediaOutbox{})
	videoID := uuid.New()
	userID := uuid.New()

	_, err := service.RecordViewWithMetrics(context.Background(), videoID, &userID, RecordViewInput{
		SessionID: "session-1", WatchSeconds: 12, ProgressSeconds: 12, DurationSeconds: 24,
	})

	require.NoError(t, err)
	require.NotNil(t, videoRepository.outboxEvent)
	require.Equal(t, "media.video.playback", videoRepository.outboxEvent.EventType)
}

func TestMusicServiceRejectsNonAudioUpload(t *testing.T) {
	t.Parallel()

	service := NewMusicService(&MediaService{}, &fakeMusicRepository{tracks: map[uuid.UUID]domain.MusicTrack{}}, nil)

	_, err := service.Create(context.Background(), uuid.New(), CreateMusicInput{File: UploadInput{Filename: "image.png", ContentType: "image/png", Content: []byte("data")}})

	require.ErrorIs(t, err, ErrValidation)
}

func TestMusicServiceCreatesTrackAndInvalidatesUserCache(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	mediaRepository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{}}
	mediaCache := &fakeMediaCache{values: map[string][]byte{musicCacheKey(userID): []byte(`stale`)}}
	mediaService := NewMediaService(mediaRepository, mediaCache, &fakeObjectStorage{objects: map[string][]byte{}}, "general-project")
	musicRepository := &fakeMusicRepository{tracks: map[uuid.UUID]domain.MusicTrack{}}
	service := NewMusicService(mediaService, musicRepository, mediaCache)

	track, err := service.Create(context.Background(), userID, CreateMusicInput{Title: "  Трек  ", Artist: "  Автор ", File: UploadInput{Filename: "song.mp3", ContentType: "audio/mpeg", Content: []byte("audio")}})

	require.NoError(t, err)
	require.Equal(t, "Трек", track.Title)
	require.Equal(t, "Автор", track.Artist)
	require.Equal(t, userID, track.UserID)
	require.Nil(t, mediaCache.values[musicCacheKey(userID)])
}

func TestMusicServiceAddsVisibleTrackToLibrary(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	viewerID := uuid.New()
	trackID := uuid.New()
	musicRepository := &fakeMusicRepository{
		tracks: map[uuid.UUID]domain.MusicTrack{
			trackID: {ID: trackID, UserID: ownerID, Title: "Чужой трек"},
		},
		saved: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	mediaCache := &fakeMediaCache{values: map[string][]byte{musicCacheKey(viewerID): []byte(`stale`)}}
	service := NewMusicService(nil, musicRepository, mediaCache)

	err := service.AddToLibrary(context.Background(), viewerID, trackID)

	require.NoError(t, err)
	require.True(t, musicRepository.saved[viewerID][trackID])
	require.Nil(t, mediaCache.values[musicCacheKey(viewerID)])

	tracks, err := service.ListMine(context.Background(), viewerID)
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	require.True(t, tracks[0].IsSaved)
}
