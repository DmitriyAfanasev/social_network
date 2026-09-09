package httptransport

import (
	"net/http"

	"general-project/social/internal/application"
)

// Subscribe создаёт подписку текущего пользователя на целевого.
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := parseTargetID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.Subscribe(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeAction(w, http.StatusOK, result)
}

// Unsubscribe отменяет подписку текущего пользователя.
func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := parseTargetID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.Unsubscribe(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeAction(w, http.StatusOK, result)
}
