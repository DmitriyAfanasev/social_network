// Package main запускает messaging-сервис.
//
// @title Messaging API
// @version 1.0
// @description HTTP-контракт прямых диалогов и сообщений.
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
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/cache"
	"general-project/libs/platform/httpx"
	platformpostgres "general-project/libs/platform/postgres"
	"general-project/libs/platform/ratelimit"
	accessadapter "general-project/messaging/internal/adapters/access"
	eventsadapter "general-project/messaging/internal/adapters/events"
	postgresadapter "general-project/messaging/internal/adapters/postgres"
	redisadapter "general-project/messaging/internal/adapters/redis"
	"general-project/messaging/internal/application"
	"general-project/messaging/internal/config"
	"general-project/messaging/internal/ports"
	httptransport "general-project/messaging/internal/transport/http"
	ssestransport "general-project/messaging/internal/transport/sse"
	websockettransport "general-project/messaging/internal/transport/websocket"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := platformpostgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("messaging database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	repository := postgresadapter.NewRepository(pool)
	broker := redisadapter.NewRedisRealtimeBrokerWithLimit(redisClient, cfg.RealtimeInflightLimit)
	defer broker.Close()
	presence := redisadapter.NewRedisPresenceStore(redisClient)
	messageAccess := accessadapter.NewClient(cfg.ProfilesURL, cfg.SocialURL)
	messageService := application.NewMessageService(repository, cache.NewRedis(redisClient), broker, messageAccess)
	verifier := auth.NewJWTVerifier(cfg.JWTSecret)
	notificationHub := ssestransport.NewHub()
	consumerGroupID := cfg.KafkaGroupID + "-" + strconv.Itoa(os.Getpid())
	notificationConsumer := eventsadapter.NewConsumer(cfg.KafkaBrokers, cfg.EventsTopic, consumerGroupID, cfg.KafkaMaxBytes)
	notificationHandler := application.NewNotificationEventHandler(notificationHub)
	handler := httptransport.NewHandler(postgresadapter.NewHealthChecker(pool), messageService, presence, verifier, notificationHub)
	hub := websockettransport.NewHub()
	websocketHandler := websockettransport.NewHandler(messageService, verifier, hub, presence)
	presenceFanout := ports.PresenceFanout(presence)
	go func() {
		if err := presenceFanout.SubscribePresence(ctx, func(_ context.Context, event ports.PresenceEvent) error {
			payload, marshalErr := json.Marshal(map[string]any{"type": "presence.changed", "user_id": event.UserID, "online": event.Online})
			if marshalErr != nil {
				return marshalErr
			}
			hub.BroadcastAll(payload)
			return nil
		}); err != nil && ctx.Err() == nil {
			logger.Error("messaging presence fanout stopped", "error", err)
			stop()
		}
	}()
	go func() {
		if err := notificationConsumer.Run(ctx, notificationHandler.Handle); err != nil && ctx.Err() == nil {
			logger.Error("messaging notification consumer stopped", "error", err)
			stop()
		}
	}()
	go func() {
		err := broker.Subscribe(ctx, func(_ context.Context, event ports.RealtimeEvent) error {
			hub.Broadcast(event.RecipientIDs, event.Payload)
			notificationHub.Publish(event)
			return nil
		})
		if err != nil && ctx.Err() == nil {
			logger.Error("messaging realtime subscriber stopped", "error", err)
			stop()
		}
	}()

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Get("/ws", websocketHandler.ServeHTTP)
	streamLimiter := ratelimit.NewRedisFixedWindow(redisClient)
	router.With(ratelimit.Middleware(streamLimiter, 30, time.Minute, func(r *http.Request) string {
		return "messaging:notifications:stream:" + httpx.ClientIP(r)
	})).Get("/v1/notifications/stream", handler.StreamNotifications)
	router.Route("/v1/messaging", func(router chi.Router) {
		limiter := ratelimit.NewRedisFixedWindow(redisClient)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:read:" + httpx.ClientIP(r)
		})).Get("/conversations", handler.ListConversations)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Patch("/conversations/{conversationID}/archive", handler.ArchiveConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Patch("/conversations/{conversationID}/unarchive", handler.UnarchiveConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Delete("/conversations/{conversationID}", handler.HideConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Post("/conversations/{conversationID}/unread", handler.MarkConversationUnread)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Patch("/conversations/{conversationID}/pin", handler.PinConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Patch("/conversations/{conversationID}/unpin", handler.UnpinConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Patch("/conversations/{conversationID}/mute", handler.MuteConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Patch("/conversations/{conversationID}/unmute", handler.UnmuteConversation)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:state:" + httpx.ClientIP(r)
		})).Delete("/conversations/{conversationID}/history", handler.ClearHistory)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "messaging:conversations:direct:" + httpx.ClientIP(r)
		})).Post("/conversations/direct", handler.GetOrCreateDirect)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "messaging:messages:read:" + httpx.ClientIP(r)
		})).Get("/conversations/{conversationID}/messages", handler.ListMessages)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 120, time.Minute, func(r *http.Request) string {
			return "messaging:presence:read:" + httpx.ClientIP(r)
		})).Get("/presence/{userID}", handler.GetPresence)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:messages:write:" + httpx.ClientIP(r)
		})).Post("/conversations/{conversationID}/messages", handler.SendMessage)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "messaging:messages:read:" + httpx.ClientIP(r)
		})).Post("/conversations/{conversationID}/messages/{messageID}/read", handler.MarkRead)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "messaging:messages:write:" + httpx.ClientIP(r)
		})).Patch("/messages/{messageID}", handler.UpdateMessage)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "messaging:messages:write:" + httpx.ClientIP(r)
		})).Delete("/messages/{messageID}", handler.DeleteMessage)
		router.With(auth.Middleware(verifier), ratelimit.Middleware(limiter, 30, time.Minute, func(r *http.Request) string {
			return "messaging:messages:write:" + httpx.ClientIP(r)
		})).Delete("/conversations/{conversationID}/messages/{messageID}/media", handler.RemoveMessageMedia)
	})

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("messaging service started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("messaging service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
