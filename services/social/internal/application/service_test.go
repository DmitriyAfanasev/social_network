package application

import (
	"context"
	"testing"
	"time"

	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/social/internal/domain"
	"general-project/social/internal/ports"
)

type fakeBlockRepository struct {
	blocked     bool
	blocks      []string
	outboxEvent ports.OutboxEvent
}

func (f *fakeBlockRepository) IsBlocked(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return f.blocked, nil
}

func (f *fakeBlockRepository) Block(_ context.Context, blockerID uuid.UUID, blockedID uuid.UUID) error {
	f.blocks = append(f.blocks, blockerID.String()+":"+blockedID.String())
	return nil
}

func (f *fakeBlockRepository) BlockWithOutbox(ctx context.Context, blockerID uuid.UUID, blockedID uuid.UUID, event ports.OutboxEvent) error {
	f.outboxEvent = event
	return f.Block(ctx, blockerID, blockedID)
}

func (f *fakeBlockRepository) Unblock(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}

type fakeFriendshipRepository struct {
	createdFirst  uuid.UUID
	createdSecond uuid.UUID
	subscribed    map[string]bool
	outboxEvent   ports.OutboxEvent
	requests      map[uuid.UUID]domain.FriendRequest
}

func (f *fakeFriendshipRepository) IsFriend(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (f *fakeFriendshipRepository) IsSubscribed(_ context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error) {
	return f.subscribed[subscriberID.String()+":"+targetID.String()], nil
}

func (f *fakeFriendshipRepository) ListFriends(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeFriendshipRepository) ListSubscribers(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeFriendshipRepository) ListSubscriptions(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (f *fakeFriendshipRepository) CreateFriendRequest(_ context.Context, request domain.FriendRequest, event *ports.OutboxEvent) (domain.FriendRequest, bool, error) {
	if f.requests == nil {
		f.requests = make(map[uuid.UUID]domain.FriendRequest)
	}
	for _, existing := range f.requests {
		if existing.IsPending() && existing.SenderID == request.SenderID && existing.RecipientID == request.RecipientID {
			return existing, false, nil
		}
		if existing.IsPending() && existing.SenderID == request.RecipientID && existing.RecipientID == request.SenderID {
			return domain.FriendRequest{}, false, ports.ErrFriendRequestState
		}
	}
	f.requests[request.ID] = request
	if event != nil {
		f.outboxEvent = *event
	}
	return request, true, nil
}

func (f *fakeFriendshipRepository) GetFriendRequest(_ context.Context, requestID uuid.UUID) (domain.FriendRequest, error) {
	request, ok := f.requests[requestID]
	if !ok {
		return domain.FriendRequest{}, ports.ErrFriendRequestNotFound
	}
	return request, nil
}

func (f *fakeFriendshipRepository) ListFriendRequests(_ context.Context, userID uuid.UUID, incoming bool) ([]domain.FriendRequest, error) {
	result := make([]domain.FriendRequest, 0)
	for _, request := range f.requests {
		if !request.IsPending() || (incoming && request.RecipientID != userID) || (!incoming && request.SenderID != userID) {
			continue
		}
		result = append(result, request)
	}
	return result, nil
}

func (f *fakeFriendshipRepository) TransitionFriendRequest(_ context.Context, requestID uuid.UUID, actorID uuid.UUID, status domain.FriendRequestStatus, event *ports.OutboxEvent) (domain.FriendRequest, error) {
	request, ok := f.requests[requestID]
	if !ok {
		return domain.FriendRequest{}, ports.ErrFriendRequestNotFound
	}
	if !request.IsPending() {
		return domain.FriendRequest{}, ports.ErrFriendRequestState
	}
	if (status == domain.FriendRequestAccepted || status == domain.FriendRequestDeclined) && request.RecipientID != actorID {
		return domain.FriendRequest{}, ports.ErrFriendRequestForbidden
	}
	if status == domain.FriendRequestCancelled && request.SenderID != actorID {
		return domain.FriendRequest{}, ports.ErrFriendRequestForbidden
	}
	now := time.Now().UTC()
	request.Status = status
	request.RespondedAt = &now
	f.requests[requestID] = request
	if event != nil {
		f.outboxEvent = *event
	}
	return request, nil
}

func (f *fakeFriendshipRepository) CreateFriendship(_ context.Context, first uuid.UUID, second uuid.UUID) error {
	f.createdFirst = first
	f.createdSecond = second
	return nil
}

func (f *fakeFriendshipRepository) CreateFriendshipWithOutbox(ctx context.Context, first uuid.UUID, second uuid.UUID, event ports.OutboxEvent) error {
	f.outboxEvent = event
	return f.CreateFriendship(ctx, first, second)
}

func (f *fakeFriendshipRepository) RemoveFriendship(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}

func (f *fakeFriendshipRepository) Subscribe(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (f *fakeFriendshipRepository) SubscribeWithOutbox(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID, event ports.OutboxEvent) error {
	f.outboxEvent = event
	return f.Subscribe(ctx, subscriberID, targetID)
}

func (f *fakeFriendshipRepository) Unsubscribe(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}

func TestSocialServiceRejectsSelfBlock(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	service := NewSocialService(&fakeBlockRepository{}, &fakeFriendshipRepository{})

	_, err := service.Block(context.Background(), userID, userID)

	require.ErrorIs(t, err, ErrValidation)
}

func TestSocialServiceWritesBlockOutboxEvent(t *testing.T) {
	t.Parallel()

	blocks := &fakeBlockRepository{}
	actorID := uuid.New()
	targetID := uuid.New()
	service := NewSocialServiceWithOutbox(blocks, &fakeFriendshipRepository{}, nil, fakeSocialOutbox{})

	_, err := service.Block(context.Background(), actorID, targetID)

	require.NoError(t, err)
	require.Equal(t, "social.block.created", blocks.outboxEvent.EventType)
	require.Equal(t, targetID, *blocks.outboxEvent.AggregateID)
	require.JSONEq(t, `{"blocker_id":"`+actorID.String()+`","blocked_id":"`+targetID.String()+`"}`, string(blocks.outboxEvent.Payload))
}

func TestSocialServiceCreatesFriendRequest(t *testing.T) {
	t.Parallel()

	first := uuid.New()
	second := uuid.New()
	blocks := &fakeBlockRepository{}
	friendships := &fakeFriendshipRepository{}
	service := NewSocialService(blocks, friendships)

	result, err := service.AddFriend(context.Background(), second, first)

	require.NoError(t, err)
	require.Equal(t, string(domain.FriendRequestPending), result.State)
	request, ok := friendships.requests[result.RequestID]
	require.True(t, ok)
	require.Equal(t, second, request.SenderID)
	require.Equal(t, first, request.RecipientID)
	require.Equal(t, domain.FriendRequestPending, request.Status)
}

func TestSocialServiceRejectsFriendshipWhenBlocked(t *testing.T) {
	t.Parallel()

	blocks := &fakeBlockRepository{blocked: true}
	service := NewSocialService(blocks, &fakeFriendshipRepository{})

	_, err := service.AddFriend(context.Background(), uuid.New(), uuid.New())

	require.ErrorIs(t, err, ErrBlocked)
}

func TestSocialServiceCreatesFriendRequestOutboxEvent(t *testing.T) {
	t.Parallel()

	first := uuid.New()
	second := uuid.New()
	friendships := &fakeFriendshipRepository{}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, friendships, nil, fakeSocialOutbox{})

	_, err := service.AddFriend(context.Background(), first, second)

	require.NoError(t, err)
	require.Equal(t, "social.friend_request.created", friendships.outboxEvent.EventType)
	require.JSONEq(t, `{"request_id":"`+friendships.outboxEvent.AggregateID.String()+`","actor_id":"`+first.String()+`","target_id":"`+second.String()+`"}`, string(friendships.outboxEvent.Payload))
}

func TestSocialServiceAcceptsFriendRequestAndWritesFriendshipEvent(t *testing.T) {
	t.Parallel()

	senderID, recipientID := uuid.New(), uuid.New()
	requestID := uuid.New()
	friendships := &fakeFriendshipRepository{requests: map[uuid.UUID]domain.FriendRequest{
		requestID: {ID: requestID, SenderID: senderID, RecipientID: recipientID, Status: domain.FriendRequestPending, CreatedAt: time.Now().UTC()},
	}}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, friendships, nil, fakeSocialOutbox{})

	result, err := service.AcceptFriendRequest(context.Background(), recipientID, requestID)

	require.NoError(t, err)
	require.Equal(t, string(domain.FriendRequestAccepted), result.State)
	require.Equal(t, domain.FriendRequestAccepted, friendships.requests[requestID].Status)
	require.Equal(t, "social.friendship.created", friendships.outboxEvent.EventType)
	require.JSONEq(t, `{"request_id":"`+requestID.String()+`","actor_id":"`+recipientID.String()+`","target_id":"`+senderID.String()+`"}`, string(friendships.outboxEvent.Payload))
}

func TestSocialServiceRejectsFriendRequestTransitionByWrongUser(t *testing.T) {
	t.Parallel()

	senderID, recipientID := uuid.New(), uuid.New()
	requestID := uuid.New()
	friendships := &fakeFriendshipRepository{requests: map[uuid.UUID]domain.FriendRequest{
		requestID: {ID: requestID, SenderID: senderID, RecipientID: recipientID, Status: domain.FriendRequestPending, CreatedAt: time.Now().UTC()},
	}}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, friendships, nil, fakeSocialOutbox{})

	_, err := service.AcceptFriendRequest(context.Background(), senderID, requestID)

	require.ErrorIs(t, err, ErrFriendRequestForbidden)
}

