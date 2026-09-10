package token

import (
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWTIssuer(t *testing.T) {
	userID := uuid.New()
	issuer := NewJWTIssuer("secret", time.Minute)
	raw, err := issuer.IssueAccess(userID)
	require.NoError(t, err)
	parsed, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		require.Equal(t, jwt.SigningMethodHS256.Alg(), token.Method.Alg())
		return []byte("secret"), nil
	})
	require.NoError(t, err)
	require.Equal(t, userID.String(), parsed.Claims.(*jwt.RegisteredClaims).Subject)
}

func TestJWTIssuerRejectsInvalidConfiguration(t *testing.T) {
	for _, issuer := range []*JWTIssuer{NewJWTIssuer("", time.Minute), NewJWTIssuer("secret", 0)} {
		_, err := issuer.IssueAccess(uuid.New())
		require.Error(t, err)
	}
}
