// Package main запускает profiles-сервис.
//
// @title Profiles API
// @version 1.0
// @description Новый HTTP-контракт публичных профилей.
// @BasePath /
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/cache"
	"general-project/libs/platform/httpx"
	"general-project/libs/platform/postgres"
	"general-project/libs/platform/ratelimit"
	eventsadapter "general-project/profiles/internal/adapters/events"
	mediaadapter "general-project/profiles/internal/adapters/media"
	postgresadapter "general-project/profiles/internal/adapters/postgres"
	socialadapter "general-project/profiles/internal/adapters/social"
	"general-project/profiles/internal/application"
	"general-project/profiles/internal/config"
	"general-project/profiles/internal/ports"
	httptransport "general-project/profiles/internal/transport/http"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("profiles database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	readiness := postgresadapter.NewHealthChecker(pool)
	profiles := postgresadapter.NewProfileRepository(pool)
	profileMediaRepository := postgresadapter.NewProfileMediaRepository(pool)
	profileCache := cache.NewRedis(redisClient)
	socialClient := socialadapter.NewClient(cfg.SocialURL)
	profileService := application.NewProfileService(profiles, profileCache, socialClient)
	eventConsumer := eventsadapter.NewConsumer(cfg.KafkaBrokers, cfg.EventsTopic, cfg.ConsumerGroup)
	defer eventConsumer.Close()
	go func() {
		handleEvent := func(eventCtx context.Context, event ports.IntegrationEvent) error {
			if event.EventType != "identity.user.registered" {
				return nil
			}
			var payload struct {
				UserID string `json:"user_id"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return err
			}
			userID, err := uuid.Parse(payload.UserID)
			if err != nil {
				return err
			}
			return profileService.EnsureProfile(eventCtx, userID)
		}
		if consumeErr := eventConsumer.Run(ctx, handleEvent); consumeErr != nil && ctx.Err() == nil {
			logger.Error("profiles event consumer stopped", "error", consumeErr)
			stop()
		}
	}()
	profileMediaService := application.NewGalleryService(profiles, profileMediaRepository, profileMediaRepository, profileCache, mediaadapter.NewClient(cfg.MediaURL))
	verifier := auth.NewJWTVerifier(cfg.JWTSecret)
	handler := httptransport.NewHandler(readiness, profileService, profileMediaService)

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Route("/v1/profiles", func(router chi.Router) {
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(ratelimit.NewRedisFixedWindow(redisClient), 300, time.Minute, func(r *http.Request) string { return "profiles:photo:visibility:" + httpx.ClientIP(r) })).Get("/photo-media/{mediaID}/visibility", handler.PhotoMediaVisibility)
		limiter := ratelimit.NewRedisFixedWindow(redisClient)
		router.With(ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "profiles:search:" + httpx.ClientIP(r)
		})).Get("/search", handler.Search)
		router.With(auth.OptionalMiddleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "profiles:read:" + httpx.ClientIP(r)
		})).Get("/{handle}", handler.GetByHandle)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "profiles:me:" + httpx.ClientIP(r)
		})).Get("/me", handler.GetMine)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:set-handle:" + httpx.ClientIP(r)
		})).Put("/me/handle", handler.SetHandle)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:update:" + httpx.ClientIP(r)
		})).Patch("/me", handler.UpdatePublicProfile)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "profiles:privacy:read:" + httpx.ClientIP(r)
		})).Get("/me/privacy", handler.GetPrivacy)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:privacy:write:" + httpx.ClientIP(r)
		})).Put("/me/privacy", handler.UpdatePrivacy)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "profiles:photos:read:" + httpx.ClientIP(r)
		})).Get("/{handle}/photos", handler.GetPhotos)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:album:create:" + httpx.ClientIP(r)
		})).Post("/me/photo-albums", handler.CreatePhotoAlbum)
		router.Group(func(g chi.Router) {
			g.Use(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string { return "profiles:gallery:" + httpx.ClientIP(r) }))
			g.Put("/me/photo-albums/{albumID}", handler.UpdatePhotoAlbum)
			g.Put("/me/photos/{photoID}", handler.UpdatePhoto)
			g.Get("/photos/{photoID}/comments", handler.ListPhotoComments)
			g.Post("/photos/{photoID}/comments", handler.AddPhotoComment)
		})
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "profiles:photo:add:" + httpx.ClientIP(r)
		})).Post("/me/photo-albums/{albumID}/photos", handler.AddPhoto)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "profiles:photo:delete:" + httpx.ClientIP(r)
		})).Delete("/me/photos/{photoID}", handler.DeletePhoto)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:avatar:write:" + httpx.ClientIP(r)
		})).Put("/me/avatar", handler.SetAvatar)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "profiles:avatar:read:" + httpx.ClientIP(r)
		})).Get("/me/avatar/history", handler.AvatarHistory)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:avatar:select:" + httpx.ClientIP(r)
		})).Post("/me/avatar/select", handler.SelectAvatar)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "profiles:avatar:delete:" + httpx.ClientIP(r)
		})).Delete("/me/avatar", handler.RemoveAvatar)
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("profiles service started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("profiles service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
