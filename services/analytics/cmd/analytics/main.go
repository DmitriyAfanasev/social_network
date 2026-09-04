// Package main запускает analytics worker.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"general-project/analytics/internal/adapters/kafka"
	"general-project/analytics/internal/adapters/postgres"
	"general-project/analytics/internal/application"
	"general-project/analytics/internal/config"
	platformpostgres "general-project/libs/platform/postgres"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("analytics database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	outbox := postgres.NewOutboxRepository(pool)
	processed := postgres.NewProcessedEventRepository(pool)
	recordHandler := postgres.NewEventRecordHandler(pool)
	eventPublisher := kafka.NewPublisherWithDLQ(cfg.KafkaBrokers, cfg.EventsTopic, cfg.DLQTopic)
	defer eventPublisher.Close()
	eventConsumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.EventsTopic, cfg.ConsumerGroup, cfg.KafkaMaxBytes)
	outboxPublisher := application.NewOutboxPublisher(outbox, eventPublisher)
	consumer := application.NewEventConsumer(processed, recordHandler, eventPublisher, cfg.MaxAttempts, cfg.RetryBase)

	go func() {
		if err := outboxPublisher.Run(ctx, cfg.PollInterval); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("analytics outbox publisher stopped", "error", err)
			stop()
		}
	}()
	if err := eventConsumer.Run(ctx, consumer.Handle); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("analytics event consumer stopped", "error", err)
		stop()
	}
}
