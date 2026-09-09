// Package main запускает admin consumer moderation-аудита.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"general-project/admin/internal/adapters/events"
	"general-project/admin/internal/adapters/postgres"
	"general-project/admin/internal/application"
	"general-project/admin/internal/config"
	platformpostgres "general-project/libs/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 5)
	if err != nil {
		logger.Error("admin database unavailable", "error", err)
		return err
	}
	defer pool.Close()

	audit := application.NewAuditService(postgres.NewAuditRepository(pool))
	handler := application.NewEventHandler(audit)
	consumer := events.NewConsumer(cfg.KafkaBrokers, cfg.EventsTopic, cfg.ConsumerGroup)
	if err := consumer.Run(ctx, func(ctx context.Context, event events.Event) error {
		return handler.Handle(ctx, event.ID, event.EventType, event.CorrelationID, event.Payload, event.CreatedAt)
	}); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("admin audit consumer stopped", "error", err)
		return err
	}
	return nil
}
