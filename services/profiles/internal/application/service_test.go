package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/profiles/internal/domain"
	"general-project/profiles/internal/ports"
)

type fakeProfileRepository struct {
	byHandle map[string]domain.Profile
	byUser   map[uuid.UUID]domain.Profile
}

func (f *fakeProfileRepository) FindByHandle(_ context.Context, handle string) (domain.Profile, error) {
	profile, ok := f.byHandle[handle]
	if !ok {
		return domain.Profile{}, ports.ErrNotFound
	}
	return profile, nil
}

func (f *fakeProfileRepository) SearchByHandle(_ context.Context, query string, limit int) ([]domain.Profile, error) {
	result := make([]domain.Profile, 0, limit)
	for handle, profile := range f.byHandle {
		if len(result) == limit {
			break
		}
		if strings.HasPrefix(handle, query) {
			result = append(result, profile)
		}
	}
	return result, nil
}

func (f *fakeProfileRepository) FindByUserID(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
	profile, ok := f.byUser[userID]
	if !ok {
		return domain.Profile{}, ports.ErrNotFound
	}
	return profile, nil
}

func (f *fakeProfileRepository) EnsureByUserID(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
	profile, ok := f.byUser[userID]
	if !ok {
		profile = domain.Profile{UserID: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		f.byUser[userID] = profile
	}
	return profile, nil
}

func (f *fakeProfileRepository) SetHandle(_ context.Context, userID uuid.UUID, handle string) (domain.Profile, error) {
	if _, ok := f.byHandle[handle]; ok {
		return domain.Profile{}, ports.ErrAlreadyExists
	}
	profile := f.byUser[userID]
	profile.UserID = userID
	profile.Handle = &handle
	profile.UpdatedAt = time.Now()
	f.byUser[userID] = profile
	f.byHandle[handle] = profile
	return profile, nil
}

func (f *fakeProfileRepository) UpdatePublicProfile(_ context.Context, userID uuid.UUID, bio string) (domain.Profile, error) {
	profile, ok := f.byUser[userID]
	if !ok {
		return domain.Profile{}, ports.ErrNotFound
	}
	profile.Bio = bio
	profile.UpdatedAt = time.Now()
	f.byUser[userID] = profile
	return profile, nil
}

func (f *fakeProfileRepository) UpdateProfileDetails(_ context.Context, userID uuid.UUID, details domain.ProfileDetails) (domain.Profile, error) {
	profile, ok := f.byUser[userID]
	if !ok {
		return domain.Profile{}, ports.ErrNotFound
	}
	profile.Details = details
	profile.UpdatedAt = time.Now()
	f.byUser[userID] = profile
	return profile, nil
}

func (f *fakeProfileRepository) UpdateProfilePrivacy(_ context.Context, userID uuid.UUID, privacy domain.ProfilePrivacy) (domain.Profile, error) {
	profile, ok := f.byUser[userID]
	if !ok {
		return domain.Profile{}, ports.ErrNotFound
	}
	profile.Privacy = privacy
	profile.UpdatedAt = time.Now()
	f.byUser[userID] = profile
	return profile, nil
}

func (f *fakeProfileRepository) UpdateAvatar(_ context.Context, userID uuid.UUID, avatarURL string) (domain.Profile, error) {
	profile, ok := f.byUser[userID]
	if !ok {
		return domain.Profile{}, ports.ErrNotFound
	}
	profile.AvatarURL = avatarURL
	f.byUser[userID] = profile
	return profile, nil
}

type fakeProfileCache struct {
	values map[string][]byte
}

type fakeRelationshipReader struct {
	access ports.RelationshipAccess
}

func (f fakeRelationshipReader) GetRelationship(context.Context, uuid.UUID, uuid.UUID) (ports.RelationshipAccess, error) {
	return f.access, nil
}

func (f *fakeProfileCache) Get(_ context.Context, key string) ([]byte, error) {
	return f.values[key], nil
}

func (f *fakeProfileCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.values[key] = value
	return nil
}

func (f *fakeProfileCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func TestProfileServiceSetHandleNormalizesAndCachesInvalidation(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	oldHandle := "old_name"
	repository := &fakeProfileRepository{
		byHandle: map[string]domain.Profile{
			oldHandle: {UserID: userID, Handle: &oldHandle},
		},
		byUser: map[uuid.UUID]domain.Profile{
			userID: {UserID: userID, Handle: &oldHandle},
		},
	}
	profileCache := &fakeProfileCache{values: map[string][]byte{
		profileCacheKey(oldHandle): []byte(`stale`),
	}}
	service := NewProfileService(repository, profileCache)

	result, err := service.SetHandle(context.Background(), userID, " New_Name ")

	require.NoError(t, err)
	require.Equal(t, "new_name", *result.Handle)
	require.Nil(t, profileCache.values[profileCacheKey(oldHandle)])
	require.Nil(t, profileCache.values[profileCacheKey("new_name")])
}

func TestProfileServiceGetByHandleUsesCache(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	handle := "alice"
	profile := ProfileDTO{UserID: userID, Handle: &handle}
	encoded, err := json.Marshal(profile)
	require.NoError(t, err)

	service := NewProfileService(
		&fakeProfileRepository{byHandle: map[string]domain.Profile{}},
		&fakeProfileCache{values: map[string][]byte{profileCacheKey(handle): encoded}},
	)

	result, err := service.GetByHandle(context.Background(), " Alice ")

	require.NoError(t, err)
	require.Equal(t, profile.UserID, result.UserID)
	require.Equal(t, profile.UserID, result.UserID)
}

func TestProfileServiceRejectsInvalidHandle(t *testing.T) {
	t.Parallel()

	service := NewProfileService(&fakeProfileRepository{}, nil)

	_, err := service.SetHandle(context.Background(), uuid.New(), "bad handle")

	require.ErrorIs(t, err, ErrValidation)
}

func TestProfileServiceFiltersDetailsByPrivacy(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	viewerID := uuid.New()
	repository := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{
		ownerID: {
			UserID:  ownerID,
			Details: domain.ProfileDetails{PhoneNumber: "+70000000000", Country: "Россия", City: "Самара", Status: "На связи"},
			Privacy: domain.ProfilePrivacy{PhoneVisibility: "friends", LocationVisibility: "friends_of_friends", StatusVisibility: "nobody"},
		},
	}}
	service := NewProfileService(repository, nil, fakeRelationshipReader{access: ports.RelationshipAccess{IsFriendOfFriend: true}})

	result, err := service.GetByUserID(context.Background(), ownerID, viewerID)

	require.NoError(t, err)
	require.Empty(t, result.Details.PhoneNumber)
	require.Equal(t, "Россия", result.Details.Country)
	require.Empty(t, result.Details.Status)
}

func TestProfileServiceEnsuresInitialProfile(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repository := &fakeProfileRepository{byUser: map[uuid.UUID]domain.Profile{}}
	service := NewProfileService(repository, nil)

	err := service.EnsureProfile(context.Background(), userID)

	require.NoError(t, err)
	require.Equal(t, userID, repository.byUser[userID].UserID)
}

func TestProfileServiceUpdatesPublicFieldsAndInvalidatesHandleCache(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	handle := "alice"
	repository := &fakeProfileRepository{
		byUser: map[uuid.UUID]domain.Profile{
			userID: {UserID: userID, Handle: &handle},
		},
		byHandle: map[string]domain.Profile{},
	}
	profileCache := &fakeProfileCache{values: map[string][]byte{
		profileCacheKey(handle): []byte(`stale`),
	}}
	service := NewProfileService(repository, profileCache)

	result, err := service.UpdatePublicProfile(context.Background(), userID, UpdateProfileInput{
		Bio: "Открытое описание",
	})

	require.NoError(t, err)
	require.Equal(t, "Открытое описание", result.Bio)
	require.Nil(t, profileCache.values[profileCacheKey(handle)])
}
