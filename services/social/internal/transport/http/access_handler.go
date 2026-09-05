package httptransport

import (
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/social/internal/application"
)

// GetRelationship возвращает социальную близость текущего пользователя к цели.
// @Summary Проверить социальную близость
// @Tags social
// @Produce json
// @Param targetID path string true "UUID пользователя"
// @Success 200 {object} relationshipAccessResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/social/relationships/{targetID} [get]
func (h *Handler) GetRelationship(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "targetID"))
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.GetRelationship(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, relationshipAccessResponse{IsFriend: result.IsFriend, IsFriendOfFriend: result.IsFriendOfFriend, IsBlocked: result.IsBlocked})
}

type relationshipAccessResponse struct {
	IsFriend         bool `json:"is_friend"`
	IsFriendOfFriend bool `json:"is_friend_of_friend"`
	IsBlocked        bool `json:"is_blocked"`
}
