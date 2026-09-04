// Package auth содержит общие примитивы проверки access-токенов.
package auth

import (
	"errors"
	"fmt"
	"strings"

	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrUnauthorized означает, что access-токен отсутствует или недействителен.
	ErrUnauthorized = errors.New("unauthorized")
)

// JWTVerifier проверяет access-токены, выпущенные identity-сервисом.
type JWTVerifier struct {
	secret []byte
}

// NewJWTVerifier создаёт проверяющий адаптер с общим секретом.
func NewJWTVerifier(secret string) *JWTVerifier {
	return &JWTVerifier{secret: []byte(secret)}
}

// UserID извлекает UUID пользователя из bearer access-токена.
func (v *JWTVerifier) UserID(tokenValue string) (uuid.UUID, error) {
	if len(v.secret) == 0 || strings.TrimSpace(tokenValue) == "" {
		return uuid.Nil(), ErrUnauthorized
	}
	token, err := jwt.ParseWithClaims(tokenValue, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected JWT signing method")
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return uuid.Nil(), ErrUnauthorized
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil(), ErrUnauthorized
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil(), ErrUnauthorized
	}
	return userID, nil
}
