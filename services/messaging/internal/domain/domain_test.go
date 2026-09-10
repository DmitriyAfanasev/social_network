package domain

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestPairKeyIsOrderIndependent(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	require.Equal(t, PairKey(a, b), PairKey(b, a))
	require.Contains(t, PairKey(a, b), ":")
}

func BenchmarkPairKey(b *testing.B) {
	first, second := uuid.New(), uuid.New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = PairKey(first, second)
	}
}
