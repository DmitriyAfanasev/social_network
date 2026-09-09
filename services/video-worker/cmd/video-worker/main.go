// Package main запускает worker транскодирования видео.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"general-project/video-worker/internal/config"
	"general-project/video-worker/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	w, err := worker.New(cfg, logger)
	if err != nil {
		logger.Error("worker initialization failed", "error", err)
		os.Exit(1)
	}

	logger.Info("video worker started", "group_id", cfg.KafkaGroupID, "parallel_jobs", cfg.ParallelJobs)
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("video worker stopped with error", "error", err)
		os.Exit(1)
	}
	stop()
	logger.Info("video worker stopped gracefully")
}
