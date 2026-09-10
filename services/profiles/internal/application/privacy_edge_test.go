package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
)

func TestProfileServiceReadsAndUpdatesPrivacy(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repository := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{
		userID: {UserID: userID, Privacy: domain.ProfilePrivacy{ProfileVisibility: "everyone"}},
	}}
	service := NewProfileService(repository, nil)
	privacy := domain.ProfilePrivacy{
		ProfileVisibility:   "friends",
		FriendRequestPolicy: "nobody",
		MessagePolicy:       "friends_of_friends",
		PhoneVisibility:     "nobody",
		BirthDateVisibility: "friends",
		GenderVisibility:    "everyone",
		LocationVisibility:  "friends",
		StatusVisibility:    "everyone",
		FriendsVisibility:   "friends",
		PostsVisibility:     "everyone",
		MusicVisibility:     "friends",
	}

	before, err := service.GetPrivacy(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, "everyone", before.ProfileVisibility)

	updated, err := service.UpdatePrivacy(context.Background(), userID, UpdatePrivacyInput{Privacy: privacy})

	require.NoError(t, err)
	require.Equal(t, toProfileDTO(domain.Profile{Privacy: privacy}).Privacy, updated)
	require.Equal(t, privacy, repository.byUser[userID].Privacy)
}

func TestProfileServiceRejectsInvalidPrivacyAndDetails(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repository := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{userID: {UserID: userID}}}
	service := NewProfileService(repository, nil)

	_, err := service.UpdatePrivacy(context.Background(), userID, UpdatePrivacyInput{Privacy: domain.ProfilePrivacy{ProfileVisibility: "invalid"}})
	require.ErrorIs(t, err, ErrValidation)

	_, err = service.UpdateProfile(context.Background(), userID, UpdateProfileInput{Details: domain.ProfileDetails{FirstName: string(make([]rune, 81))}})
	require.ErrorIs(t, err, ErrValidation)
}

func TestProfileServiceSearchValidatesQueryAndLimit(t *testing.T) {
	t.Parallel()

	service := NewProfileService(&fakeProfileRepository{}, nil)
	tests := []struct {
		name  string
		query string
		limit int
	}{
		{name: "empty query", query: "", limit: 10},
		{name: "zero limit", query: "alice", limit: 0},
		{name: "limit above maximum", query: "alice", limit: 51},
		{name: "query above maximum", query: string(make([]rune, 33)), limit: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.SearchByHandle(context.Background(), tt.query, tt.limit)

			require.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestProfileServicePropagatesRelationshipAccessErrorsAsPrivateView(t *testing.T) {
	t.Parallel()

	ownerID, viewerID := uuid.New(), uuid.New()
	handle := "alice"
	repository := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{
		ownerID: {UserID: ownerID, Handle: &handle, Bio: "private", Privacy: domain.ProfilePrivacy{ProfileVisibility: "friends"}},
	}}
	service := NewProfileService(repository, nil, relationshipErrorReader{})

	result, err := service.GetByUserID(context.Background(), ownerID, viewerID)

	require.NoError(t, err)
	require.Empty(t, result.Bio)
}

type relationshipErrorReader struct{}

func (relationshipErrorReader) GetRelationship(context.Context, uuid.UUID, uuid.UUID) (ports.RelationshipAccess, error) {
	return ports.RelationshipAccess{}, ports.ErrNotFound
}
