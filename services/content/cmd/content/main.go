// Package main запускает content-сервис.
//
// @title Content API
// @version 1.0
// @description HTTP-контракт текстовых постов.
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

	eventsadapter "general-project/content/internal/adapters/events"
	mediahttpadapter "general-project/content/internal/adapters/mediahttp"
	"general-project/content/internal/adapters/postgres"
	"general-project/content/internal/application"
	"general-project/content/internal/config"
	httptransport "general-project/content/internal/transport/http"
	"general-project/libs/platform/auth"
	"general-project/libs/platform/cache"
	"general-project/libs/platform/httpx"
	platformpostgres "general-project/libs/platform/postgres"
	"general-project/libs/platform/ratelimit"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("content database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("content Redis close failed", "error", err)
		}
	}()

	readiness := postgres.NewHealthChecker(pool)
	posts := postgres.NewPostRepository(pool)
	comments := postgres.NewCommentRepository(pool)
	likes := postgres.NewLikeRepository(pool)
	outbox := postgres.NewOutboxRepository(pool)
	postCache := cache.NewRedis(redisClient)
	mediaChecker := mediahttpadapter.NewClient(cfg.MediaAddr)
	contentService := application.NewContentServiceWithLikes(posts, postCache, mediaChecker, outbox, likes)
	eventPublisher := eventsadapter.NewPublisher(cfg.KafkaBrokers, cfg.EventsTopic)
	defer func() {
		if err := eventPublisher.Close(); err != nil {
			logger.Error("content event publisher close failed", "error", err)
		}
	}()
	outboxPublisher := application.NewOutboxPublisher(outbox, eventPublisher)
	go func() {
		if publishErr := outboxPublisher.Run(ctx, time.Second); publishErr != nil && ctx.Err() == nil {
			logger.Error("content outbox publisher stopped", "error", publishErr)
			stop()
		}
	}()
	commentService := application.NewCommentService(posts, comments, postCache)
	likeService := application.NewLikeService(posts, likes)
	verifier := auth.NewJWTVerifier(cfg.JWTSecret)
	handler := httptransport.NewHandler(readiness, contentService, commentService, likeService)

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Route("/v1/content", func(router chi.Router) {
		limiter := ratelimit.NewRedisFixedWindow(redisClient)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "content:read:" + httpx.ClientIP(r)
		})).Get("/posts/{postID}", handler.GetByID)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "content:feed:" + httpx.ClientIP(r)
		})).Get("/feed", handler.Feed)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "content:create:" + httpx.ClientIP(r)
		})).Post("/posts", handler.Create)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "content:update:" + httpx.ClientIP(r)
		})).Patch("/posts/{postID}", handler.Update)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "content:delete:" + httpx.ClientIP(r)
		})).Delete("/posts/{postID}", handler.Delete)
		router.With(ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "content:comments:read:" + httpx.ClientIP(r)
		})).Get("/posts/{postID}/comments", handler.ListComments)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "content:comments:create:" + httpx.ClientIP(r)
		})).Post("/posts/{postID}/comments", handler.CreateComment)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "content:comments:delete:" + httpx.ClientIP(r)
		})).Delete("/comments/{commentID}", handler.DeleteComment)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "content:likes:" + httpx.ClientIP(r)
		})).Post("/posts/{postID}/like", handler.Like)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "content:likes:" + httpx.ClientIP(r)
		})).Delete("/posts/{postID}/like", handler.Unlike)
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("content service started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("content service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("content HTTP server shutdown failed", "error", err)
	}
	stop()
}
