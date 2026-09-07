package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/media/internal/application"
)

type musicResponse struct {
	ID        string   `json:"id"`
	MediaID   string   `json:"media_id"`
	UserID    string   `json:"user_id"`
	Title     string   `json:"title"`
	Artist    string   `json:"artist"`
	Duration  *float64 `json:"duration,omitempty"`
	CreatedAt string   `json:"created_at"`
	IsSaved   bool     `json:"is_saved"`
}

type musicListResponse struct {
	Tracks []musicResponse `json:"tracks"`
}

func mapMusicResponse(track application.MusicDTO) musicResponse {
	return musicResponse{ID: track.ID.String(), MediaID: track.MediaID.String(), UserID: track.UserID.String(), Title: track.Title, Artist: track.Artist, Duration: track.Duration, CreatedAt: track.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"), IsSaved: track.IsSaved}
}

func writeMusic(w http.ResponseWriter, status int, track application.MusicDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]musicResponse{"track": mapMusicResponse(track)})
}

func writeMusicList(w http.ResponseWriter, status int, tracks []application.MusicDTO) {
	response := musicListResponse{Tracks: make([]musicResponse, 0, len(tracks))}
	for _, track := range tracks {
		response.Tracks = append(response.Tracks, mapMusicResponse(track))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
