package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/call-signaling/internal/domain"
	"general-project/call-signaling/internal/ports"
)

type fakeSessionStore struct {
	session      domain.Session
	created      bool
	refreshed    bool
	transitioned string
}

func (f *fakeSessionStore) Create(_ context.Context, session domain.Session) error {
	f.session = session
	f.created = true
	return nil
}

func (f *fakeSessionStore) Get(_ context.Context, _ uuid.UUID) (domain.Session, error) {
	if !f.created {
		return domain.Session{}, ports.ErrNotFound
	}
	return f.session, nil
}

func (f *fakeSessionStore) Refresh(_ context.Context, _ uuid.UUID) error {
	f.refreshed = true
	return nil
}

func (f *fakeSessionStore) Transition(_ context.Context, _ uuid.UUID, userID uuid.UUID, action string) (domain.Session, error) {
	var err error
	switch action {
	case "accept":
		f.session, err = f.session.Accept(userID)
	case "reject":
		f.session, err = f.session.Reject(userID)
	case "end":
		f.session, err = f.session.End(userID)
	default:
		return domain.Session{}, domain.ErrInvalidState
	}
	if err == nil {
		f.transitioned = action
	}
	return f.session, err
}

type fakeAuthorizer struct {
	canStart bool
}

func (f *fakeAuthorizer) CanSubscribe(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (f *fakeAuthorizer) CanStart(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	if !f.canStart {
		return domain.ErrNotParticipant
	}
	return nil
}

func (f *fakeAuthorizer) CanUseCall(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}

func TestStartTransitionAndKeepAlive(t *testing.T) {
	caller, callee, conversation := uuid.New(), uuid.New(), uuid.New()
	store := &fakeSessionStore{}
	service := NewService(store, &fakeAuthorizer{canStart: true})

	session, err := service.Start(context.Background(), conversation, caller, callee, domain.Video)
	require.NoError(t, err)
	require.True(t, store.created)
	require.Equal(t, domain.Ringing, session.Status)

	active, err := service.Transition(context.Background(), conversation, callee, session.CallID, "accept")
	require.NoError(t, err)
	require.Equal(t, domain.Active, active.Status)
	require.Equal(t, "accept", store.transitioned)

	require.NoError(t, service.KeepAlive(context.Background(), conversation, caller, session.CallID))
	require.True(t, store.refreshed)
}

func TestSignalRequiresActiveCall(t *testing.T) {
	caller, callee, conversation := uuid.New(), uuid.New(), uuid.New()
	store := &fakeSessionStore{}
	service := NewService(store, &fakeAuthorizer{canStart: true})

	session, err := service.Start(context.Background(), conversation, caller, callee, domain.Audio)
	require.NoError(t, err)

	_, err = service.AuthorizeSignal(context.Background(), conversation, caller, session.CallID)
	require.ErrorIs(t, err, ErrCallNotActive)
}

func TestMissingSessionIsMappedToApplicationError(t *testing.T) {
	service := NewService(&fakeSessionStore{}, &fakeAuthorizer{canStart: true})

	_, err := service.AuthorizeSignal(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, ErrCallNotFound)
}
