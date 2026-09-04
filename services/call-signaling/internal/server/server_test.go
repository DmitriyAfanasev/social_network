package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidSignal(t *testing.T) {
	require.True(t, validSignal(&Signal{Kind: "offer", SDP: []byte(`{"type":"offer"}`)}))
	require.True(t, validSignal(&Signal{Kind: "ice", Candidate: []byte(`{"candidate":"candidate"}`)}))
	require.False(t, validSignal(&Signal{Kind: "answer"}))
	require.False(t, validSignal(&Signal{Kind: "unknown", SDP: []byte(`{}`)}))
	require.False(t, validSignal(nil))
}
