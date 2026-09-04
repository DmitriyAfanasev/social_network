package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/messaging/internal/domain"
	"general-project/messaging/internal/ports"
)

type fakeMessagingRepository struct {
	conversations map[uuid.UUID]domain.Conversation
	messages      map[uuid.UUID]domain.Message
	participants  map[uuid.UUID][]uuid.UUID
	archived      bool
	actions       []string
}

func (f *fakeMessagingRepository) GetOrCreateDirect(_ context.Context, userID uuid.UUID, otherUserID uuid.UUID) (domain.Conversation, error) {
	conversation := domain.Conversation{ID: uuid.New(), DirectKey: domain.PairKey(userID, otherUserID), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	f.conversations[conversation.ID] = conversation
	f.participants[conversation.ID] = []uuid.UUID{userID, otherUserID}
	return conversation, nil
}

func (f *fakeMessagingRepository) ListConversations(_ context.Context, userID uuid.UUID, archived bool) ([]domain.Conversation, error) {
	f.archived = archived
	result := make([]domain.Conversation, 0)
	for conversationID, conversation := range f.conversations {
		for _, participantID := range f.participants[conversationID] {
			if participantID == userID {
				result = append(result, conversation)
				break
			}
		}
	}
	return result, nil
}

func (f *fakeMessagingRepository) ListMessages(_ context.Context, userID uuid.UUID, conversationID uuid.UUID, offset int, limit int) ([]domain.Message, bool, error) {
	if !f.isParticipant(userID, conversationID) {
		return nil, false, ports.ErrNotFound
	}
	result := make([]domain.Message, 0, limit)
	for _, message := range f.messages {
		if message.ConversationID == conversationID {
			result = append(result, message)
		}
	}
	if offset >= len(result) {
		return nil, false, nil
	}
	result = result[offset:]
	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}
	return result, hasMore, nil
}

func (f *fakeMessagingRepository) CreateMessage(_ context.Context, message domain.Message) (domain.Message, error) {
	if !f.isParticipant(message.SenderID, message.ConversationID) {
		return domain.Message{}, ports.ErrNotFound
	}
	message.CreatedAt = time.Now().UTC()
	f.messages[message.ID] = message
	return message, nil
}

func (f *fakeMessagingRepository) UpdateMessage(_ context.Context, userID uuid.UUID, messageID uuid.UUID, body string) (domain.Message, error) {
	message, ok := f.messages[messageID]
	if !ok || message.SenderID != userID {
		return domain.Message{}, ports.ErrNotFound
	}
	message.Body = body
	now := time.Now().UTC()
	message.EditedAt = &now
	f.messages[messageID] = message
	return message, nil
}

func (f *fakeMessagingRepository) DeleteMessage(_ context.Context, userID uuid.UUID, messageID uuid.UUID) error {
	message, ok := f.messages[messageID]
	if !ok || message.SenderID != userID {
		return ports.ErrNotFound
	}
	now := time.Now().UTC()
	message.DeletedAt = &now
	f.messages[messageID] = message
	return nil
}

func (f *fakeMessagingRepository) RemoveMessageMedia(_ context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) (domain.Message, error) {
	message, ok := f.messages[messageID]
	if !ok || message.SenderID != userID || message.ConversationID != conversationID {
		return domain.Message{}, ports.ErrNotFound
	}
	message.MediaID = nil
	f.messages[messageID] = message
	return message, nil
}

func (f *fakeMessagingRepository) MarkRead(_ context.Context, userID uuid.UUID, conversationID uuid.UUID, _ uuid.UUID) error {
	if !f.isParticipant(userID, conversationID) {
		return ports.ErrNotFound
	}
	return nil
}

func (f *fakeMessagingRepository) ArchiveConversation(context.Context, uuid.UUID, uuid.UUID, bool) error {
	f.actions = append(f.actions, "archive")
	return nil
}

func (f *fakeMessagingRepository) SetConversationPinned(context.Context, uuid.UUID, uuid.UUID, bool) error {
	f.actions = append(f.actions, "pin")
	return nil
}

func (f *fakeMessagingRepository) SetConversationMuted(context.Context, uuid.UUID, uuid.UUID, bool) error {
	f.actions = append(f.actions, "mute")
	return nil
}

func (f *fakeMessagingRepository) MarkConversationUnread(context.Context, uuid.UUID, uuid.UUID) error {
	f.actions = append(f.actions, "unread")
	return nil
}

func (f *fakeMessagingRepository) HideConversation(context.Context, uuid.UUID, uuid.UUID) error {
	f.actions = append(f.actions, "hide")
	return nil
}

func (f *fakeMessagingRepository) ClearConversationHistory(context.Context, uuid.UUID, uuid.UUID) error {
	f.actions = append(f.actions, "clear")
	return nil
}

func (f *fakeMessagingRepository) ParticipantIDs(_ context.Context, userID uuid.UUID, conversationID uuid.UUID) ([]uuid.UUID, error) {
	if !f.isParticipant(userID, conversationID) {
		return nil, ports.ErrNotFound
	}
	return f.participants[conversationID], nil
}

func (f *fakeMessagingRepository) isParticipant(userID uuid.UUID, conversationID uuid.UUID) bool {
	for _, participantID := range f.participants[conversationID] {
		if participantID == userID {
			return true
		}
	}
	return false
}

type fakeMessageCache struct {
	values map[string][]byte
}

