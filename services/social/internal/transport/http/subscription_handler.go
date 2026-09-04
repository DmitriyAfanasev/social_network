package httptransport

import (
	"net/http"

	"general-project/social/internal/application"
)

// Subscribe создаёт подписку текущего пользователя на целевого.
// @Summary Подписаться на пользователя
// @Tags subscriptions
// @Produce json
// @Param targetID path string true "UUID пользователя"
// @Success 200 {object} actionResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/social/subscriptions/{targetID} [put]
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
// @Summary Отписаться от пользователя
// @Tags subscriptions
// @Produce json
// @Param targetID path string true "UUID пользователя"
// @Success 200 {object} actionResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/social/subscriptions/{targetID} [delete]
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
