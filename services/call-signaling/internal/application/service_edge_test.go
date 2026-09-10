package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/call-signaling/internal/domain"
)

func TestCallServiceRejectsUnauthorizedStartAndInvalidTransition(t *testing.T) {
	t.Parallel()

	conversationID, callerID, calleeID := uuid.New(), uuid.New(), uuid.New()
	service := NewService(&fakeSessionStore{}, &fakeAuthorizer{canStart: false})

	_, err := service.Start(context.Background(), conversationID, callerID, calleeID, domain.Audio)
	require.ErrorIs(t, err, domain.ErrNotParticipant)

	store := &fakeSessionStore{}
	service = NewService(store, &fakeAuthorizer{canStart: true})
	session, err := service.Start(context.Background(), conversationID, callerID, calleeID, domain.Audio)
	require.NoError(t, err)

	_, err = service.Transition(context.Background(), conversationID, callerID, session.CallID, "unknown")
	require.ErrorIs(t, err, domain.ErrInvalidState)
}

func TestCallServiceRejectsKeepAliveUntilCallIsActive(t *testing.T) {
	t.Parallel()

	conversationID, callerID, calleeID := uuid.New(), uuid.New(), uuid.New()
	store := &fakeSessionStore{}
	service := NewService(store, &fakeAuthorizer{canStart: true})
	session, err := service.Start(context.Background(), conversationID, callerID, calleeID, domain.Audio)
	require.NoError(t, err)

	err = service.KeepAlive(context.Background(), conversationID, callerID, session.CallID)

	require.ErrorIs(t, err, ErrCallNotActive)
	require.False(t, store.refreshed)
}