func TestSocialServiceDeclinesAndCancelsFriendRequests(t *testing.T) {
	t.Parallel()

	senderID, recipientID := uuid.New(), uuid.New()
	declineID, cancelID := uuid.New(), uuid.New()
	friendships := &fakeFriendshipRepository{requests: map[uuid.UUID]domain.FriendRequest{
		declineID: {ID: declineID, SenderID: senderID, RecipientID: recipientID, Status: domain.FriendRequestPending, CreatedAt: time.Now().UTC()},
		cancelID:  {ID: cancelID, SenderID: senderID, RecipientID: recipientID, Status: domain.FriendRequestPending, CreatedAt: time.Now().UTC()},
	}}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, friendships, nil, fakeSocialOutbox{})

	declined, err := service.DeclineFriendRequest(context.Background(), recipientID, declineID)
	require.NoError(t, err)
	require.Equal(t, string(domain.FriendRequestDeclined), declined.State)

	cancelled, err := service.CancelFriendRequest(context.Background(), senderID, cancelID)
	require.NoError(t, err)
	require.Equal(t, string(domain.FriendRequestCancelled), cancelled.State)
}

func TestSocialServiceCreatesSubscription(t *testing.T) {
	t.Parallel()

	blocks := &fakeBlockRepository{}
	friendships := &fakeFriendshipRepository{}
	service := NewSocialService(blocks, friendships)
	actorID := uuid.New()
	targetID := uuid.New()

	result, err := service.Subscribe(context.Background(), actorID, targetID)

	require.NoError(t, err)
	require.Equal(t, "subscribed", result.State)
}

type fakeSocialOutbox struct{}

func (fakeSocialOutbox) Claim(context.Context, int, time.Time) ([]ports.OutboxEvent, error) {
	return nil, nil
}

func (fakeSocialOutbox) MarkPublished(context.Context, uuid.UUID, time.Time) error { return nil }

func (fakeSocialOutbox) MarkFailed(context.Context, uuid.UUID, time.Time, string) error { return nil }

func TestSocialServiceCreatesSubscriptionOutboxEvent(t *testing.T) {
	t.Parallel()

	blocks := &fakeBlockRepository{}
	friendships := &fakeFriendshipRepository{}
	service := NewSocialServiceWithOutbox(blocks, friendships, nil, fakeSocialOutbox{})
	actorID := uuid.New()
	targetID := uuid.New()

	_, err := service.Subscribe(context.Background(), actorID, targetID)

	require.NoError(t, err)
	require.Equal(t, "social.subscription.created", friendships.outboxEvent.EventType)
	require.Equal(t, targetID, *friendships.outboxEvent.AggregateID)
	require.JSONEq(t, `{"subscriber_id":"`+actorID.String()+`","target_id":"`+targetID.String()+`"}`, string(friendships.outboxEvent.Payload))
}
