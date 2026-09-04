package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/messaging/internal/application"
)

type messageResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	SenderID       string  `json:"sender_id"`
	Body           string  `json:"body"`
	MediaID        *string `json:"media_id,omitempty"`
	CreatedAt      string  `json:"created_at"`
	EditedAt       string  `json:"edited_at,omitempty"`
	Deleted        bool    `json:"deleted"`
}

type messagesResponse struct {
	Messages []messageResponse `json:"messages"`
	HasMore  bool              `json:"has_more"`
}

func mapMessageResponse(message application.MessageDTO) messageResponse {
	response := messageResponse{ID: message.ID.String(), ConversationID: message.ConversationID.String(), SenderID: message.SenderID.String(), Body: message.Body, CreatedAt: message.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"), Deleted: message.DeletedAt != nil}
	if response.Deleted {
		response.Body = ""
	}
	if message.MediaID != nil {
		value := message.MediaID.String()
		response.MediaID = &value
	}
	if message.EditedAt != nil {
		response.EditedAt = message.EditedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00")
	}
	return response
}

func writeMessage(w http.ResponseWriter, status int, message application.MessageDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapMessageResponse(message))
}

func writeMessages(w http.ResponseWriter, status int, messages []application.MessageDTO, hasMore bool) {
	response := messagesResponse{Messages: make([]messageResponse, 0, len(messages)), HasMore: hasMore}
	for _, message := range messages {
		response.Messages = append(response.Messages, mapMessageResponse(message))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
