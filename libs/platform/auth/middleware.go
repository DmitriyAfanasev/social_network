package auth

import (
	"context"
	"net/http"
	"strings"

	"uuid"

	"general-project/libs/platform/httpx"
)

type userIDContextKey struct{}

// TokenVerifier извлекает идентификатор пользователя из access-токена.
type TokenVerifier interface {
	UserID(token string) (uuid.UUID, error)
}

// Middleware проверяет bearer access-токен и добавляет UUID пользователя в контекст.
func Middleware(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := BearerToken(r.Header.Get("Authorization"))
			if !ok {
				unauthorized(w)
				return
			}
			userID, err := verifier.UserID(token)
			if err != nil {
				unauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware добавляет пользователя в контекст, если access-токен передан.
// При переданном недействительном токене запрос отклоняется как неавторизованный.
func OptionalMiddleware(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if strings.TrimSpace(header) == "" {
				next.ServeHTTP(w, r)
				return
			}
			token, ok := BearerToken(header)
			if !ok {
				unauthorized(w)
				return
			}
			userID, err := verifier.UserID(token)
			if err != nil {
				unauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext возвращает идентификатор пользователя, установленный middleware.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(uuid.UUID)
	return userID, ok && userID != uuid.Nil()
}

// BearerToken извлекает токен из значения HTTP-заголовка Authorization.
func BearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func unauthorized(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
}
