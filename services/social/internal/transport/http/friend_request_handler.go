package httptransport

import (
	"context"
	"net/http"
	"strings"
	"uuid"

	"general-project/social/internal/application"
)

// ListFriendRequests возвращает входящие и исходящие pending-заявки.
func (h *Handler) ListFriendRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	direction := strings.ToLower(r.URL.Query().Get("direction"))
	if direction == "" {
		direction = "all"
	}
	if direction != "all" && direction != "incoming" && direction != "outgoing" {
		writeSocialError(w, application.ErrValidation)
		return
	}
	response := friendRequestsResponse{}
	if direction == "all" || direction == "incoming" {
		requests, err := h.social.ListFriendRequests(r.Context(), userID, true)
		if err != nil {
			writeSocialError(w, err)
			return
		}
		response.Incoming = mapFriendRequests(requests)
	}
	if direction == "all" || direction == "outgoing" {
		requests, err := h.social.ListFriendRequests(r.Context(), userID, false)
		if err != nil {
			writeSocialError(w, err)
			return
		}
		response.Outgoing = mapFriendRequests(requests)
	}
	writeJSON(w, http.StatusOK, response)
}

// AcceptFriendRequest принимает входящую заявку и создаёт friendship.
func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	h.transitionFriendRequest(w, r, h.social.AcceptFriendRequest)
}

// DeclineFriendRequest отклоняет входящую заявку.
func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	h.transitionFriendRequest(w, r, h.social.DeclineFriendRequest)
}

// CancelFriendRequest отменяет исходящую заявку.
func (h *Handler) CancelFriendRequest(w http.ResponseWriter, r *http.Request) {
	h.transitionFriendRequest(w, r, h.social.CancelFriendRequest)
}

type friendRequestTransition func(context.Context, uuid.UUID, uuid.UUID) (application.FriendRequestActionDTO, error)

func (h *Handler) transitionFriendRequest(w http.ResponseWriter, r *http.Request, transition friendRequestTransition) {
	actorID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	requestID, err := parseRequestID(r)
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	result, err := transition(r.Context(), actorID, requestID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeFriendRequestAction(w, http.StatusOK, result)
}
