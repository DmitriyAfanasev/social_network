package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"uuid"

	"general-project/libs/platform/correlation"
	"general-project/social/internal/domain"
	"general-project/social/internal/ports"
)

var (
	// ErrValidation означает, что операция отношения недопустима.
	ErrValidation = errors.New("social validation failed")
	// ErrBlocked означает, что один из пользователей заблокировал другого.
	ErrBlocked = errors.New("social interaction blocked")
	// ErrFriendRequestNotFound означает, что заявка не существует.
	ErrFriendRequestNotFound = errors.New("friend request not found")
	// ErrFriendRequestForbidden означает, что пользователь не может изменить заявку.
	ErrFriendRequestForbidden = errors.New("friend request operation forbidden")
	// ErrFriendRequestState означает, что переход заявки недопустим.
	ErrFriendRequestState = errors.New("friend request state conflict")
)

// ActionDTO описывает результат изменения социального отношения.
type ActionDTO struct {
	State string
}

// FriendRequestActionDTO описывает результат операции над заявкой.
type FriendRequestActionDTO struct {
	RequestID uuid.UUID
	State     string
}

// FriendRequestDTO описывает заявку на границе application-слоя.
type FriendRequestDTO struct {
	ID          uuid.UUID
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Status      string
	CreatedAt   time.Time
	RespondedAt *time.Time
}

// RelationshipsDTO содержит направленные и ненаправленные связи пользователя.
type RelationshipsDTO struct {
	Friends       []uuid.UUID
	Subscribers   []uuid.UUID
	Subscriptions []uuid.UUID
}

// RecommendationDTO описывает кандидата в друзья и число общих друзей.
type RecommendationDTO struct {
	UserID        uuid.UUID
	CommonFriends int
}

// SocialService реализует блокировки, подписки и дружбу.
type SocialService struct {
	blocks        ports.BlockRepository
	relationships ports.FriendshipRepository
	cache         ports.SocialCache
	outbox        ports.OutboxRepository
}

// NewSocialService создаёт application-сервис социальных отношений.
func NewSocialService(blocks ports.BlockRepository, relationships ports.FriendshipRepository, caches ...ports.SocialCache) *SocialService {
	var socialCache ports.SocialCache
	if len(caches) > 0 {
		socialCache = caches[0]
	}
	return &SocialService{blocks: blocks, relationships: relationships, cache: socialCache}
}

// NewSocialServiceWithOutbox создаёт social-сервис с надёжной публикацией событий.
func NewSocialServiceWithOutbox(blocks ports.BlockRepository, relationships ports.FriendshipRepository, cache ports.SocialCache, outbox ports.OutboxRepository) *SocialService {
	service := NewSocialService(blocks, relationships, cache)
	service.outbox = outbox
	return service
}

const socialCacheTTL = time.Minute

// GetRelationships возвращает списки друзей, подписчиков и подписок пользователя.
func (s *SocialService) GetRelationships(ctx context.Context, userID uuid.UUID) (RelationshipsDTO, error) {
	key := relationshipsCacheKey(userID)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, key); err == nil && payload != nil {
			var cached RelationshipsDTO
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	friends, err := s.relationships.ListFriends(ctx, userID)
	if err != nil {
		return RelationshipsDTO{}, err
	}
	subscribers, err := s.relationships.ListSubscribers(ctx, userID)
	if err != nil {
		return RelationshipsDTO{}, err
	}
	subscriptions, err := s.relationships.ListSubscriptions(ctx, userID)
	if err != nil {
		return RelationshipsDTO{}, err
	}
	result := RelationshipsDTO{Friends: friends, Subscribers: subscribers, Subscriptions: subscriptions}
	s.cacheJSON(ctx, key, result)
	return result, nil
}

