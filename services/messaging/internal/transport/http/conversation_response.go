package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/messaging/internal/application"
)

type conversationResponse = ConversationResponse

type conversationsResponse = ConversationsResponse

func mapConversationResponse(conversation application.ConversationDTO) conversationResponse {
	response := conversationResponse{
		ID:        conversation.ID.String(),
		Archived:  conversation.Archived,
		Pinned:    conversation.Pinned,
		Muted:     conversation.Muted,
		CreatedAt: conversation.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt: conversation.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
	}
	response.ParticipantIDs = make([]string, 0, len(conversation.ParticipantIDs))
	for _, participantID := range conversation.ParticipantIDs {
		response.ParticipantIDs = append(response.ParticipantIDs, participantID.String())
	}
	if conversation.LastMessage != nil {
		message := mapMessageResponse(*conversation.LastMessage)
		response.LastMessage = &message
	}
	return response
}

func writeConversation(w http.ResponseWriter, status int, conversation application.ConversationDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapConversationResponse(conversation))
}

func writeConversations(w http.ResponseWriter, status int, conversations []application.ConversationDTO) {
	response := conversationsResponse{Conversations: make([]conversationResponse, 0, len(conversations))}
	for _, conversation := range conversations {
		response.Conversations = append(response.Conversations, mapConversationResponse(conversation))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
