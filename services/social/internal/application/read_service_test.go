package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/social/internal/domain"
	"general-project/social/internal/ports"
)

type graphFriendshipRepository struct {
	friends       map[uuid.UUID][]uuid.UUID
	subscribed    map[string]bool
	listCallCount int
}

type fakeProfilePolicy struct {
	friendsVisibility   string
	friendRequestPolicy string
}

func (f fakeProfilePolicy) GetFriendRequestPolicy(_ context.Context, _ uuid.UUID) (string, error) {
	if f.friendRequestPolicy != "" {
		return f.friendRequestPolicy, nil
	}
	return "everyone", nil
}

func (f fakeProfilePolicy) GetFriendsVisibility(_ context.Context, _ uuid.UUID) (string, error) {
	return f.friendsVisibility, nil
}

func (r *graphFriendshipRepository) IsFriend(_ context.Context, userID uuid.UUID, friendID uuid.UUID) (bool, error) {
	first, second := userID, friendID
	if strings.Compare(second.String(), first.String()) < 0 {
		first, second = second, first
	}
	for _, candidate := range r.friends[first] {
		if candidate == second {
			return true, nil
		}
	}
	return false, nil
}

func (r *graphFriendshipRepository) IsSubscribed(_ context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error) {
	return r.subscribed[subscriberID.String()+":"+targetID.String()], nil
}

func (r *graphFriendshipRepository) CreateFriendship(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (r *graphFriendshipRepository) CreateFriendshipWithOutbox(ctx context.Context, first uuid.UUID, second uuid.UUID, _ ports.OutboxEvent) error {
	return r.CreateFriendship(ctx, first, second)
}

func (r *graphFriendshipRepository) RemoveFriendship(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}

func (r *graphFriendshipRepository) Subscribe(_ context.Context, subscriberID uuid.UUID, targetID uuid.UUID) error {
	r.subscribed[subscriberID.String()+":"+targetID.String()] = true
	return nil
}

func (r *graphFriendshipRepository) SubscribeWithOutbox(ctx context.Context, subscriberID uuid.UUID, targetID uuid.UUID, _ ports.OutboxEvent) error {
	return r.Subscribe(ctx, subscriberID, targetID)
}

func (r *graphFriendshipRepository) Unsubscribe(_ context.Context, subscriberID uuid.UUID, targetID uuid.UUID) (bool, error) {
	key := subscriberID.String() + ":" + targetID.String()
	wasSubscribed := r.subscribed[key]
	delete(r.subscribed, key)
	return wasSubscribed, nil
}

func (r *graphFriendshipRepository) ListFriends(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	r.listCallCount++
	return append([]uuid.UUID(nil), r.friends[userID]...), nil
}

func (r *graphFriendshipRepository) ListSubscribers(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *graphFriendshipRepository) ListSubscriptions(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *graphFriendshipRepository) CreateFriendRequest(_ context.Context, request domain.FriendRequest, _ *ports.OutboxEvent) (domain.FriendRequest, bool, error) {
	return request, true, nil
}

func (r *graphFriendshipRepository) GetFriendRequest(_ context.Context, _ uuid.UUID) (domain.FriendRequest, error) {
	return domain.FriendRequest{}, ports.ErrFriendRequestNotFound
}

func (r *graphFriendshipRepository) ListFriendRequests(_ context.Context, _ uuid.UUID, _ bool) ([]domain.FriendRequest, error) {
	return nil, nil
}

func (r *graphFriendshipRepository) TransitionFriendRequest(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.FriendRequestStatus, _ *ports.OutboxEvent) (domain.FriendRequest, error) {
	return domain.FriendRequest{}, ports.ErrFriendRequestNotFound
}

type fakeSocialCache struct {
	values map[string][]byte
}

func (c *fakeSocialCache) Get(_ context.Context, key string) ([]byte, error) {
	return c.values[key], nil
}

func (c *fakeSocialCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.values[key] = value
	return nil
}

func (c *fakeSocialCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(c.values, key)
	}
	return nil
}

func TestSocialServiceRecommendationsRankByCommonFriendsAndUseCache(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	firstFriend := uuid.New()
	secondFriend := uuid.New()
	candidate := uuid.New()
	repository := &graphFriendshipRepository{
		friends: map[uuid.UUID][]uuid.UUID{
			userID:       {firstFriend, secondFriend},
			firstFriend:  {userID, candidate},
			secondFriend: {userID, candidate},
		},
		subscribed: map[string]bool{},
	}
	cache := &fakeSocialCache{values: map[string][]byte{}}
	service := NewSocialService(&fakeBlockRepository{}, repository, cache)

	result, err := service.GetRecommendations(context.Background(), userID, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, candidate, result[0].UserID)
	require.Equal(t, 2, result[0].CommonFriends)
	callCount := repository.listCallCount

	result, err = service.GetRecommendations(context.Background(), userID, 10)
	require.NoError(t, err)
	require.Equal(t, candidate, result[0].UserID)
	require.Equal(t, callCount, repository.listCallCount)
}

func TestSocialServicePublicFriendsUsesTargetOwnerAndPrivacy(t *testing.T) {
	viewerID := uuid.New()
	targetID := uuid.New()
	targetFriendID := uuid.New()
	viewerFriendID := uuid.New()
	repository := &graphFriendshipRepository{
		friends: map[uuid.UUID][]uuid.UUID{
			targetID: {targetFriendID},
			viewerID: {viewerFriendID},
		},
		subscribed: map[string]bool{},
	}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, repository, nil, fakeSocialOutbox{}, fakeProfilePolicy{friendsVisibility: "everyone"})

	result, err := service.GetPublicFriends(context.Background(), viewerID, targetID)
	require.NoError(t, err)
	require.True(t, result.Visible)
	require.Equal(t, []uuid.UUID{targetFriendID}, result.Friends)
}

func TestSocialServicePublicFriendsHidesRestrictedTargetList(t *testing.T) {
	viewerID := uuid.New()
	targetID := uuid.New()
	repository := &graphFriendshipRepository{
		friends:    map[uuid.UUID][]uuid.UUID{targetID: {uuid.New()}},
		subscribed: map[string]bool{},
	}
	service := NewSocialServiceWithOutbox(&fakeBlockRepository{}, repository, nil, fakeSocialOutbox{}, fakeProfilePolicy{friendsVisibility: "friends"})

	result, err := service.GetPublicFriends(context.Background(), viewerID, targetID)
	require.NoError(t, err)
	require.False(t, result.Visible)
	require.Empty(t, result.Friends)
}

func TestSocialServiceInvalidatesRelationshipCacheAfterMutation(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	targetID := uuid.New()
	repository := &graphFriendshipRepository{
		friends:    map[uuid.UUID][]uuid.UUID{userID: {}},
		subscribed: map[string]bool{},
	}
	cache := &fakeSocialCache{values: map[string][]byte{}}
	service := NewSocialService(&fakeBlockRepository{}, repository, cache)

	_, err := service.GetRelationships(context.Background(), userID)
	require.NoError(t, err)
	require.Contains(t, cache.values, relationshipsCacheKey(userID))

	_, err = service.Subscribe(context.Background(), userID, targetID)
	require.NoError(t, err)
	require.NotContains(t, cache.values, relationshipsCacheKey(userID))

	payload, err := json.Marshal(RelationshipsDTO{Friends: []uuid.UUID{targetID}})
	require.NoError(t, err)
	cache.values[relationshipsCacheKey(userID)] = payload
	result, err := service.GetRelationships(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{targetID}, result.Friends)
}
