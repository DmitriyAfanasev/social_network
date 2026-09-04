// Package main запускает social-сервис.
//
// @title Social API
// @version 1.0
// @description HTTP-контракт социальных отношений.
// @BasePath /
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

	"general-project/libs/platform/auth"
	"general-project/libs/platform/cache"
	"general-project/libs/platform/httpx"
	platformpostgres "general-project/libs/platform/postgres"
	"general-project/libs/platform/ratelimit"
	eventsadapter "general-project/social/internal/adapters/events"
	postgresadapter "general-project/social/internal/adapters/postgres"
	"general-project/social/internal/application"
	"general-project/social/internal/config"
	httptransport "general-project/social/internal/transport/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("social database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	blocks := postgresadapter.NewBlockRepository(pool)
	relationships := postgresadapter.NewFriendshipRepository(pool)
	outbox := postgresadapter.NewOutboxRepository(pool)
	eventPublisher := eventsadapter.NewPublisher(cfg.KafkaBrokers, cfg.EventsTopic)
	defer eventPublisher.Close()
	socialCache := cache.NewRedis(redisClient)
	socialService := application.NewSocialServiceWithOutbox(blocks, relationships, socialCache, outbox)
	outboxPublisher := application.NewOutboxPublisher(outbox, eventPublisher)
	go func() {
		if publishErr := outboxPublisher.Run(ctx, time.Second); publishErr != nil && ctx.Err() == nil {
			logger.Error("social outbox publisher stopped", "error", publishErr)
			stop()
		}
	}()
	verifier := auth.NewJWTVerifier(cfg.JWTSecret)
	readiness := postgresadapter.NewHealthChecker(pool)
	handler := httptransport.NewHandler(readiness, socialService)

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Route("/v1/social", func(router chi.Router) {
		limiter := ratelimit.NewRedisFixedWindow(redisClient)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:block:" + httpx.ClientIP(r)
		})).Put("/blocks/{targetID}", handler.Block)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:block:" + httpx.ClientIP(r)
		})).Delete("/blocks/{targetID}", handler.Unblock)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:friendship:" + httpx.ClientIP(r)
		})).Put("/friendships/{targetID}", handler.AddFriend)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:friendship:" + httpx.ClientIP(r)
		})).Delete("/friendships/{targetID}", handler.RemoveFriend)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "social:friend-requests:read:" + httpx.ClientIP(r)
		})).Get("/friend-requests", handler.ListFriendRequests)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:friend-requests:write:" + httpx.ClientIP(r)
		})).Post("/friend-requests/{requestID}/accept", handler.AcceptFriendRequest)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:friend-requests:write:" + httpx.ClientIP(r)
		})).Post("/friend-requests/{requestID}/decline", handler.DeclineFriendRequest)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:friend-requests:write:" + httpx.ClientIP(r)
		})).Delete("/friend-requests/{requestID}", handler.CancelFriendRequest)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "social:subscription:" + httpx.ClientIP(r)
		})).Put("/subscriptions/{targetID}", handler.Subscribe)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "social:subscription:" + httpx.ClientIP(r)
		})).Delete("/subscriptions/{targetID}", handler.Unsubscribe)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "social:relationships:read:" + httpx.ClientIP(r)
		})).Get("/relationships", handler.GetRelationships)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "social:recommendations:read:" + httpx.ClientIP(r)
		})).Get("/recommendations", handler.GetRecommendations)
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("social service started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("social service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
