package application

import (
	"context"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/messaging/internal/domain"
	"general-project/messaging/internal/ports"
)

func TestMessageServiceListMessagesValidatesPaginationAndMapsPage(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	userID, otherID := uuid.New(), uuid.New()
	conversation, err := repository.GetOrCreateDirect(context.Background(), userID, otherID)
	require.NoError(t, err)
	messageID := uuid.New()
	repository.messages[messageID] = domain.Message{ID: messageID, ConversationID: conversation.ID, SenderID: userID, Body: "hello"}
	service := NewMessageService(repository, nil, nil)

	tests := []struct {
		name   string
		offset int
		limit  int
	}{
		{name: "negative offset", offset: -1, limit: 10},
		{name: "zero limit", offset: 0, limit: 0},
		{name: "limit above maximum", offset: 0, limit: 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := service.ListMessages(context.Background(), userID, conversation.ID, tt.offset, tt.limit)

			require.ErrorIs(t, err, ErrValidation)
		})
	}

	messages, hasMore, err := service.ListMessages(context.Background(), userID, conversation.ID, 0, 10)

	require.NoError(t, err)
	require.False(t, hasMore)
	require.Len(t, messages, 1)
	require.Equal(t, messageID, messages[0].ID)
}

func TestMessageServiceRejectsMessageOutsideConversation(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	service := NewMessageService(repository, nil, nil)

	_, _, err := service.SendMessage(context.Background(), uuid.New(), uuid.New(), SendMessageInput{Body: "hello"})

	require.ErrorIs(t, err, ports.ErrNotFound)
}

func TestMessageServiceValidatesMessageBodyBoundaries(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	userID, otherID := uuid.New(), uuid.New()
	conversation, err := repository.GetOrCreateDirect(context.Background(), userID, otherID)
	require.NoError(t, err)
	service := NewMessageService(repository, nil, nil)

	tests := []struct {
		name string
		body string
	}{
		{name: "blank", body: " \t\n"},
		{name: "too long", body: string(make([]rune, 5001))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := service.SendMessage(context.Background(), userID, conversation.ID, SendMessageInput{Body: tt.body})

			require.ErrorIs(t, err, ErrValidation)
		})
	}
}
