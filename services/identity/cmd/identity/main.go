// Package main запускает identity-сервис.
//
// @title Identity API
// @version 1.0
// @description Новый HTTP-контракт identity-сервиса.
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

	eventsadapter "general-project/identity/internal/adapters/events"
	"general-project/identity/internal/adapters/notifications"
	postgresadapter "general-project/identity/internal/adapters/postgres"
	"general-project/identity/internal/adapters/security"
	"general-project/identity/internal/adapters/token"
	"general-project/identity/internal/application"
	"general-project/identity/internal/config"
	httptransport "general-project/identity/internal/transport/http"
	platformauth "general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/libs/platform/postgres"
	"general-project/libs/platform/ratelimit"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Open(ctx, cfg.DatabaseURL, 10)
	if err != nil {
		logger.Error("identity database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()

	readiness := postgresadapter.NewHealthChecker(pool)
	users := postgresadapter.NewUserRepository(pool)
	refreshTokens := postgresadapter.NewRefreshTokenStore(pool)
	verificationTokens := postgresadapter.NewVerificationTokenStore(pool)
	outbox := postgresadapter.NewOutboxRepository(pool)
	eventPublisher := eventsadapter.NewPublisher(cfg.KafkaBrokers, cfg.EventsTopic)
	defer eventPublisher.Close()
	outboxPublisher := application.NewOutboxPublisher(outbox, eventPublisher)
	go func() {
		if publishErr := outboxPublisher.Run(ctx, time.Second); publishErr != nil && ctx.Err() == nil {
			logger.Error("identity outbox publisher stopped", "error", publishErr)
			stop()
		}
	}()
	hasher := security.NewBcryptHasher(0)
	tokens := token.NewJWTIssuer(cfg.JWTSecret, cfg.AccessTTL)
	verificationSender := notifications.NewSMTP(cfg.SMTPAddr, cfg.SMTPFrom)
	authService := application.NewAuthService(users, hasher, tokens, refreshTokens, cfg.AccessTTL, cfg.RefreshTTL, application.VerificationConfig{
		Tokens:           verificationTokens,
		Notifications:    verificationSender,
		ConfirmationTTL:  cfg.ConfirmationTTL,
		PasswordResetTTL: cfg.PasswordResetTTL,
		FrontendBaseURL:  cfg.FrontendBaseURL,
	})
	verifier := platformauth.NewJWTVerifier(cfg.JWTSecret)
	handler := httptransport.NewHandler(readiness, authService)
	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Get("/healthz", handler.Health)
	router.Get("/readyz", handler.Ready)
	router.Route("/v1/auth", func(router chi.Router) {
		limiter := ratelimit.NewRedisFixedWindow(redisClient)
		router.With(ratelimit.Middleware(limiter, 5, time.Minute, func(r *http.Request) string {
			return "identity:register:" + httpx.ClientIP(r)
		})).Post("/register", handler.Register)
		router.With(ratelimit.Middleware(limiter, 5, time.Minute, func(r *http.Request) string {
			return "identity:registration-confirmation-request:" + httpx.ClientIP(r)
		})).Post("/registration-confirmation-requests", handler.RequestRegistrationConfirmation)
		router.With(ratelimit.Middleware(limiter, 10, time.Minute, func(r *http.Request) string {
			return "identity:registration-confirm:" + httpx.ClientIP(r)
		})).Post("/confirm-registration", handler.ConfirmRegistration)
		router.With(ratelimit.Middleware(limiter, 5, time.Minute, func(r *http.Request) string {
			return "identity:password-reset-request:" + httpx.ClientIP(r)
		})).Post("/password-reset-requests", handler.RequestPasswordReset)
		router.With(ratelimit.Middleware(limiter, 10, time.Minute, func(r *http.Request) string {
			return "identity:password-reset:" + httpx.ClientIP(r)
		})).Post("/password-reset", handler.ResetPassword)
		router.With(ratelimit.Middleware(limiter, 10, time.Minute, func(r *http.Request) string {
			return "identity:login:" + httpx.ClientIP(r)
		})).Post("/login", handler.Login)
		router.With(ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "identity:refresh:" + httpx.ClientIP(r)
		})).Post("/refresh", handler.Refresh)
		router.With(ratelimit.Middleware(limiter, 20, time.Minute, func(r *http.Request) string {
			return "identity:logout:" + httpx.ClientIP(r)
		})).Post("/logout", handler.Logout)
		router.With(platformauth.Middleware(verifier), ratelimit.Middleware(limiter, 60, time.Minute, func(r *http.Request) string {
			return "identity:permissions:" + httpx.ClientIP(r)
		})).Get("/me/permissions", handler.Permissions)
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("identity service started", "addr", cfg.HTTPAddr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("identity service stopped", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
