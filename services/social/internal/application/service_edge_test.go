package application

import (
	"context"
	"testing"
	"time"

	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/social/internal/domain"
)

func TestSocialServiceListsIncomingAndOutgoingFriendRequests(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	incoming := domain.FriendRequest{ID: uuid.New(), SenderID: uuid.New(), RecipientID: userID, Status: domain.FriendRequestPending, CreatedAt: time.Now().UTC()}
	outgoing := domain.FriendRequest{ID: uuid.New(), SenderID: userID, RecipientID: uuid.New(), Status: domain.FriendRequestPending, CreatedAt: time.Now().UTC()}
	friendships := &fakeFriendshipRepository{requests: map[uuid.UUID]domain.FriendRequest{incoming.ID: incoming, outgoing.ID: outgoing}}
	service := NewSocialService(&fakeBlockRepository{}, friendships)

	incomingResult, err := service.ListFriendRequests(context.Background(), userID, true)
	require.NoError(t, err)
	require.Equal(t, []FriendRequestDTO{mapFriendRequest(incoming)}, incomingResult)

	outgoingResult, err := service.ListFriendRequests(context.Background(), userID, false)
	require.NoError(t, err)
	require.Equal(t, []FriendRequestDTO{mapFriendRequest(outgoing)}, outgoingResult)
}

func TestSocialServiceUnblocksAndRemovesFriend(t *testing.T) {
	t.Parallel()

	actorID, targetID := uuid.New(), uuid.New()
	friendships := &fakeFriendshipRepository{}
	service := NewSocialService(&fakeBlockRepository{}, friendships)

	unblocked, err := service.Unblock(context.Background(), actorID, targetID)
	require.NoError(t, err)
	require.Equal(t, "unblocked", unblocked.State)

	removed, err := service.RemoveFriend(context.Background(), actorID, targetID)
	require.NoError(t, err)
	require.Equal(t, "not_friends", removed.State)
	// Pair normalizes the order used by the repository.
	first, second := domain.Pair(actorID, targetID)
	require.Equal(t, first, friendships.removedFirst)
	require.Equal(t, second, friendships.removedSecond)
}

func TestSocialServiceUnsubscribes(t *testing.T) {
	t.Parallel()

	actorID, targetID := uuid.New(), uuid.New()
	service := NewSocialService(&fakeBlockRepository{}, &fakeFriendshipRepository{})

	result, err := service.Unsubscribe(context.Background(), actorID, targetID)

	require.NoError(t, err)
	require.Equal(t, "not_subscribed", result.State)
}

func TestSocialServiceAllowsFriendRequestFromFriendOfFriend(t *testing.T) {
	t.Parallel()

	actorID, targetID, commonFriendID := uuid.New(), uuid.New(), uuid.New()
	friendships := &fakeFriendshipRepository{
		friends:   []uuid.UUID{commonFriendID},
		isFriends: map[string]bool{commonFriendID.String() + ":" + targetID.String(): true},
	}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, friendships, nil, nil, fakeProfilePolicy{friendRequestPolicy: "friends_of_friends"})

	result, err := service.SendFriendRequest(context.Background(), actorID, targetID)

	require.NoError(t, err)
	require.Equal(t, string(domain.FriendRequestPending), result.State)
}
