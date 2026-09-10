package correlation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	ctx := WithID(context.Background(), "request-42")
	require.Equal(t, "request-42", ID(ctx))
	require.Empty(t, ID(context.Background()))
}
