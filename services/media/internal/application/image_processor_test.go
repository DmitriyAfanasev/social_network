package application

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/media/internal/domain"
)

func TestImageProcessorConvertsSmallerImageToWebP(t *testing.T) {
	mediaID := uuid.New()
	imageData := solidPNG(t, 128, 128)
	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{
		mediaID: {ID: mediaID, ObjectKey: "objects/photo.png", MediaType: "image", ContentType: "image/png", Size: int64(len(imageData))},
	}}
	storage := &fakeObjectStorage{objects: map[string][]byte{"objects/photo.png": imageData}}
	processor := NewImageProcessor(repository, nil, storage, ImageProcessingLimits{MaxInputBytes: 1024 * 1024, MaxPixels: 1_000_000, MemoryBudgetBytes: 128 * 1024 * 1024, Quality: 80})

	require.NoError(t, processor.Process(context.Background(), mediaID))
	processed := repository.media[mediaID]
	require.Equal(t, webPContentType, processed.ContentType)
	require.Less(t, processed.Size, int64(len(imageData)))
	require.NotEmpty(t, processed.Checksum)
	require.Equal(t, 128, *processed.Width)
	require.Equal(t, 128, *processed.Height)
	require.Equal(t, []byte("RIFF"), storage.objects["objects/photo.png"][:4])
}

func TestImageProcessorRejectsMemoryRiskBeforeDecoding(t *testing.T) {
	processor := NewImageProcessor(nil, nil, nil, ImageProcessingLimits{MaxPixels: 12_000_000, MemoryBudgetBytes: 256 * 1024 * 1024})

	err := processor.validateImageConfig(image.Config{Width: 10_000, Height: 10_000})
	require.ErrorIs(t, err, ErrImageDimensionsTooLarge)
}

func TestImageProcessorSkipsWebPAlreadyStored(t *testing.T) {
	mediaID := uuid.New()
	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{
		mediaID: {ID: mediaID, ObjectKey: "objects/photo.webp", MediaType: "image", ContentType: "image/webp", Size: 42},
	}}
	processor := NewImageProcessor(repository, nil, &fakeObjectStorage{}, ImageProcessingLimits{})

	require.NoError(t, processor.Process(context.Background(), mediaID))
}

func solidPNG(t *testing.T, width int, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: 30, G: 90, B: 180, A: 255})
		}
	}
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, img))
	return data.Bytes()
}