type fakeRealtimeBroker struct {
	events []ports.RealtimeEvent
}

func (f *fakeRealtimeBroker) Publish(_ context.Context, event ports.RealtimeEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (*fakeRealtimeBroker) Subscribe(_ context.Context, _ func(context.Context, ports.RealtimeEvent) error) error {
	return nil
}

func (*fakeRealtimeBroker) Close() error {
	return nil
}

func (f *fakeMessageCache) Get(_ context.Context, key string) ([]byte, error) {
	return f.values[key], nil
}

func (f *fakeMessageCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.values[key] = value
	return nil
}

func (f *fakeMessageCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(f.values, key)
	}
	return nil
}

func newFakeMessagingRepository() *fakeMessagingRepository {
	return &fakeMessagingRepository{conversations: map[uuid.UUID]domain.Conversation{}, messages: map[uuid.UUID]domain.Message{}, participants: map[uuid.UUID][]uuid.UUID{}}
}

func TestMessageServiceRejectsDirectConversationWithSelf(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	service := NewMessageService(newFakeMessagingRepository(), nil, nil)

	_, err := service.GetOrCreateDirect(context.Background(), userID, userID)

	require.ErrorIs(t, err, ErrValidation)
}

func TestMessageServiceSendsMessageToParticipants(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	broker := &fakeRealtimeBroker{}
	first, second := uuid.New(), uuid.New()
	conversation, err := repository.GetOrCreateDirect(context.Background(), first, second)
	require.NoError(t, err)
	service := NewMessageService(repository, nil, broker)

	message, recipients, err := service.SendMessage(context.Background(), first, conversation.ID, SendMessageInput{Body: "  привет  "})

	require.NoError(t, err)
	require.Equal(t, "привет", message.Body)
	require.ElementsMatch(t, []uuid.UUID{first, second}, recipients)
	require.Len(t, broker.events, 1)
	var created MessageCreatedEvent
	require.NoError(t, json.Unmarshal(broker.events[0].Payload, &created))
	require.Equal(t, "message.new", created.Type)
	require.Equal(t, message.ID, created.Message.ID)
}

func TestMessageServicePublishesReadEvent(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	broker := &fakeRealtimeBroker{}
	first, second := uuid.New(), uuid.New()
	conversation, err := repository.GetOrCreateDirect(context.Background(), first, second)
	require.NoError(t, err)
	service := NewMessageService(repository, nil, broker)

	err = service.MarkRead(context.Background(), second, conversation.ID, uuid.New())

	require.NoError(t, err)
	require.Len(t, broker.events, 1)
	var read MessageReadEvent
	require.NoError(t, json.Unmarshal(broker.events[0].Payload, &read))
	require.Equal(t, "message.read", read.Type)
	require.Equal(t, second, read.UserID)
	require.ElementsMatch(t, []uuid.UUID{first, second}, broker.events[0].RecipientIDs)
}

func TestMessageServicePreservesDeletedMarker(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	deleted := toMessageDTO(domain.Message{ID: uuid.New(), Body: "секрет", DeletedAt: &now})
	require.Equal(t, &now, deleted.DeletedAt)
	require.Equal(t, "секрет", deleted.Body)
}

func TestMessageServiceCachesConversationList(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	userID, otherID := uuid.New(), uuid.New()
	conversation, err := repository.GetOrCreateDirect(context.Background(), userID, otherID)
	require.NoError(t, err)
	cache := &fakeMessageCache{values: map[string][]byte{}}
	service := NewMessageService(repository, cache, nil)

	first, err := service.ListConversations(context.Background(), userID)
	require.NoError(t, err)
	delete(repository.conversations, conversation.ID)
	second, err := service.ListConversations(context.Background(), userID)

	require.NoError(t, err)
	require.Len(t, first, 1)
	require.Equal(t, first, second)
}

func TestMessageServiceForwardsConversationStateActions(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	service := NewMessageService(repository, nil, nil)
	userID, conversationID := uuid.New(), uuid.New()

	require.NoError(t, service.ArchiveConversation(context.Background(), userID, conversationID, true))
	require.NoError(t, service.HideConversation(context.Background(), userID, conversationID))
	require.NoError(t, service.MarkConversationUnread(context.Background(), userID, conversationID))
	require.NoError(t, service.SetConversationPinned(context.Background(), userID, conversationID, true))
	require.NoError(t, service.SetConversationMuted(context.Background(), userID, conversationID, true))
	require.NoError(t, service.ClearConversationHistory(context.Background(), userID, conversationID))

	require.Equal(t, []string{"archive", "hide", "unread", "pin", "mute", "clear"}, repository.actions)
}

func TestMessageServiceRemovesMessageMedia(t *testing.T) {
	t.Parallel()

	repository := newFakeMessagingRepository()
	userID, otherID := uuid.New(), uuid.New()
	conversation, err := repository.GetOrCreateDirect(context.Background(), userID, otherID)
	require.NoError(t, err)
	messageID := uuid.New()
	mediaID := uuid.New()
	repository.messages[messageID] = domain.Message{ID: messageID, ConversationID: conversation.ID, SenderID: userID, Body: "файл", MediaID: &mediaID}
	service := NewMessageService(repository, nil, nil)

	message, err := service.RemoveMessageMedia(context.Background(), userID, conversation.ID, messageID)

	require.NoError(t, err)
	require.Nil(t, message.MediaID)
	require.Nil(t, repository.messages[messageID].MediaID)
}
