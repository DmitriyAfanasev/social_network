package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"

	"general-project/messaging/internal/domain"
	"general-project/messaging/internal/ports"
)

var (
	// ErrValidation означает, что входные данные messaging некорректны.
	ErrValidation = errors.New("messaging validation failed")
	// ErrInteractionForbidden означает, что политика профиля запрещает сообщение.
	ErrInteractionForbidden = errors.New("messaging interaction forbidden")
)

const (
	messageCacheTTL      = 10 * time.Second
	conversationCacheTTL = 10 * time.Second
)

// MessageDTO представляет безопасное сообщение на границе application-слоя.
type MessageDTO struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	Body           string     `json:"body"`
	MediaID        *uuid.UUID `json:"media_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// MessageCreatedEvent содержит событие для realtime-доставки нового сообщения.
type MessageCreatedEvent struct {
	Type    string     `json:"type"`
	Message MessageDTO `json:"message"`
}

// MessageReadEvent содержит событие о прочтении сообщения.
type MessageReadEvent struct {
	Type           string    `json:"type"`
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
	UserID         uuid.UUID `json:"user_id"`
}

// ConversationDTO представляет безопасный диалог на границе application-слоя.
type ConversationDTO struct {
	ID             uuid.UUID
	ParticipantIDs []uuid.UUID
	Archived       bool
	Pinned         bool
	Muted          bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastMessage    *MessageDTO
}

// SendMessageInput содержит входные данные отправки сообщения.
type SendMessageInput struct {
	Body    string
	MediaID *uuid.UUID
}

// MessageService реализует сценарии conversations и messages.
type MessageService struct {
	repository ports.MessagingRepository
	cache      ports.MessageCache
	broker     ports.RealtimeBroker
	policy     ports.MessagePolicyReader
}

// NewMessageService создаёт application-сервис messaging.
func NewMessageService(repository ports.MessagingRepository, cache ports.MessageCache, broker ports.RealtimeBroker, policies ...ports.MessagePolicyReader) *MessageService {
	var policy ports.MessagePolicyReader
	if len(policies) > 0 {
		policy = policies[0]
	}
	return &MessageService{repository: repository, cache: cache, broker: broker, policy: policy}
}

// GetOrCreateDirect возвращает прямой диалог между двумя пользователями.
func (s *MessageService) GetOrCreateDirect(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (ConversationDTO, error) {
	if userID == otherUserID {
		return ConversationDTO{}, ErrValidation
	}
	if s.policy != nil {
		allowed, err := s.policy.CanMessage(ctx, userID, otherUserID)
		if err != nil {
			return ConversationDTO{}, err
		}
		if !allowed {
			return ConversationDTO{}, ErrInteractionForbidden
		}
	}
	conversation, err := s.repository.GetOrCreateDirect(ctx, userID, otherUserID)
	if err != nil {
		return ConversationDTO{}, err
	}
	s.invalidateConversations(ctx, userID, otherUserID)
	return toConversationDTO(conversation), nil
}

// ListConversations возвращает диалоги пользователя и кеширует результат.
func (s *MessageService) ListConversations(ctx context.Context, userID uuid.UUID, archivedValues ...bool) ([]ConversationDTO, error) {
	archived := false
	if len(archivedValues) > 0 {
		archived = archivedValues[0]
	}
	cacheKey := conversationsCacheKey(userID, archived)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached []ConversationDTO
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	conversations, err := s.repository.ListConversations(ctx, userID, archived)
	if err != nil {
		return nil, err
	}
	result := make([]ConversationDTO, 0, len(conversations))
	for _, conversation := range conversations {
		result = append(result, toConversationDTO(conversation))
	}
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(result); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, conversationCacheTTL)
		}
	}
	return result, nil
}

// ArchiveConversation архивирует или возвращает диалог в основной список.
func (s *MessageService) ArchiveConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, archived bool) error {
	if err := s.repository.ArchiveConversation(ctx, userID, conversationID, archived); err != nil {
		return err
	}
	s.invalidateConversations(ctx, userID)
	return nil
}

// HideConversation скрывает диалог у текущего пользователя.
func (s *MessageService) HideConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	if err := s.repository.HideConversation(ctx, userID, conversationID); err != nil {
		return err
	}
	s.invalidateConversations(ctx, userID)
	return nil
}

// MarkConversationUnread сбрасывает отметку прочтения диалога.
func (s *MessageService) MarkConversationUnread(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	if err := s.repository.MarkConversationUnread(ctx, userID, conversationID); err != nil {
		return err
	}
	s.invalidateConversations(ctx, userID)
	return nil
}

// SetConversationPinned закрепляет или открепляет диалог.
func (s *MessageService) SetConversationPinned(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, pinned bool) error {
	if err := s.repository.SetConversationPinned(ctx, userID, conversationID, pinned); err != nil {
		return err
	}
	s.invalidateConversations(ctx, userID)
	return nil
}

// SetConversationMuted включает или выключает mute диалога.
func (s *MessageService) SetConversationMuted(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, muted bool) error {
	if err := s.repository.SetConversationMuted(ctx, userID, conversationID, muted); err != nil {
		return err
	}
	s.invalidateConversations(ctx, userID)
	return nil
}

// ClearConversationHistory очищает историю диалога для текущего пользователя.
func (s *MessageService) ClearConversationHistory(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	if err := s.repository.ClearConversationHistory(ctx, userID, conversationID); err != nil {
		return err
	}
	s.invalidateConversations(ctx, userID)
	return nil
}

// ListMessages возвращает страницу сообщений только участнику диалога.
func (s *MessageService) ListMessages(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, offset int, limit int) ([]MessageDTO, bool, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, false, ErrValidation
	}
	messages, hasMore, err := s.repository.ListMessages(ctx, userID, conversationID, offset, limit)
	if err != nil {
		return nil, false, err
	}
	result := make([]MessageDTO, 0, len(messages))
	for _, message := range messages {
		result = append(result, toMessageDTO(message))
	}
	return result, hasMore, nil
}

// SendMessage создаёт сообщение от участника диалога.
func (s *MessageService) SendMessage(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, input SendMessageInput) (MessageDTO, []uuid.UUID, error) {
	body := strings.TrimSpace(input.Body)
	if len([]rune(body)) < 1 || len([]rune(body)) > 5000 {
		return MessageDTO{}, nil, ErrValidation
	}
	recipients, err := s.repository.ParticipantIDs(ctx, userID, conversationID)
	if err != nil {
		return MessageDTO{}, nil, err
	}
	if s.policy != nil {
		for _, recipientID := range recipients {
			if recipientID == userID {
				continue
			}
			allowed, policyErr := s.policy.CanMessage(ctx, userID, recipientID)
			if policyErr != nil {
				return MessageDTO{}, nil, policyErr
			}
			if !allowed {
				return MessageDTO{}, nil, ErrInteractionForbidden
			}
		}
	}
	message, err := s.repository.CreateMessage(ctx, domain.Message{ID: uuid.New(), ConversationID: conversationID, SenderID: userID, Body: body, MediaID: input.MediaID})
	if err != nil {
		return MessageDTO{}, nil, err
	}
	s.invalidateConversations(ctx, recipients...)
	if s.broker != nil {
		payload, marshalErr := json.Marshal(MessageCreatedEvent{Type: "message.new", Message: toMessageDTO(message)})
		if marshalErr != nil {
			return MessageDTO{}, nil, marshalErr
		}
		if publishErr := s.broker.Publish(ctx, ports.RealtimeEvent{RecipientIDs: recipients, Payload: payload}); publishErr != nil {
			return MessageDTO{}, nil, publishErr
		}
	}
	return toMessageDTO(message), recipients, nil
}

// UpdateMessage изменяет текст сообщения его автором.
func (s *MessageService) UpdateMessage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID, input SendMessageInput) (MessageDTO, error) {
	body := strings.TrimSpace(input.Body)
	if len([]rune(body)) < 1 || len([]rune(body)) > 5000 {
		return MessageDTO{}, ErrValidation
	}
	message, err := s.repository.UpdateMessage(ctx, userID, messageID, body)
	if err != nil {
		return MessageDTO{}, err
	}
	return toMessageDTO(message), nil
}

// DeleteMessage помечает сообщение удалённым.
func (s *MessageService) DeleteMessage(ctx context.Context, userID uuid.UUID, messageID uuid.UUID) error {
	return s.repository.DeleteMessage(ctx, userID, messageID)
}

// RemoveMessageMedia отсоединяет вложение от сообщения его автора.
func (s *MessageService) RemoveMessageMedia(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) (MessageDTO, error) {
	message, err := s.repository.RemoveMessageMedia(ctx, userID, conversationID, messageID)
	if err != nil {
		return MessageDTO{}, err
	}
	s.invalidateConversations(ctx, userID)
	return toMessageDTO(message), nil
}

// MarkRead обновляет позицию прочитанного сообщения пользователя.
func (s *MessageService) MarkRead(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, messageID uuid.UUID) error {
	if err := s.repository.MarkRead(ctx, userID, conversationID, messageID); err != nil {
		return err
	}
	recipients, err := s.repository.ParticipantIDs(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	if s.broker != nil {
		payload, marshalErr := json.Marshal(MessageReadEvent{
			Type:           "message.read",
			ConversationID: conversationID,
			MessageID:      messageID,
			UserID:         userID,
		})
		if marshalErr != nil {
			return marshalErr
		}
		if publishErr := s.broker.Publish(ctx, ports.RealtimeEvent{RecipientIDs: recipients, Payload: payload}); publishErr != nil {
			return publishErr
		}
	}
	return nil
}

func toMessageDTO(message domain.Message) MessageDTO {
	return MessageDTO{ID: message.ID, ConversationID: message.ConversationID, SenderID: message.SenderID, Body: message.Body, MediaID: message.MediaID, CreatedAt: message.CreatedAt, EditedAt: message.EditedAt, DeletedAt: message.DeletedAt}
}

func toConversationDTO(conversation domain.Conversation) ConversationDTO {
	dto := ConversationDTO{ID: conversation.ID, ParticipantIDs: conversation.ParticipantIDs, Archived: conversation.Archived, Pinned: conversation.Pinned, Muted: conversation.Muted, CreatedAt: conversation.CreatedAt, UpdatedAt: conversation.UpdatedAt}
	if conversation.LastMessage != nil {
		message := toMessageDTO(*conversation.LastMessage)
		dto.LastMessage = &message
	}
	return dto
}

func conversationsCacheKey(userID uuid.UUID, archived bool) string {
	state := "active"
	if archived {
		state = "archived"
	}
	return "messaging:v1:conversations:" + state + ":" + userID.String()
}

func (s *MessageService) invalidateConversations(ctx context.Context, userIDs ...uuid.UUID) {
	if s.cache == nil {
		return
	}
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, conversationsCacheKey(userID, false), conversationsCacheKey(userID, true))
	}
	_ = s.cache.Delete(ctx, keys...)
}
