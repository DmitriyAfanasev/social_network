package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"uuid"

	"general-project/video-worker/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/kafka-go"
)

// Job содержит команду на транскодирование видео.
type Job struct {
	VideoID          uuid.UUID `json:"video_id"`
	MediaID          uuid.UUID `json:"media_id"`
	Bucket           string    `json:"bucket"`
	ObjectKey        string    `json:"object_key"`
	RequestedHeights []int     `json:"requested_heights"`
}

// Rendition содержит результат обработки видео одного разрешения.
type Rendition struct {
	Height      int     `json:"height"`
	ObjectKey   string  `json:"object_key"`
	ContentType string  `json:"content_type"`
	Size        int64   `json:"size"`
	Duration    float64 `json:"duration"`
}

// Worker читает команды транскодирования и публикует результаты.
type Worker struct {
	cfg      config.Config
	logger   *slog.Logger
	storage  *minio.Client
	reader   *kafka.Reader
	producer *kafka.Writer
	slots    chan struct{}
}

func New(cfg config.Config, logger *slog.Logger) (*Worker, error) {
	storage, err := minio.New(cfg.S3Endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""), Secure: cfg.S3Secure})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	return &Worker{cfg: cfg, logger: logger, storage: storage,
		reader:   kafka.NewReader(kafka.ReaderConfig{Brokers: []string{cfg.KafkaBrokers}, Topic: config.VideoTranscodeRequestedTopic, GroupID: cfg.KafkaGroupID, MinBytes: 1, MaxBytes: cfg.KafkaMaxBytes}),
		producer: &kafka.Writer{Addr: kafka.TCP(cfg.KafkaBrokers), Topic: config.VideoTranscodeCompletedTopic, Balancer: &kafka.Hash{}}, slots: make(chan struct{}, cfg.ParallelJobs)}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	defer w.reader.Close()
	defer w.producer.Close()
	for {
		message, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("fetch Kafka message: %w", err)
		}
		var job Job
		if err := json.Unmarshal(message.Value, &job); err != nil {
			w.logger.Error("invalid video job", "error", err)
			_ = w.reader.CommitMessages(ctx, message)
			continue
		}
		if job.ObjectKey == "" {
			w.logger.Warn("obsolete video job skipped", "video_id", job.VideoID)
			_ = w.reader.CommitMessages(ctx, message)
			continue
		}
		w.logger.Info("video job started", "video_id", job.VideoID, "media_id", job.MediaID, "qualities", job.RequestedHeights)
		if err := w.process(ctx, job); err != nil {
			w.logger.Error("video job failed", "video_id", job.VideoID, "error", err)
			if publishErr := w.producer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(job.VideoID.String()),
				Value: mustJSON(map[string]any{"video_id": job.VideoID, "status": "failed", "error_message": err.Error()}),
			}); publishErr != nil {
				w.logger.Error("video failure event failed", "video_id", job.VideoID, "error", publishErr)
				continue
			}
			if commitErr := w.reader.CommitMessages(ctx, message); commitErr != nil {
				return fmt.Errorf("commit failed video message: %w", commitErr)
			}
			continue
		}
		if err := w.reader.CommitMessages(ctx, message); err != nil {
			return fmt.Errorf("commit Kafka message: %w", err)
		}
		w.logger.Info("video job completed", "video_id", job.VideoID)
	}
}

func (w *Worker) process(ctx context.Context, job Job) error {
	input, err := os.CreateTemp("", "video-original-*")
	if err != nil {
		return err
	}
	inputPath := input.Name()
	defer os.Remove(inputPath)
	input.Close()
	bucket := job.Bucket
	if bucket == "" {
		bucket = w.cfg.S3Bucket
	}
	if err := w.storage.FGetObject(ctx, bucket, job.ObjectKey, inputPath, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("download original: %w", err)
	}

	duration, err := probeDuration(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("probe duration: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(job.RequestedHeights))
	renditionCh := make(chan Rendition, len(job.RequestedHeights))
	for _, height := range job.RequestedHeights {
		height := height
		wg.Go(func() {
			w.slots <- struct{}{}
			defer func() { <-w.slots }()
			rendition, err := w.transcode(ctx, inputPath, job.VideoID, height, duration)
			if err != nil {
				errCh <- err
			} else {
				renditionCh <- rendition
			}
		})
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return err
	}
	close(renditionCh)
	renditions := make([]Rendition, 0, len(job.RequestedHeights))
	for rendition := range renditionCh {
		renditions = append(renditions, rendition)
	}
	return w.producer.WriteMessages(ctx, kafka.Message{
		Key: []byte(job.VideoID.String()),
		Value: mustJSON(map[string]any{
			"video_id":   job.VideoID,
			"status":     "ready",
			"duration":   duration,
			"renditions": renditions,
		}),
	})
}

func (w *Worker) transcode(ctx context.Context, input string, videoID uuid.UUID, height int, duration float64) (Rendition, error) {
	w.logger.Info("transcoding started", "video_id", videoID, "height", height)
	output, err := os.CreateTemp("", "video-rendition-*.mp4")
	if err != nil {
		return Rendition{}, err
	}
	outputPath := output.Name()
	defer os.Remove(outputPath)
	output.Close()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", input, "-vf", fmt.Sprintf("scale=-2:%d", height), "-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-c:a", "aac", "-movflags", "+faststart", outputPath)
	if logs, err := cmd.CombinedOutput(); err != nil {
		return Rendition{}, fmt.Errorf("ffmpeg %dp: %w: %s", height, err, string(logs))
	}
	key := filepath.Join("media", "videos", "renditions", videoID.String(), fmt.Sprintf("%dp.mp4", height))
	if _, err := w.storage.FPutObject(ctx, w.cfg.S3Bucket, key, outputPath, minio.PutObjectOptions{ContentType: "video/mp4"}); err != nil {
		return Rendition{}, err
	}
	info, err := w.storage.StatObject(ctx, w.cfg.S3Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return Rendition{}, err
	}
	w.logger.Info("transcoding completed", "video_id", videoID, "height", height, "object_key", key)
	return Rendition{Height: height, ObjectKey: key, ContentType: "video/mp4", Size: info.Size, Duration: duration}, nil
}

func probeDuration(ctx context.Context, input string) (float64, error) {
	output, err := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		input,
	).Output()
	if err != nil {
		return 0, err
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}
	return duration, nil
}

func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }
