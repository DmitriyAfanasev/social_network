package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"runtime"
	"strings"

	"uuid"

	"github.com/deepteams/webp"

	"general-project/media/internal/ports"
)

const (
	webPContentType              = "image/webp"
	imageProcessingOverhead      = 32 * 1024 * 1024
	imageProcessingBytesPerPixel = 16
)

var (
	// ErrImageInputTooLarge означает, что исходный объект превышает лимит worker.
	ErrImageInputTooLarge = errors.New("image input exceeds worker limit")
	// ErrImageDimensionsTooLarge означает, что изображение требует опасный объём памяти.
	ErrImageDimensionsTooLarge = errors.New("image dimensions exceed worker limit")
	// ErrImageMemoryBudgetExceeded означает, что обработка превысит бюджет памяти worker.
	ErrImageMemoryBudgetExceeded = errors.New("image processing exceeds memory budget")
	// ErrImageRejected означает, что файл не является поддерживаемым статичным изображением.
	ErrImageRejected = errors.New("image rejected")
)

// ImageProcessingLimits ограничивает ресурсы одной фоновой обработки изображения.
type ImageProcessingLimits struct {
	MaxInputBytes     int64
	MaxPixels         int64
	MemoryBudgetBytes int64
	Quality           float32
}

// ImageProcessor преобразует изображения в WebP, не загружая исходный файл целиком в память.
type ImageProcessor struct {
	media   ports.MediaRepository
	cache   ports.MediaCache
	storage ports.ObjectStorage
	limits  ImageProcessingLimits
}

// NewImageProcessor создаёт обработчик с явными лимитами ресурсов.
func NewImageProcessor(media ports.MediaRepository, cache ports.MediaCache, storage ports.ObjectStorage, limits ImageProcessingLimits) *ImageProcessor {
	return &ImageProcessor{media: media, cache: cache, storage: storage, limits: limits}
}

// Process преобразует одно активное изображение в WebP и обновляет его метаданные.
func (p *ImageProcessor) Process(ctx context.Context, mediaID uuid.UUID) error {
	metadata, err := p.media.FindByID(ctx, mediaID)
	if errors.Is(err, ports.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if metadata.MediaType != "image" || strings.EqualFold(strings.TrimSpace(strings.SplitN(metadata.ContentType, ";", 2)[0]), webPContentType) {
		return nil
	}
	if metadata.Size > p.limits.MaxInputBytes {
		return fmt.Errorf("%w: %d bytes", ErrImageInputTooLarge, metadata.Size)
	}

	source, err := p.storage.Get(ctx, metadata.ObjectKey)
	if err != nil {
		return err
	}
	defer source.Close()

	inputFile, err := os.CreateTemp("", "media-image-input-*")
	if err != nil {
		return err
	}
	inputPath := inputFile.Name()
	defer os.Remove(inputPath)
	defer inputFile.Close()
	if err := copyWithLimit(inputFile, source, p.limits.MaxInputBytes); err != nil {
		return err
	}
	if _, err := inputFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	config, format, err := image.DecodeConfig(inputFile)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrImageRejected, err)
	}
	// image.Decode обрабатывает только первый GIF-кадр; не заменяем анимацию статичным файлом.
	if format == "gif" {
		return fmt.Errorf("%w: GIF is not supported", ErrImageRejected)
	}
	if err := p.validateImageConfig(config); err != nil {
		return err
	}
	if _, err := inputFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	decoded, _, err := image.Decode(inputFile)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrImageRejected, err)
	}

	outputFile, err := os.CreateTemp("", "media-image-output-*.webp")
	if err != nil {
		return err
	}
	outputPath := outputFile.Name()
	defer os.Remove(outputPath)
	defer outputFile.Close()
	if err := webp.Encode(outputFile, decoded, &webp.EncoderOptions{Quality: p.limits.Quality, Method: 4}); err != nil {
		return err
	}
	outputInfo, err := outputFile.Stat()
	if err != nil {
		return err
	}
	if outputInfo.Size() <= 0 {
		return errors.New("webp encoder produced empty image")
	}
	if outputInfo.Size() >= metadata.Size {
		return nil
	}
	if _, err := outputFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, outputFile); err != nil {
		return err
	}
	if _, err := outputFile.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := p.storage.Put(ctx, metadata.ObjectKey, webPContentType, outputFile, outputInfo.Size()); err != nil {
		return err
	}
	if err := p.media.UpdateProcessedImage(ctx, mediaID, webPContentType, outputInfo.Size(), hex.EncodeToString(digest.Sum(nil)), config.Width, config.Height); err != nil {
		return err
	}
	if p.cache != nil {
		_ = p.cache.Delete(ctx, mediaCacheKey(mediaID))
	}
	return nil
}

func (p *ImageProcessor) validateImageConfig(config image.Config) error {
	if config.Width <= 0 || config.Height <= 0 {
		return ErrImageDimensionsTooLarge
	}
	width := int64(config.Width)
	height := int64(config.Height)
	if width > math.MaxInt64/height {
		return fmt.Errorf("%w: %dx%d", ErrImageDimensionsTooLarge, config.Width, config.Height)
	}
	pixels := width * height
	if pixels <= 0 || pixels > p.limits.MaxPixels {
		return fmt.Errorf("%w: %dx%d", ErrImageDimensionsTooLarge, config.Width, config.Height)
	}
	estimated, overflow := safeImageMemoryEstimate(pixels)
	if overflow || estimated > p.limits.MemoryBudgetBytes {
		return fmt.Errorf("%w: estimated %d bytes", ErrImageMemoryBudgetExceeded, estimated)
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	if memory.HeapAlloc > uint64(p.limits.MemoryBudgetBytes-estimated) {
		return ErrImageMemoryBudgetExceeded
	}
	return nil
}

func copyWithLimit(destination io.Writer, source io.Reader, maxBytes int64) error {
	written, err := io.Copy(destination, io.LimitReader(source, maxBytes+1))
	if err != nil {
		return err
	}
	if written > maxBytes {
		return ErrImageInputTooLarge
	}
	return nil
}

func safeImageMemoryEstimate(pixels int64) (int64, bool) {
	if pixels > (math.MaxInt64-imageProcessingOverhead)/imageProcessingBytesPerPixel {
		return 0, true
	}
	return pixels*imageProcessingBytesPerPixel + imageProcessingOverhead, false
}
