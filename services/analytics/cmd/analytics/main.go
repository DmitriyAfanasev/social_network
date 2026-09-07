// Package main запускает analytics worker.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"general-project/analytics/internal/adapters/kafka"
	"general-project/analytics/internal/adapters/postgres"
	"general-project/analytics/internal/application"
	"general-project/analytics/internal/config"
	httptransport "general-project/analytics/internal/transport/http"
	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
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
	videoAnalytics := application.NewVideoAnalyticsService(postgres.NewVideoAnalyticsRepository(pool))
	handler := httptransport.NewHandler(videoAnalytics, postgres.NewHealthChecker(pool))
	verifier := auth.NewJWTVerifier(cfg.JWTSecret)
	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Route("/v1/analytics", func(router chi.Router) {
		router.With(auth.Middleware(verifier)).Get("/videos/{videoID}", handler.GetVideoStats)
		router.With(auth.Middleware(verifier)).Get("/me/videos", handler.ListMyVideoStats)
	})
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("analytics HTTP server started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("analytics HTTP server stopped", "error", serveErr)
			stop()
		}
	}()

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
