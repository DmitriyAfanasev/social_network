package httptransport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseByteRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		size      int64
		wantStart int64
		wantEnd   int64
		wantValid bool
	}{
		{name: "closed range", value: "bytes=10-19", size: 100, wantStart: 10, wantEnd: 19, wantValid: true},
		{name: "open range", value: "bytes=90-", size: 100, wantStart: 90, wantEnd: 99, wantValid: true},
		{name: "suffix range", value: "bytes=-10", size: 100, wantStart: 90, wantEnd: 99, wantValid: true},
		{name: "end is clipped", value: "bytes=90-200", size: 100, wantStart: 90, wantEnd: 99, wantValid: true},
		{name: "range starts after file", value: "bytes=100-", size: 100},
		{name: "multiple ranges", value: "bytes=0-1,4-5", size: 100},
		{name: "wrong unit", value: "items=0-1", size: 100},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			start, end, valid := parseByteRange(test.value, test.size)
			require.Equal(t, test.wantValid, valid)
			if test.wantValid {
				require.Equal(t, test.wantStart, start)
				require.Equal(t, test.wantEnd, end)
			}
		})
	}
}
