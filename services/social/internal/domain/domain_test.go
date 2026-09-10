package domain

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestFriendRequestPermissions(t *testing.T) {
	sender, recipient, other := uuid.New(), uuid.New(), uuid.New()
	r := FriendRequest{SenderID: sender, RecipientID: recipient, Status: FriendRequestPending}
	require.True(t, r.IsPending())
	require.True(t, r.CanBeAcceptedBy(recipient))
	require.True(t, r.CanBeDeclinedBy(recipient))
	require.True(t, r.CanBeCancelledBy(sender))
	require.False(t, r.CanBeAcceptedBy(other))
	r.Status = FriendRequestAccepted
	require.False(t, r.IsPending())
	require.False(t, r.CanBeCancelledBy(sender))
}

func TestPairIsStable(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	left, right := Pair(a, b)
	left2, right2 := Pair(b, a)
	require.Equal(t, left, left2)
	require.Equal(t, right, right2)
}

func BenchmarkPair(b *testing.B) {
	first, second := uuid.New(), uuid.New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Pair(first, second)
	}
}
