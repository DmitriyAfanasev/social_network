package domain

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestNewSessionValidationAndLifecycle(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name           string
		caller, callee uuid.UUID
		kind           CallType
	}{
		{"missing caller", uuid.Nil(), b, Audio}, {"missing callee", a, uuid.Nil(), Audio}, {"self call", a, a, Audio}, {"invalid type", a, b, CallType("screen")},
	} {
		t.Run(tc.name, func(t *testing.T) { _, err := NewSession(tc.caller, tc.callee, tc.kind); require.Error(t, err) })
	}
	s, err := NewSession(a, b, Video)
	require.NoError(t, err)
	require.Equal(t, Ringing, s.Status)
	require.True(t, s.Includes(a))
	require.False(t, s.Includes(uuid.New()))
	_, err = s.Accept(a)
	require.ErrorIs(t, err, ErrOnlyCallee)
	s, err = s.Accept(b)
	require.NoError(t, err)
	require.Equal(t, Active, s.Status)
	_, err = s.Accept(b)
	require.ErrorIs(t, err, ErrInvalidState)
	s, err = s.End(a)
	require.NoError(t, err)
	require.Equal(t, Ended, s.Status)
	_, err = s.End(uuid.New())
	require.ErrorIs(t, err, ErrNotParticipant)
}

func TestRejectAndEndInvalidStates(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	s, err := NewSession(a, b, Audio)
	require.NoError(t, err)
	_, err = s.Reject(a)
	require.ErrorIs(t, err, ErrOnlyCallee)
	s, err = s.Reject(b)
	require.NoError(t, err)
	require.Equal(t, Rejected, s.Status)
	_, err = s.End(a)
	require.ErrorIs(t, err, ErrInvalidState)
}
