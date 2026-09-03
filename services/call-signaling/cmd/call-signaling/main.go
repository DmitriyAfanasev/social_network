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

	"general-project/call-signaling/internal/auth"
	"general-project/call-signaling/internal/authz"
	"general-project/call-signaling/internal/config"
	"general-project/call-signaling/internal/server"
	"general-project/call-signaling/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	/*
		The process owns only control-plane dependencies. PostgreSQL is used for
		fresh authorization decisions, Redis is used for ephemeral state and
		fan-out, and the HTTP server carries WebSocket signaling. No media server
		or RTP proxy is started here because the current product is browser-to-
		browser one-to-one WebRTC.
	*/
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("Redis is unavailable", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		logger.Error("PostgreSQL pool creation failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("PostgreSQL is unavailable", "error", err)
		os.Exit(1)
	}

	ttlSeconds := int(cfg.SessionTTL / time.Second)
	sessionStore := store.NewRedisStore(redisClient, ttlSeconds)
	hub := server.NewHub(redisClient, sessionStore, authz.NewRepository(pool), logger)
	handler := &server.Handler{Hub: hub, Verifier: auth.NewVerifier(cfg.JWTSecret, cfg.JWTAlgorithm), Logger: logger, AllowedOrigins: cfg.AllowedOrigins, MaxMessageSize: cfg.MaxMessageSize}
	mux := http.NewServeMux()
	mux.Handle("/ws", handler)
	mux.HandleFunc("/healthz", server.Health)

	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("call signaling service started", "addr", cfg.HTTPAddr, "ttl", cfg.SessionTTL.String())
		serveErr := httpServer.ListenAndServe()
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", serveErr)
			stop()
		}
	}()

	if err := hub.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Redis subscriber stopped", "error", err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
