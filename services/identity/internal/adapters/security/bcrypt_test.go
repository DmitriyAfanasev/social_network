package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher(4)
	hash, err := hasher.Hash("correct horse")
	require.NoError(t, err)
	require.NotEqual(t, "correct horse", hash)
	require.NoError(t, hasher.Compare("correct horse", hash))
	require.Error(t, hasher.Compare("wrong", hash))
}

func TestBcryptHasherUsesDefaultForTooLowCost(t *testing.T) {
	require.Equal(t, 10, NewBcryptHasher(1).cost)
}
