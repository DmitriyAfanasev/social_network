// Package proxy содержит edge-маршрутизацию gateway между Go-сервисами.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"general-project/gateway/internal/config"
	"general-project/gateway/internal/swagger"
	"general-project/libs/platform/httpx"
	"general-project/libs/platform/ratelimit"
)

type backend struct {
	name string
	base string
}

// NewRouter создаёт HTTP-маршрутизатор gateway с REST, SSE и WebSocket proxy.
func NewRouter(cfg config.Config, logger *slog.Logger, limiter *ratelimit.RedisFixedWindow) (http.Handler, error) {
	backends := []backend{
		{name: "identity", base: cfg.IdentityURL},
		{name: "profiles", base: cfg.ProfilesURL},
		{name: "social", base: cfg.SocialURL},
		{name: "content", base: cfg.ContentURL},
		{name: "media", base: cfg.MediaURL},
		{name: "messaging", base: cfg.MessagingURL},
		{name: "call-signaling", base: cfg.CallsURL},
		{name: "analytics", base: cfg.AnalyticsURL},
	}

	proxies := make(map[string]http.Handler, len(backends))
	for _, item := range backends {
		proxy, err := newReverseProxy(item, logger, nil)
		if err != nil {
			return nil, err
		}
		proxies[item.name] = proxy
	}

	messagingWebSocket, err := newReverseProxy(backend{name: "messaging", base: cfg.MessagingURL}, logger, func(request *http.Request) {
		request.URL.Path = "/ws"
		request.URL.RawPath = ""
	})
	if err != nil {
		return nil, err
	}
	callsWebSocket, err := newReverseProxy(backend{name: "call-signaling", base: cfg.CallsURL}, logger, func(request *http.Request) {
		request.URL.Path = "/ws"
		request.URL.RawPath = ""
	})
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recovery(logger))
	router.Use(httpx.AccessLog(logger))
	router.Use(cors(cfg.AllowedOrigins))
	router.Get("/healthz", health)
	router.Get("/readyz", readiness(backends, cfg.ReadinessTimeout, logger))
	router.Get("/swagger", swagger.Redirect)
	router.Get("/swagger/", swagger.UI)
	router.Get("/swagger/index.html", swagger.UI)
	router.Get("/swagger.json", swagger.JSON)
	router.Get("/swagger.yaml", swagger.YAML)
	router.NotFound(notFound)
	router.MethodNotAllowed(methodNotAllowed)

	edge := func(handler http.Handler) http.Handler {
		if limiter == nil {
			return handler
		}
		return ratelimit.Middleware(limiter, cfg.RateLimit, cfg.RateLimitWindow, func(request *http.Request) string {
			return "gateway:request:" + httpx.ClientIP(request)
		})(handler)
	}

	// WebSocket routes need an explicit rewrite: the public gateway path is
	// versioned, while the services expose a deliberately short internal path.
	router.Handle("/v1/messaging/ws", edge(messagingWebSocket))
	router.Handle("/ws", edge(messagingWebSocket))
	router.Handle("/v1/calls/ws", edge(callsWebSocket))
	router.Handle("/calls/ws", edge(callsWebSocket))

	router.Mount("/v1/auth", edge(proxies["identity"]))
	router.Mount("/v1/profiles", edge(proxies["profiles"]))
	router.Mount("/v1/social", edge(proxies["social"]))
	router.Mount("/v1/content", edge(proxies["content"]))
	router.Mount("/v1/media", edge(proxies["media"]))
	router.Mount("/v1/messaging", edge(proxies["messaging"]))
	router.Mount("/v1/notifications", edge(proxies["messaging"]))
	router.Mount("/v1/analytics", edge(proxies["analytics"]))

	return router, nil
}

func newReverseProxy(item backend, logger *slog.Logger, rewrite func(*http.Request)) (http.Handler, error) {
	target, err := url.Parse(strings.TrimRight(item.base, "/"))
	if err != nil {
		return nil, errors.New("invalid " + item.name + " upstream URL: " + err.Error())
	}
	if target.Scheme == "" || target.Host == "" {
		return nil, errors.New("invalid " + item.name + " upstream URL: scheme and host are required")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Rewrite = func(request *httputil.ProxyRequest) {
		request.SetURL(target)
		if rewrite != nil {
			rewrite(request.Out)
		}
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		logger.ErrorContext(request.Context(), "gateway upstream request failed", "backend", item.name, "error", err)
		httpx.WriteError(writer, http.StatusBadGateway, "upstream_unavailable", "upstream service unavailable")
	}
	return proxy, nil
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]string{"status": "ok"}); err != nil {
		return
	}
}

func readiness(backends []backend, timeout time.Duration, logger *slog.Logger) http.HandlerFunc {
	client := &http.Client{Timeout: timeout}
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), timeout)
		defer cancel()

		type result struct {
			name string
			err  error
		}
		results := make(chan result, len(backends))
		var waitGroup sync.WaitGroup
		for _, item := range backends {
			item := item
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				probeRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(item.base, "/")+"/readyz", nil)
				if err != nil {
					results <- result{name: item.name, err: err}
					return
				}
				response, err := client.Do(probeRequest)
				if err == nil {
					if closeErr := response.Body.Close(); closeErr != nil {
						err = closeErr
					}
					if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
						err = errors.New(response.Status)
					}
				}
				results <- result{name: item.name, err: err}
			}()
		}
		waitGroup.Wait()
		close(results)

		var failed []string
		for item := range results {
			if item.err != nil {
				failed = append(failed, item.name)
				logger.WarnContext(request.Context(), "gateway upstream is not ready", "backend", item.name, "error", item.err)
			}
		}
		if len(failed) > 0 {
			httpx.WriteError(writer, http.StatusServiceUnavailable, "dependencies_not_ready", "gateway dependencies are not ready")
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(map[string]string{"status": "ready"}); err != nil {
			return
		}
	}
}

func cors(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			_, isAllowed := allowed[origin]
			if isAllowed {
				writer.Header().Set("Access-Control-Allow-Origin", origin)
				writer.Header().Set("Access-Control-Allow-Credentials", "true")
				writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				writer.Header().Add("Vary", "Origin")
			}
			if request.Method == http.MethodOptions {
				if !isAllowed {
					httpx.WriteError(writer, http.StatusForbidden, "origin_not_allowed", "origin is not allowed")
					return
				}
				writer.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func notFound(writer http.ResponseWriter, _ *http.Request) {
	httpx.WriteError(writer, http.StatusNotFound, "not_found", "route not found")
}

func methodNotAllowed(writer http.ResponseWriter, _ *http.Request) {
	httpx.WriteError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}
