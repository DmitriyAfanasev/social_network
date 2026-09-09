package httptransport

import (
	"encoding/json"
	"net/http"
	"uuid"

	"general-project/media/internal/application"
	"general-project/media/internal/ports"
)

type videoResponse struct {
	ID                 string              `json:"id"`
	MediaID            string              `json:"media_id"`
	AlbumID            *string             `json:"album_id,omitempty"`
	OwnerID            string              `json:"owner_id,omitempty"`
	Title              string              `json:"title"`
	OriginalFilename   string              `json:"original_filename,omitempty"`
	Status             string              `json:"status"`
	Duration           *float64            `json:"duration,omitempty"`
	ViewsCount         int                 `json:"views_count"`
	LikesCount         int                 `json:"likes_count"`
	LikedByViewer      bool                `json:"is_liked_by_current"`
	BookmarkedByViewer bool                `json:"is_bookmarked_by_current"`
	FavoritedByViewer  bool                `json:"is_favorited_by_current"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
	Renditions         []renditionResponse `json:"renditions"`
}

type renditionResponse struct {
	ID          string  `json:"id"`
	Height      int     `json:"height"`
	ContentType string  `json:"content_type"`
	Size        int64   `json:"size"`
	Duration    float64 `json:"duration"`
	CreatedAt   string  `json:"created_at"`
}

type videoAlbumResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type videoAlbumListResponse struct {
	Albums []videoAlbumListItem `json:"albums"`
}

type videoAlbumListItem struct {
	ID     *string         `json:"id,omitempty"`
	Title  string          `json:"title"`
	Videos []videoResponse `json:"videos"`
}

type videoViewResponse struct {
	ViewsCount int `json:"views_count"`
}

type videoLikeResponse struct {
	LikesCount int  `json:"likes_count"`
	Liked      bool `json:"liked"`
}

type videoFlagResponse struct {
	Bookmarked *bool `json:"bookmarked,omitempty"`
	Favorited  *bool `json:"favorited,omitempty"`
}

func mapVideoResponse(video application.VideoDTO) videoResponse {
	response := videoResponse{ID: video.ID.String(), MediaID: video.MediaID.String(), OwnerID: video.OwnerID.String(), Title: video.Title, OriginalFilename: video.OriginalFilename, Status: video.Status, Duration: video.Duration, ViewsCount: video.ViewsCount, LikesCount: video.LikesCount, LikedByViewer: video.LikedByViewer, BookmarkedByViewer: video.BookmarkedByViewer, FavoritedByViewer: video.FavoritedByViewer, CreatedAt: video.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"), UpdatedAt: video.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"), Renditions: make([]renditionResponse, 0, len(video.Renditions))}
	if video.AlbumID != nil {
		albumID := video.AlbumID.String()
		response.AlbumID = &albumID
	}
	for _, rendition := range video.Renditions {
		response.Renditions = append(response.Renditions, renditionResponse{ID: rendition.ID.String(), Height: rendition.Height, ContentType: rendition.ContentType, Size: rendition.Size, Duration: rendition.Duration, CreatedAt: rendition.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00")})
	}
	return response
}
func mapAlbumResponse(albumID uuid.UUID, title string) videoAlbumResponse {
	return videoAlbumResponse{ID: albumID.String(), Title: title}
}
func writeVideoAlbum(w http.ResponseWriter, status int, albumID uuid.UUID, title string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapAlbumResponse(albumID, title))
}
func writeVideoAlbums(w http.ResponseWriter, status int, albums []application.VideoAlbumDTO) {
	response := videoAlbumListResponse{Albums: make([]videoAlbumListItem, 0, len(albums))}
	for _, album := range albums {
		item := videoAlbumListItem{Title: album.Title, Videos: make([]videoResponse, 0, len(album.Videos))}
		if album.ID != nil {
			id := album.ID.String()
			item.ID = &id
		}
		for _, video := range album.Videos {
			item.Videos = append(item.Videos, mapVideoResponse(video))
		}
		response.Albums = append(response.Albums, item)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
func writeVideoView(w http.ResponseWriter, status int, count int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(videoViewResponse{ViewsCount: count})
}
func writeVideoLike(w http.ResponseWriter, status int, result ports.InteractionResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(videoLikeResponse{LikesCount: result.LikesCount, Liked: result.Liked})
}
func writeVideoBookmark(w http.ResponseWriter, status int, enabled bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(videoFlagResponse{Bookmarked: &enabled})
}
func writeVideoFavorite(w http.ResponseWriter, status int, enabled bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(videoFlagResponse{Favorited: &enabled})
}
func writeVideo(w http.ResponseWriter, status int, video application.VideoDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapVideoResponse(video))
}
