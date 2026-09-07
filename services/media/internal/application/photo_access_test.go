package application

import (
	"context"
	"errors"
	"general-project/media/internal/domain"
	"github.com/stretchr/testify/require"
	"testing"
	"uuid"
)

type fakePhotoAccess struct {
	allowed bool
	err     error
}

func (f fakePhotoAccess) CanViewPhoto(context.Context, uuid.UUID) (bool, error) {
	return f.allowed, f.err
}

func TestPhotoContentRequiresLivePermission(t *testing.T) {
	id := uuid.New()
	repository := &fakeMediaRepository{media: map[uuid.UUID]domain.Media{id: {ID: id, MediaType: "image"}}}
	s := NewProtectedMediaService(repository, nil, &fakeObjectStorage{}, "test", fakePhotoAccess{allowed: true}, nil)
	require.NoError(t, s.AuthorizeContent(context.Background(), id))
	s.photoAccess = fakePhotoAccess{allowed: false}
	require.ErrorIs(t, s.AuthorizeContent(context.Background(), id), ErrForbidden)
	failure := errors.New("profiles unavailable")
	s.photoAccess = fakePhotoAccess{err: failure}
	require.ErrorIs(t, s.AuthorizeContent(context.Background(), id), failure)
	s.photoAccess = nil
	require.ErrorIs(t, s.AuthorizeContent(context.Background(), id), ErrForbidden)
}
