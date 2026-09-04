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
// @Summary Получить фото профиля
// @Tags profiles
// @Produce json
// @Param handle path string true "Публичный handle"
// @Success 200 {object} profilePhotosResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/profiles/{handle}/photos [get]
func (h *Handler) GetPhotos(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "требуется действующий access-токен")
		return
	}
	profile, err := h.profiles.GetByHandle(r.Context(), chi.URLParam(r, "handle"))
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
// @Summary Создать фотоальбом
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body photoAlbumRequest true "Название альбома"
// @Success 201 {object} photoAlbumResponse
// @Router /v1/profiles/me/photo-albums [post]
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
	album, err := h.media.CreateAlbum(r.Context(), userID, request.Title)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, photoAlbumResponse{Album: mapPhotoAlbumResponse(album)})
}

// AddPhoto добавляет медиаобъект в фотоальбом текущего пользователя.
// @Summary Добавить фото в альбом
// @Tags profiles
// @Accept json
// @Produce json
// @Param albumID path string true "UUID альбома"
// @Param request body addPhotoRequest true "Медиаобъект и подпись"
// @Success 201 {object} photoAlbumResponse
// @Router /v1/profiles/me/photo-albums/{albumID}/photos [post]
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
// @Summary Удалить фото профиля
// @Tags profiles
// @Param photoID path string true "UUID фотографии"
// @Success 204
// @Router /v1/profiles/me/photos/{photoID} [delete]
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
// @Summary Установить аватар
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body avatarRequest true "Медиаобъект"
// @Success 200 {object} avatarHistoryResponse
// @Router /v1/profiles/me/avatar [put]
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
// @Summary Выбрать аватар из истории
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body avatarRequest true "Медиаобъект"
// @Success 200 {object} avatarHistoryResponse
// @Router /v1/profiles/me/avatar/select [post]
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
// @Summary Удалить аватар
// @Tags profiles
// @Success 204
// @Router /v1/profiles/me/avatar [delete]
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
// @Summary Получить историю аватаров
// @Tags profiles
// @Produce json
// @Success 200 {object} avatarHistoryResponse
// @Router /v1/profiles/me/avatar/history [get]
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

type photoAlbumRequest struct {
	Title string `json:"title"`
}

type addPhotoRequest struct {
	MediaID string `json:"media_id"`
	Caption string `json:"caption"`
}

type avatarRequest struct {
	MediaID string `json:"media_id"`
}

type photoAlbumResponse struct {
	Album photoAlbumPayload `json:"album"`
}

type photoAlbumPayload struct {
	ID        *string        `json:"id,omitempty"`
	Title     string         `json:"title"`
	Kind      string         `json:"kind"`
	CreatedAt string         `json:"created_at"`
	Photos    []photoPayload `json:"photos"`
}

type photoPayload struct {
	ID        string  `json:"id"`
	AlbumID   *string `json:"album_id,omitempty"`
	MediaID   string  `json:"media_id"`
	URL       string  `json:"photo_url"`
	Caption   *string `json:"caption,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type profilePhotosResponse struct {
	OwnerID      string              `json:"owner_id"`
	IsOwnProfile bool                `json:"is_own_profile"`
	Albums       []photoAlbumPayload `json:"albums"`
}

type avatarHistoryResponse struct {
	CurrentAvatar string          `json:"current_avatar"`
	Avatars       []avatarPayload `json:"avatars"`
}

type avatarPayload struct {
	ID        string `json:"id"`
	MediaID   string `json:"media_id"`
	URL       string `json:"avatar_url"`
	CreatedAt string `json:"created_at"`
	IsCurrent bool   `json:"is_current"`
}

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
	result := photoAlbumPayload{ID: id, Title: album.Title, Kind: album.Kind, CreatedAt: album.CreatedAt.UTC().Format(timeFormat), Photos: make([]photoPayload, 0, len(album.Photos))}
	for _, photo := range album.Photos {
		var albumID *string
		if photo.AlbumID != nil {
			value := photo.AlbumID.String()
			albumID = &value
		}
		result.Photos = append(result.Photos, photoPayload{ID: photo.ID.String(), AlbumID: albumID, MediaID: photo.MediaID.String(), URL: photo.URL, Caption: photo.Caption, CreatedAt: photo.CreatedAt.UTC().Format(timeFormat)})
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
