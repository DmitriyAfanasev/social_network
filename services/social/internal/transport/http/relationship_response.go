package httptransport

import (
	"encoding/json"
	"net/http"
	"uuid"

	"general-project/social/internal/application"
)

type actionResponse struct {
	State string `json:"state"`
}

type friendRequestActionResponse struct {
	RequestID string `json:"request_id,omitempty"`
	State     string `json:"state"`
}

func writeAction(w http.ResponseWriter, status int, result application.ActionDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(actionResponse{State: result.State})
}

func writeFriendRequestAction(w http.ResponseWriter, status int, result application.FriendRequestActionDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	requestID := ""
	if result.RequestID != uuid.Nil() {
		requestID = result.RequestID.String()
	}
	_ = json.NewEncoder(w).Encode(friendRequestActionResponse{RequestID: requestID, State: result.State})
}
