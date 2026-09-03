/*
Package auth authenticates the same HttpOnly access-token cookie as the Python
API. The browser WebSocket constructor naturally sends cookies for the origin,
so the server derives the user id from the verified JWT and never trusts a user
id supplied inside a signaling JSON message.
*/
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

var ErrUnauthorized = errors.New("unauthorized")

type Verifier struct {
	secret    []byte
	algorithm string
}

// NewVerifier builds a verifier using the secret shared with Python.
func NewVerifier(secret, algorithm string) *Verifier {
	return &Verifier{secret: []byte(secret), algorithm: algorithm}
}

// UserID validates signature, algorithm, expiry and the user_id claim.
// Restricting the HMAC method prevents an algorithm-confusion attack.
func (v *Verifier) UserID(r *http.Request) (int64, error) {
	cookie, err := r.Cookie("access-token")
	if err != nil || cookie.Value == "" {
		return 0, ErrUnauthorized
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (any, error) {
		method, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok || method.Alg() != v.algorithm {
			return nil, fmt.Errorf("unexpected JWT signing method")
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{v.algorithm}))
	if err != nil || !token.Valid {
		return 0, ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrUnauthorized
	}
	user_id, ok := claims["user_id"]
	if !ok {
		return 0, ErrUnauthorized
	}
	switch id := user_id.(type) {
	case float64:
		return int64(id), nil
	case string:
		parsed, parseErr := strconv.ParseInt(id, 10, 64)
		if parseErr == nil {
			return parsed, nil
		}
	}
	return 0, ErrUnauthorized
}
