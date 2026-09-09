// Package main запускает gateway/BFF для локальной композиции Go-сервисов.
//
// @title General Project API
// @version 1.0
// @description Единый публичный HTTP-контракт Go-бэкенда.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Access token в формате `Bearer <token>`.
// @security BearerAuth
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

	"github.com/redis/go-redis/v9"

	"general-project/gateway/internal/config"
	"general-project/gateway/internal/proxy"
	"general-project/libs/platform/ratelimit"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("gateway Redis unavailable", "error", err)
		os.Exit(1)
	}

	router, err := proxy.NewRouter(cfg, logger, ratelimit.NewRedisFixedWindow(redisClient))
	if err != nil {
		logger.Error("gateway configuration is invalid", "error", err)
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Error("gateway Redis close failed", "error", closeErr)
		}
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("gateway started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("gateway stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("gateway shutdown failed", "error", err)
	}
	if err := redisClient.Close(); err != nil {
		logger.Error("gateway Redis close failed", "error", err)
	}
	stop()
}
