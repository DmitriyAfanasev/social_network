package domain

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestOwnership(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	require.True(t, (Post{AuthorID: owner}).CanBeManagedBy(owner))
	require.False(t, (Post{AuthorID: owner}).CanBeManagedBy(other))
	require.True(t, (Comment{AuthorID: owner}).CanBeManagedBy(owner))
	require.False(t, (Comment{AuthorID: owner}).CanBeManagedBy(other))
}