// GetRecommendations строит рекомендации по общим друзьям и блокировкам.
func (s *SocialService) GetRecommendations(ctx context.Context, userID uuid.UUID, limit int) ([]RecommendationDTO, error) {
	if limit < 1 || limit > 50 {
		return nil, ErrValidation
	}
	key := recommendationsCacheKey(userID)
	var all []RecommendationDTO
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, key); err == nil && payload != nil {
			_ = json.Unmarshal(payload, &all)
		}
	}
	if all == nil {
		friends, err := s.relationships.ListFriends(ctx, userID)
		if err != nil {
			return nil, err
		}
		direct := make(map[uuid.UUID]struct{}, len(friends))
		for _, friendID := range friends {
			direct[friendID] = struct{}{}
		}
		counts := make(map[uuid.UUID]int)
		for _, friendID := range friends {
			friendsOfFriend, err := s.relationships.ListFriends(ctx, friendID)
			if err != nil {
				return nil, err
			}
			for _, candidateID := range friendsOfFriend {
				if candidateID == userID {
					continue
				}
				if _, exists := direct[candidateID]; exists {
					continue
				}
				blocked, err := s.blocks.IsBlocked(ctx, userID, candidateID)
				if err != nil {
					return nil, err
				}
				if blocked {
					continue
				}
				subscribed, err := s.relationships.IsSubscribed(ctx, userID, candidateID)
				if err != nil {
					return nil, err
				}
				if subscribed {
					continue
				}
				counts[candidateID]++
			}
		}
		all = make([]RecommendationDTO, 0, len(counts))
		for candidateID, commonFriends := range counts {
			all = append(all, RecommendationDTO{UserID: candidateID, CommonFriends: commonFriends})
		}
		sort.Slice(all, func(i, j int) bool {
			if all[i].CommonFriends != all[j].CommonFriends {
				return all[i].CommonFriends > all[j].CommonFriends
			}
			return all[i].UserID.String() < all[j].UserID.String()
		})
		s.cacheJSON(ctx, key, all)
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// Block блокирует пользователя и запрещает дальнейшие social-взаимодействия.
func (s *SocialService) Block(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (ActionDTO, error) {
	if actorID == targetID {
		return ActionDTO{}, ErrValidation
	}
	if s.outbox != nil {
		now := time.Now().UTC()
		payload, marshalErr := json.Marshal(map[string]any{"blocker_id": actorID, "blocked_id": targetID})
		if marshalErr != nil {
			return ActionDTO{}, marshalErr
		}
		if err := s.blocks.BlockWithOutbox(ctx, actorID, targetID, ports.OutboxEvent{ID: uuid.New(), EventType: "social.block.created", AggregateID: &targetID, Payload: payload, CorrelationID: correlation.ID(ctx), AvailableAt: now, CreatedAt: now}); err != nil {
			return ActionDTO{}, err
		}
	} else {
		if err := s.blocks.Block(ctx, actorID, targetID); err != nil {
			return ActionDTO{}, err
		}
		first, second := domain.Pair(actorID, targetID)
		if _, err := s.relationships.RemoveFriendship(ctx, first, second); err != nil {
			return ActionDTO{}, err
		}
		if _, err := s.relationships.Unsubscribe(ctx, actorID, targetID); err != nil {
			return ActionDTO{}, err
		}
		if _, err := s.relationships.Unsubscribe(ctx, targetID, actorID); err != nil {
			return ActionDTO{}, err
		}
	}
	s.invalidate(ctx, actorID, targetID)
	return ActionDTO{State: "blocked"}, nil
}

// Unblock снимает блокировку пользователя.
func (s *SocialService) Unblock(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (ActionDTO, error) {
	if actorID == targetID {
		return ActionDTO{}, ErrValidation
	}
	_, err := s.blocks.Unblock(ctx, actorID, targetID)
	if err != nil {
		return ActionDTO{}, err
	}
	s.invalidate(ctx, actorID, targetID)
	return ActionDTO{State: "unblocked"}, nil
}

// AddFriend создаёт pending-заявку, если пользователи не заблокировали друг друга.
func (s *SocialService) AddFriend(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (FriendRequestActionDTO, error) {
	return s.SendFriendRequest(ctx, actorID, targetID)
}

// SendFriendRequest отправляет заявку в друзья и публикует friend.requested.
func (s *SocialService) SendFriendRequest(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (FriendRequestActionDTO, error) {
	if actorID == targetID {
		return FriendRequestActionDTO{}, ErrValidation
	}
	blocked, err := s.blocks.IsBlocked(ctx, actorID, targetID)
	if err != nil {
		return FriendRequestActionDTO{}, err
	}
	if blocked {
		return FriendRequestActionDTO{}, ErrBlocked
	}
	isFriend, err := s.relationships.IsFriend(ctx, actorID, targetID)
	if err != nil {
		return FriendRequestActionDTO{}, err
	}
	if isFriend {
		return FriendRequestActionDTO{State: "friends"}, nil
	}
	now := time.Now().UTC()
	request := domain.FriendRequest{ID: uuid.New(), SenderID: actorID, RecipientID: targetID, Status: domain.FriendRequestPending, CreatedAt: now}
	var event *ports.OutboxEvent
	if s.outbox != nil {
		payload, marshalErr := json.Marshal(map[string]any{"request_id": request.ID, "actor_id": actorID, "target_id": targetID})
		if marshalErr != nil {
			return FriendRequestActionDTO{}, marshalErr
		}
		outboxEvent := ports.OutboxEvent{ID: uuid.New(), EventType: "social.friend_request.created", AggregateID: &request.ID, Payload: payload, CorrelationID: correlation.ID(ctx), AvailableAt: now, CreatedAt: now}
		event = &outboxEvent
	}
	stored, _, err := s.relationships.CreateFriendRequest(ctx, request, event)
	if err != nil {
		return FriendRequestActionDTO{}, mapFriendRequestError(err)
	}
	s.invalidate(ctx, actorID, targetID)
	return FriendRequestActionDTO{RequestID: stored.ID, State: string(stored.Status)}, nil
}

// ListFriendRequests возвращает входящие или исходящие pending-заявки.
func (s *SocialService) ListFriendRequests(ctx context.Context, userID uuid.UUID, incoming bool) ([]FriendRequestDTO, error) {
	requests, err := s.relationships.ListFriendRequests(ctx, userID, incoming)
	if err != nil {
		return nil, err
	}
	result := make([]FriendRequestDTO, 0, len(requests))
	for _, request := range requests {
		result = append(result, mapFriendRequest(request))
	}
	return result, nil
}

// AcceptFriendRequest принимает заявку и создаёт friendship атомарно.
func (s *SocialService) AcceptFriendRequest(ctx context.Context, actorID uuid.UUID, requestID uuid.UUID) (FriendRequestActionDTO, error) {
	return s.transitionFriendRequest(ctx, actorID, requestID, domain.FriendRequestAccepted, "social.friendship.created", actorID)
}

// DeclineFriendRequest отклоняет входящую заявку.
func (s *SocialService) DeclineFriendRequest(ctx context.Context, actorID uuid.UUID, requestID uuid.UUID) (FriendRequestActionDTO, error) {
	return s.transitionFriendRequest(ctx, actorID, requestID, domain.FriendRequestDeclined, "social.friend_request.declined", actorID)
}

// CancelFriendRequest отменяет исходящую pending-заявку.
func (s *SocialService) CancelFriendRequest(ctx context.Context, actorID uuid.UUID, requestID uuid.UUID) (FriendRequestActionDTO, error) {
	return s.transitionFriendRequest(ctx, actorID, requestID, domain.FriendRequestCancelled, "social.friend_request.cancelled", actorID)
}

func (s *SocialService) transitionFriendRequest(ctx context.Context, actorID uuid.UUID, requestID uuid.UUID, status domain.FriendRequestStatus, eventType string, eventActorID uuid.UUID) (FriendRequestActionDTO, error) {
	request, err := s.relationships.GetFriendRequest(ctx, requestID)
	if errors.Is(err, ports.ErrFriendRequestNotFound) {
		return FriendRequestActionDTO{}, ErrFriendRequestNotFound
	}
	if err != nil {
		return FriendRequestActionDTO{}, err
	}
	if !request.IsPending() {
		return FriendRequestActionDTO{}, ErrFriendRequestState
	}
	if status == domain.FriendRequestCancelled {
		if request.SenderID != actorID {
			return FriendRequestActionDTO{}, ErrFriendRequestForbidden
		}
	} else if status == domain.FriendRequestAccepted || status == domain.FriendRequestDeclined {
		if request.RecipientID != actorID {
			return FriendRequestActionDTO{}, ErrFriendRequestForbidden
		}
	} else {
		return FriendRequestActionDTO{}, ErrFriendRequestState
	}

	now := time.Now().UTC()
	var event *ports.OutboxEvent
	if s.outbox != nil {
		targetID := request.SenderID
		if status != domain.FriendRequestAccepted {
			targetID = request.RecipientID
		}
		payload, marshalErr := json.Marshal(map[string]any{"request_id": request.ID, "actor_id": eventActorID, "target_id": targetID})
		if marshalErr != nil {
			return FriendRequestActionDTO{}, marshalErr
		}
		eventAggregateID := request.ID
		outboxEvent := ports.OutboxEvent{ID: uuid.New(), EventType: eventType, AggregateID: &eventAggregateID, Payload: payload, CorrelationID: correlation.ID(ctx), AvailableAt: now, CreatedAt: now}
		event = &outboxEvent
	}
	updated, err := s.relationships.TransitionFriendRequest(ctx, requestID, actorID, status, event)
	if err != nil {
		return FriendRequestActionDTO{}, mapFriendRequestError(err)
	}
	s.invalidate(ctx, request.SenderID, request.RecipientID)
	return FriendRequestActionDTO{RequestID: updated.ID, State: string(updated.Status)}, nil
}

// RemoveFriend удаляет friendship между пользователями.
func (s *SocialService) RemoveFriend(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (ActionDTO, error) {
	if actorID == targetID {
		return ActionDTO{}, ErrValidation
	}
	first, second := domain.Pair(actorID, targetID)
	_, err := s.relationships.RemoveFriendship(ctx, first, second)
	if err != nil {
		return ActionDTO{}, err
	}
	s.invalidate(ctx, actorID, targetID)
	return ActionDTO{State: "not_friends"}, nil
}

// Subscribe подписывает текущего пользователя на целевой профиль.
func (s *SocialService) Subscribe(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (ActionDTO, error) {
	if actorID == targetID {
		return ActionDTO{}, ErrValidation
	}
	blocked, err := s.blocks.IsBlocked(ctx, actorID, targetID)
	if err != nil {
		return ActionDTO{}, err
	}
	if blocked {
		return ActionDTO{}, ErrBlocked
	}
	var subscribeErr error
	if s.outbox == nil {
		subscribeErr = s.relationships.Subscribe(ctx, actorID, targetID)
	} else {
		now := time.Now().UTC()
		payload, marshalErr := json.Marshal(map[string]any{"subscriber_id": actorID, "target_id": targetID})
		if marshalErr != nil {
			return ActionDTO{}, marshalErr
		}
		subscribeErr = s.relationships.SubscribeWithOutbox(ctx, actorID, targetID, ports.OutboxEvent{ID: uuid.New(), EventType: "social.subscription.created", AggregateID: &targetID, Payload: payload, CorrelationID: correlation.ID(ctx), AvailableAt: now, CreatedAt: now})
	}
	if subscribeErr != nil {
		return ActionDTO{}, subscribeErr
	}
	s.invalidate(ctx, actorID, targetID)
	return ActionDTO{State: "subscribed"}, nil
}

// Unsubscribe отменяет подписку текущего пользователя на целевой профиль.
func (s *SocialService) Unsubscribe(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (ActionDTO, error) {
	if actorID == targetID {
		return ActionDTO{}, ErrValidation
	}
	if _, err := s.relationships.Unsubscribe(ctx, actorID, targetID); err != nil {
		return ActionDTO{}, err
	}
	s.invalidate(ctx, actorID, targetID)
	return ActionDTO{State: "not_subscribed"}, nil
}

func (s *SocialService) cacheJSON(ctx context.Context, key string, value any) {
	if s.cache == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err == nil {
		_ = s.cache.Set(ctx, key, payload, socialCacheTTL)
	}
}

func (s *SocialService) invalidate(ctx context.Context, userIDs ...uuid.UUID) {
	if s.cache == nil {
		return
	}
	keys := make([]string, 0, len(userIDs)*2)
	for _, userID := range userIDs {
		keys = append(keys, relationshipsCacheKey(userID), recommendationsCacheKey(userID))
	}
	_ = s.cache.Delete(ctx, keys...)
}

func relationshipsCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("social:v1:relationships:%s", userID)
}

func recommendationsCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("social:v1:recommendations:%s", userID)
}

func mapFriendRequest(request domain.FriendRequest) FriendRequestDTO {
	return FriendRequestDTO{
		ID:          request.ID,
		SenderID:    request.SenderID,
		RecipientID: request.RecipientID,
		Status:      string(request.Status),
		CreatedAt:   request.CreatedAt,
		RespondedAt: request.RespondedAt,
	}
}

func mapFriendRequestError(err error) error {
	switch {
	case errors.Is(err, ports.ErrFriendRequestNotFound):
		return ErrFriendRequestNotFound
	case errors.Is(err, ports.ErrFriendRequestForbidden):
		return ErrFriendRequestForbidden
	case errors.Is(err, ports.ErrFriendRequestState):
		return ErrFriendRequestState
	default:
		return err
	}
}
