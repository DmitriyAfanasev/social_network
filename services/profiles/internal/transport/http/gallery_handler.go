package httptransport

import (
	"general-project/libs/platform/auth"
	"general-project/profiles/internal/application"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"
)

// PhotoMediaVisibility возвращает решение о доступе к содержимому фотографии.
func (h *Handler) PhotoMediaVisibility(w http.ResponseWriter, r *http.Request) {
	viewerID, _ := auth.UserIDFromContext(r.Context())
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		writeProfileError(w, application.ErrValidation)
		return
	}
	allowed, err := h.media.CanViewPhotoMedia(r.Context(), viewerID, mediaID)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, photoMediaVisibilityResponse{Allowed: allowed})
}

type photoMediaVisibilityResponse = PhotoMediaVisibilityResponse

// UpdatePhotoAlbum изменяет настройки альбома владельца.
func (h *Handler) UpdatePhotoAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "albumID"))
	var request photoAlbumRequest
	if err != nil || decodeJSON(r, &request) != nil {
		writeProfileError(w, application.ErrValidation)
		return
	}
	album, err := h.media.SaveAlbum(r.Context(), userID, id, application.AlbumInput{Title: request.Title, Description: request.Description, Visibility: request.Visibility, CommentPolicy: request.CommentPolicy})
	if err != nil {
		writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, photoAlbumResponse{Album: mapPhotoAlbumResponse(album)})
}

type updatePhotoRequest = UpdatePhotoRequest

// UpdatePhoto изменяет подпись, координаты и состояние архива фотографии.
func (h *Handler) UpdatePhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "photoID"))
	var request updatePhotoRequest
	if err != nil || decodeJSON(r, &request) != nil {
		writeProfileError(w, application.ErrValidation)
		return
	}
	if err := h.media.UpdatePhoto(r.Context(), userID, id, application.PhotoUpdateInput{Caption: request.Caption, Archived: request.Archived, Latitude: request.Latitude, Longitude: request.Longitude}); err != nil {
		writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type photoCommentRequest = PhotoCommentRequest
type photoCommentPayload = PhotoCommentPayload
type photoCommentsResponse = PhotoCommentsResponse

// ListPhotoComments возвращает комментарии доступной фотографии.
func (h *Handler) ListPhotoComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "photoID"))
	if err != nil {
		writeProfileError(w, application.ErrValidation)
		return
	}
	comments, err := h.media.ListPhotoComments(r.Context(), userID, id)
	if err != nil {
		writeProfileError(w, err)
		return
	}
	result := photoCommentsResponse{Comments: make([]photoCommentPayload, 0, len(comments))}
	for _, c := range comments {
		result.Comments = append(result.Comments, photoCommentPayload{ID: c.ID.String(), UserID: c.UserID.String(), AuthorName: c.AuthorName, Body: c.Body, CreatedAt: c.CreatedAt.UTC().Format(timeFormat)})
	}
	writeJSON(w, http.StatusOK, result)
}

// AddPhotoComment добавляет комментарий с проверкой политики альбома.
func (h *Handler) AddPhotoComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "photoID"))
	var request photoCommentRequest
	if err != nil || decodeJSON(r, &request) != nil {
		writeProfileError(w, application.ErrValidation)
		return
	}
	if err := h.media.AddPhotoComment(r.Context(), userID, id, request.Body); err != nil {
		writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
