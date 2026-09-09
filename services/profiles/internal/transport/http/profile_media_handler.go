package httptransport

import (
	"encoding/json"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/libs/platform/auth"
	"general-project/libs/platform/httpx"
	"general-project/profiles/internal/application"
)

// GetPhotos возвращает фотоальбомы профиля, включая историю аватаров.







func (h *Handler) GetPhotos(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	identifier := chi.URLParam(r, "handle")
	profile, err := func() (application.ProfileDTO, error) {
		if userID, parseErr := uuid.Parse(identifier); parseErr == nil {
			return h.profiles.GetByUserID(r.Context(), userID)
		}
		return h.profiles.GetByHandle(r.Context(), identifier)
	}()
	if err != nil {
		writeProfileError(w, err)
		return
	}
	result, err := h.media.ListPhotos(r.Context(), profile.UserID, viewerID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapProfilePhotosResponse(result))
}

// CreatePhotoAlbum создаёт фотоальбом текущего пользователя.







func (h *Handler) CreatePhotoAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var request photoAlbumRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return
	}
	album, err := h.media.SaveAlbum(r.Context(), userID, uuid.Nil(), application.AlbumInput{Title: request.Title, Description: request.Description, Visibility: request.Visibility, CommentPolicy: request.CommentPolicy})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, photoAlbumResponse{Album: mapPhotoAlbumResponse(album)})
}

// AddPhoto добавляет медиаобъект в фотоальбом текущего пользователя.








func (h *Handler) AddPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	albumID, mediaID, caption, ok := decodePhotoRequest(w, r)
	if !ok {
		return
	}
	album, err := h.media.AddPhoto(r.Context(), userID, albumID, mediaID, caption)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, photoAlbumResponse{Album: mapPhotoAlbumResponse(album)})
}

// DeletePhoto удаляет фотографию текущего пользователя.





func (h *Handler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	photoID, err := uuid.Parse(chi.URLParam(r, "photoID"))
	if err != nil {
		writeProfileError(w, application.ErrValidation)
		return
	}
	if err := h.media.DeletePhoto(r.Context(), userID, photoID); err != nil {
		writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetAvatar назначает медиаобъект текущим аватаром пользователя.







func (h *Handler) SetAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	mediaID, ok := decodeAvatarRequest(w, r)
	if !ok {
		return
	}
	result, err := h.media.SetAvatar(r.Context(), userID, mediaID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapAvatarHistoryResponse(result))
}

// SelectAvatar выбирает аватар из истории пользователя.







func (h *Handler) SelectAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	mediaID, ok := decodeAvatarRequest(w, r)
	if !ok {
		return
	}
	result, err := h.media.SelectAvatar(r.Context(), userID, mediaID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapAvatarHistoryResponse(result))
}

// RemoveAvatar сбрасывает текущий аватар на стабильное значение по умолчанию.




func (h *Handler) RemoveAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if err := h.media.RemoveAvatar(r.Context(), userID); err != nil {
		writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AvatarHistory возвращает список доступных аватаров пользователя.





func (h *Handler) AvatarHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	result, err := h.media.AvatarHistory(r.Context(), userID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapAvatarHistoryResponse(result))
}

type photoAlbumRequest = PhotoAlbumRequest
type addPhotoRequest = AddPhotoRequest
type avatarRequest = AvatarRequest
type photoAlbumResponse = PhotoAlbumResponse
type photoAlbumPayload = PhotoAlbumPayload
type photoPayload = PhotoPayload
type profilePhotosResponse = ProfilePhotosResponse
type avatarHistoryResponse = AvatarHistoryResponse
type avatarPayload = AvatarPayload

func currentUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
	}
	return userID, ok
}

func decodePhotoRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, string, bool) {
	albumID, err := uuid.Parse(chi.URLParam(r, "albumID"))
	if err != nil {
		writeProfileError(w, application.ErrValidation)
		return uuid.Nil(), uuid.Nil(), "", false
	}
	var request addPhotoRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return uuid.Nil(), uuid.Nil(), "", false
	}
	mediaID, err := uuid.Parse(request.MediaID)
	if err != nil {
		writeProfileError(w, application.ErrValidation)
		return uuid.Nil(), uuid.Nil(), "", false
	}
	return albumID, mediaID, request.Caption, true
}

func decodeAvatarRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	var request avatarRequest
	if err := decodeJSON(r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "некорректное тело запроса")
		return uuid.Nil(), false
	}
	mediaID, err := uuid.Parse(request.MediaID)
	if err != nil {
		writeProfileError(w, application.ErrValidation)
		return uuid.Nil(), false
	}
	return mediaID, true
}

func mapPhotoAlbumResponse(album application.ProfilePhotoAlbumDTO) photoAlbumPayload {
	var id *string
	if album.ID != nil {
		value := album.ID.String()
		id = &value
	}
	result := photoAlbumPayload{ID: id, Title: album.Title, Description: album.Description, Visibility: album.Visibility, CommentPolicy: album.CommentPolicy, Kind: album.Kind, CreatedAt: album.CreatedAt.UTC().Format(timeFormat), Photos: make([]photoPayload, 0, len(album.Photos))}
	for _, photo := range album.Photos {
		var albumID *string
		if photo.AlbumID != nil {
			value := photo.AlbumID.String()
			albumID = &value
		}
		result.Photos = append(result.Photos, photoPayload{ID: photo.ID.String(), AlbumID: albumID, MediaID: photo.MediaID.String(), URL: photo.URL, Caption: photo.Caption, CreatedAt: photo.CreatedAt.UTC().Format(timeFormat), Archived: photo.Archived, Latitude: photo.Latitude, Longitude: photo.Longitude})
	}
	return result
}

func mapProfilePhotosResponse(result application.ProfilePhotosDTO) profilePhotosResponse {
	response := profilePhotosResponse{OwnerID: result.OwnerID.String(), IsOwnProfile: result.IsOwnProfile, Albums: make([]photoAlbumPayload, 0, len(result.Albums))}
	for _, album := range result.Albums {
		response.Albums = append(response.Albums, mapPhotoAlbumResponse(album))
	}
	return response
}

func mapAvatarHistoryResponse(result application.AvatarHistoryDTO) avatarHistoryResponse {
	response := avatarHistoryResponse{CurrentAvatar: result.CurrentURL, Avatars: make([]avatarPayload, 0, len(result.Avatars))}
	for _, avatar := range result.Avatars {
		response.Avatars = append(response.Avatars, avatarPayload{ID: avatar.ID.String(), MediaID: avatar.MediaID.String(), URL: avatar.URL, CreatedAt: avatar.CreatedAt.UTC().Format(timeFormat), IsCurrent: avatar.IsCurrent})
	}
	return response
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

const timeFormat = "2006-01-02T15:04:05.999999Z07:00"
