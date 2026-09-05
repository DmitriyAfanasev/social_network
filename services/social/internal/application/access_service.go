package application

import (
	"context"
	"uuid"
)

// RelationshipAccessDTO описывает социальную близость пользователя к цели.
type RelationshipAccessDTO struct {
	IsFriend         bool
	IsFriendOfFriend bool
	IsBlocked        bool
}

// GetRelationship возвращает признаки доступа к профилю другого пользователя.
func (s *SocialService) GetRelationship(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID) (RelationshipAccessDTO, error) {
	if actorID == uuid.Nil() || targetID == uuid.Nil() || actorID == targetID {
		return RelationshipAccessDTO{}, nil
	}
	blocked, err := s.blocks.IsBlocked(ctx, actorID, targetID)
	if err != nil {
		return RelationshipAccessDTO{}, err
	}
	friends, err := s.relationships.IsFriend(ctx, actorID, targetID)
	if err != nil {
		return RelationshipAccessDTO{}, err
	}
	if friends {
		return RelationshipAccessDTO{IsFriend: true, IsBlocked: blocked}, nil
	}
	actorFriends, err := s.relationships.ListFriends(ctx, actorID)
	if err != nil {
		return RelationshipAccessDTO{}, err
	}
	for _, friendID := range actorFriends {
		connected, checkErr := s.relationships.IsFriend(ctx, friendID, targetID)
		if checkErr != nil {
			return RelationshipAccessDTO{}, checkErr
		}
		if connected {
			return RelationshipAccessDTO{IsFriendOfFriend: true, IsBlocked: blocked}, nil
		}
	}
	return RelationshipAccessDTO{IsBlocked: blocked}, nil
}
