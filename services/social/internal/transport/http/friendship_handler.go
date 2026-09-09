package httptransport

import (
	"net/http"

	"general-project/social/internal/application"
)

// AddFriend создаёт заявку в друзья для текущего и целевого пользователя.








func (h *Handler) AddFriend(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := parseTargetID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.AddFriend(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeFriendRequestAction(w, http.StatusOK, result)
}

// RemoveFriend удаляет friendship между текущим и целевым пользователем.








func (h *Handler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := parseTargetID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.RemoveFriend(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeAction(w, http.StatusOK, result)
}
