package httptransport

import (
	"encoding/json"
	"net/http"
	"uuid"

	"general-project/social/internal/application"
)

type actionResponse = ActionResponse

type friendRequestActionResponse = FriendRequestActionResponse

func writeAction(w http.ResponseWriter, status int, result application.ActionDTO) { //nolint:unparam // status is part of the response helper contract.
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
	var requestIDValue *string
	if requestID != "" {
		requestIDValue = &requestID
	}
	_ = json.NewEncoder(w).Encode(friendRequestActionResponse{RequestID: requestIDValue, State: result.State})
}
