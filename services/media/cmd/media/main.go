// Package main запускает media-сервис.
//
// @title Media API
// @version 1.0
// @description HTTP-контракт метаданных и загрузки медиаобъектов.
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
	accessadapter "general-project/media/internal/adapters/access"
	eventsadapter "general-project/media/internal/adapters/events"
	imagejobsadapter "general-project/media/internal/adapters/imagejobs"
	postgresadapter "general-project/media/internal/adapters/postgres"
	storageadapter "general-project/media/internal/adapters/storage"
	"general-project/media/internal/application"
	"general-project/media/internal/config"
	httptransport "general-project/media/internal/transport/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("media database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	storage, err := storageadapter.NewMinIOStorage(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Secure)
	if err != nil {
		logger.Error("media storage unavailable", "error", err)
		os.Exit(1)
	}
	if err := storage.EnsureBucket(ctx); err != nil {
		logger.Error("media bucket unavailable", "error", err)
		os.Exit(1)
	}
	readiness := postgresadapter.NewHealthChecker(pool)
	mediaRepository := postgresadapter.NewMediaRepository(pool)
	outbox := postgresadapter.NewOutboxRepository(pool)
	mediaCache := cache.NewRedis(redisClient)
	mediaService := application.NewProtectedMediaService(mediaRepository, mediaCache, storage, cfg.S3Bucket, accessadapter.NewClient(cfg.ProfilesURL, cfg.SocialURL), outbox)
	eventPublisher := eventsadapter.NewEventPublisher(cfg.KafkaBrokers, cfg.EventsTopic)
	defer eventPublisher.Close()
	imageJobs := imagejobsadapter.NewRedisStream(redisClient, cfg.ImageJobsStream, cfg.ImageJobsGroup, "media-outbox", cfg.ImageJobsMaxLength, cfg.ImageJobsRetryIdle)
	outboxPublisher := application.NewOutboxPublisher(outbox, eventPublisher, imageJobs)
	go func() {
		if publishErr := outboxPublisher.Run(ctx, time.Second); publishErr != nil && ctx.Err() == nil {
			logger.Error("media outbox publisher stopped", "error", publishErr)
			stop()
		}
	}()
	videoRepository := postgresadapter.NewVideoRepository(pool)
	videoPublisher := eventsadapter.NewKafkaPublisher(cfg.KafkaBrokers, config.VideoTranscodeRequestedTopic)
	defer videoPublisher.Close()
	videoService := application.NewVideoService(mediaService, videoRepository, videoPublisher, mediaCache, outbox)
	musicRepository := postgresadapter.NewMusicRepository(pool)
	musicAccess := accessadapter.NewClient(cfg.ProfilesURL, cfg.SocialURL)
	musicService := application.NewMusicService(mediaService, musicRepository, mediaCache, musicAccess)
	videoConsumer := eventsadapter.NewKafkaConsumer(cfg.KafkaBrokers, config.VideoTranscodeCompletedTopic, cfg.KafkaGroupID, cfg.KafkaMaxBytes)
	go func() {
		if consumeErr := videoConsumer.Run(ctx, videoService.Complete); consumeErr != nil && ctx.Err() == nil {
			logger.Error("video completion consumer stopped", "error", consumeErr)
			stop()
		}
	}()
	verifier := auth.NewJWTVerifier(cfg.JWTSecret)
	handler := httptransport.NewHandler(readiness, mediaService, videoService, musicService)

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Route("/v1/media", func(router chi.Router) {
		limiter := ratelimit.NewRedisFixedWindow(redisClient)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:content:" + httpx.ClientIP(r)
		})).Get("/{mediaID}/content", handler.StreamContent)
		router.With(ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:read:" + httpx.ClientIP(r)
		})).Get("/{mediaID}", handler.GetByID)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 10, time.Minute, func(r *http.Request) string {
			return "media:upload:" + httpx.ClientIP(r)
		})).Post("/", handler.Upload)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "media:delete:" + httpx.ClientIP(r)
		})).Delete("/{mediaID}", handler.Delete)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 5, time.Minute, func(r *http.Request) string {
			return "media:video:create:" + httpx.ClientIP(r)
		})).Post("/videos", handler.CreateVideo)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:list:" + httpx.ClientIP(r)
		})).Get("/videos", handler.ListVideos)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:read:" + httpx.ClientIP(r)
		})).Get("/videos/{videoID}", handler.GetVideo)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "media:video:delete:" + httpx.ClientIP(r)
		})).Delete("/videos/{videoID}", handler.DeleteVideo)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "media:video:album:create:" + httpx.ClientIP(r)
		})).Post("/videos/albums", handler.CreateAlbum)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "media:video:album:delete:" + httpx.ClientIP(r)
		})).Delete("/videos/albums/{albumID}", handler.DeleteAlbum)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:view:" + httpx.ClientIP(r)
		})).Post("/videos/{videoID}/view", handler.RecordView)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:like:" + httpx.ClientIP(r)
		})).Post("/videos/{videoID}/like", handler.ToggleLike)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:bookmark:" + httpx.ClientIP(r)
		})).Post("/videos/{videoID}/bookmark", handler.SetBookmark)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:bookmark:" + httpx.ClientIP(r)
		})).Delete("/videos/{videoID}/bookmark", handler.SetBookmark)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:favorite:" + httpx.ClientIP(r)
		})).Post("/videos/{videoID}/favorite", handler.SetFavorite)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:video:favorite:" + httpx.ClientIP(r)
		})).Delete("/videos/{videoID}/favorite", handler.SetFavorite)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "media:music:read:" + httpx.ClientIP(r)
		})).Get("/music", handler.ListMusic)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 10, time.Minute, func(r *http.Request) string {
			return "media:music:create:" + httpx.ClientIP(r)
		})).Post("/music", handler.CreateMusic)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "media:music:delete:" + httpx.ClientIP(r)
		})).Delete("/music/{trackID}", handler.DeleteMusic)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "media:music:library:" + httpx.ClientIP(r)
		})).Post("/music/{trackID}/save", handler.AddMusicToLibrary)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "media:music:library:" + httpx.ClientIP(r)
		})).Delete("/music/{trackID}/save", handler.RemoveMusicFromLibrary)
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("media service started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("media service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
