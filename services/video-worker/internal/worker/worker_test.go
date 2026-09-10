package worker

import (
	"encoding/json"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestMustJSON(t *testing.T) {
	id := uuid.New()
	data := mustJSON(Job{VideoID: id, Bucket: "videos", RequestedHeights: []int{360, 720}})
	var got Job
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, id, got.VideoID)
	require.Equal(t, []int{360, 720}, got.RequestedHeights)
}

func BenchmarkMustJSON(b *testing.B) {
	job := Job{VideoID: uuid.New(), MediaID: uuid.New(), Bucket: "videos", ObjectKey: "source.mp4", RequestedHeights: []int{360, 720, 1080}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mustJSON(job)
	}
}
