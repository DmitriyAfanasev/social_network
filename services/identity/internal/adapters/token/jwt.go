// Package token содержит адаптер выпуска JWT-токенов identity-сервиса.
package token

import (
	"errors"
	"time"

	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

// JWTIssuer выпускает подписанные access-токены.
type JWTIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

// NewJWTIssuer создаёт JWT-адаптер с секретом и временем жизни access-токена.
func NewJWTIssuer(secret string, accessTTL time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), accessTTL: accessTTL}
}

// IssueAccess выпускает access-токен с UUID пользователя в claim sub.
func (i *JWTIssuer) IssueAccess(userID uuid.UUID) (string, error) {
	if len(i.secret) == 0 || i.accessTTL <= 0 {
		return "", errors.New("JWT issuer is not configured")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.accessTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}
