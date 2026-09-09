// Package main запускает отдельный calls signaling-сервис.
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
	"github.com/redis/go-redis/v9"

	"general-project/call-signaling/internal/adapters/readiness"
	redispublisher "general-project/call-signaling/internal/adapters/redis"
	"general-project/call-signaling/internal/application"
	"general-project/call-signaling/internal/authz"
	"general-project/call-signaling/internal/config"
	"general-project/call-signaling/internal/server"
	"general-project/call-signaling/internal/store"
	platformauth "general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	platformpostgres "general-project/libs/platform/postgres"
	"general-project/libs/platform/ratelimit"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	pool, err := platformpostgres.Open(ctx, cfg.PostgresURL, 10)
	if err != nil {
		logger.Error("calls database unavailable", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("calls Redis unavailable", "error", err)
		pool.Close()
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("calls Redis close failed", "error", err)
		}
	}()

	sessions := store.NewRedisStore(redisClient, int(cfg.SessionTTL/time.Second))
	authorizer := authz.NewRepository(pool)
	calls := application.NewService(sessions, authorizer)
	broker := redispublisher.NewBroker(redisClient)
	hub := server.NewHub(broker, logger)
	handler := server.NewHandler(calls, hub, platformauth.NewJWTVerifier(cfg.JWTSecret), readiness.NewChecker(pool, redisClient), cfg.AllowedOrigins, cfg.MaxMessageSize)

	go func() {
		if err := hub.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("calls signaling subscriber stopped", "error", err)
			stop()
		}
	}()

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", server.Health)
	router.Get("/readyz", handler.Ready)
	limiter := ratelimit.NewRedisFixedWindow(redisClient)
	router.With(ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
		return "calls:websocket:handshake:" + httpx.ClientIP(r)
	})).Get("/ws", handler.ServeHTTP)

	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("calls signaling service started", "addr", cfg.HTTPAddr, "ttl", cfg.SessionTTL.String())
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("calls signaling service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("calls HTTP server shutdown failed", "error", err)
	}
	pool.Close()
	stop()
}
