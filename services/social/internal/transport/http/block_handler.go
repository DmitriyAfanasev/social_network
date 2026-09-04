package httptransport

import (
	"net/http"

	"general-project/social/internal/application"
)

// Block блокирует пользователя для текущего пользователя.
// @Summary Заблокировать пользователя
// @Tags blocks
// @Produce json
// @Param targetID path string true "UUID пользователя"
// @Success 200 {object} actionResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/social/blocks/{targetID} [put]
func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := parseTargetID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.Block(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeAction(w, http.StatusOK, result)
}

// Unblock снимает блокировку пользователя.
// @Summary Снять блокировку
// @Tags blocks
// @Produce json
// @Param targetID path string true "UUID пользователя"
// @Success 200 {object} actionResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 422 {object} httpx.ErrorResponse
// @Router /v1/social/blocks/{targetID} [delete]
func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := parseTargetID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := h.social.Unblock(r.Context(), actorID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeAction(w, http.StatusOK, result)
}
