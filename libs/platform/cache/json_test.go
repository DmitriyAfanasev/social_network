package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	values map[string][]byte
}

func (f *fakeStore) Get(_ context.Context, key string) ([]byte, error) {
	return f.values[key], nil
}

func (f *fakeStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.values[key] = value
	return nil
}

func (f *fakeStore) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func TestJSONUsesVersionedNamespaceAndRoundTripsValues(t *testing.T) {
	store := &fakeStore{values: map[string][]byte{}}
	cache, err := NewJSON(store, "profiles", "v1")
	require.NoError(t, err)

	key := cache.Key("handle", "alice")
	require.Equal(t, "profiles:v1:handle:alice", key)
	require.NoError(t, cache.Set(context.Background(), key, map[string]string{"handle": "alice"}, time.Minute))

	var value map[string]string
	found, err := cache.Get(context.Background(), key, &value)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, map[string]string{"handle": "alice"}, value)
}

func TestJSONRejectsInvalidConfigurationAndTTL(t *testing.T) {
	store := &fakeStore{values: map[string][]byte{}}
	_, err := NewJSON(store, "", "v1")
	require.Error(t, err)

	cache, err := NewJSON(store, "profiles", "v1")
	require.NoError(t, err)
	require.Error(t, cache.Set(context.Background(), "profiles:v1:key", "value", 0))
}
