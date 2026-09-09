// Package main запускает фоновую оптимизацию изображений media-сервиса.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/redis/go-redis/v9"

	"general-project/libs/platform/cache"
	platformpostgres "general-project/libs/platform/postgres"
	imagejobsadapter "general-project/media/internal/adapters/imagejobs"
	postgresadapter "general-project/media/internal/adapters/postgres"
	storageadapter "general-project/media/internal/adapters/storage"
	"general-project/media/internal/application"
	"general-project/media/internal/config"
	"general-project/media/internal/ports"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Лимит Go runtime дополняет проверку размеров до декодирования и cgroup-лимит контейнера.
	debug.SetMemoryLimit(cfg.ImageWorkerMemoryBudgetBytes)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 4)
	if err != nil {
		logger.Error("image worker database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()
	storage, err := storageadapter.NewMinIOStorage(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Secure)
	if err != nil {
		logger.Error("image worker storage unavailable", "error", err)
		os.Exit(1)
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		logger.Error("image worker bucket unavailable", "error", err)
		os.Exit(1)
	}

	consumer, err := os.Hostname()
	if err != nil || consumer == "" {
		consumer = "media-image-worker"
	}
	queue := imagejobsadapter.NewRedisStream(redisClient, cfg.ImageJobsStream, cfg.ImageJobsGroup, consumer, cfg.ImageJobsMaxLength, cfg.ImageJobsRetryIdle)
	processor := application.NewImageProcessor(
		postgresadapter.NewMediaRepository(pool),
		cache.NewRedis(redisClient),
		storage,
		application.ImageProcessingLimits{
			MaxInputBytes:     cfg.ImageWorkerMaxInputBytes,
			MaxPixels:         cfg.ImageWorkerMaxPixels,
			MemoryBudgetBytes: cfg.ImageWorkerMemoryBudgetBytes,
			Quality:           cfg.ImageWorkerWebPQuality,
		},
	)
	logger.Info("media image worker started", "stream", cfg.ImageJobsStream, "group", cfg.ImageJobsGroup, "memory_budget_bytes", cfg.ImageWorkerMemoryBudgetBytes)
	if err := queue.Run(ctx, func(jobCtx context.Context, job ports.ImageOptimizationJob) error {
		err := processor.Process(jobCtx, job.MediaID)
		if errors.Is(err, application.ErrImageInputTooLarge) || errors.Is(err, application.ErrImageDimensionsTooLarge) || errors.Is(err, application.ErrImageMemoryBudgetExceeded) || errors.Is(err, application.ErrImageRejected) {
			logger.Warn("image optimization skipped", "media_id", job.MediaID, "error", err)
			return nil
		}
		return err
	}); err != nil && ctx.Err() == nil {
		logger.Error("media image worker stopped", "error", err)
		os.Exit(1)
	}
}
