// Package application содержит сценарии доступа и социальных связей.
package application

import (
	"context"
	"uuid"

	"general-project/social/internal/ports"
)

// RelationshipAccessDTO описывает социальную близость пользователя к цели.
type RelationshipAccessDTO struct {
	IsFriend         bool
	IsFriendOfFriend bool
	IsBlocked        bool
}

// PublicFriendsDTO содержит список друзей и признак доступности списка.
type PublicFriendsDTO struct {
	Friends []uuid.UUID
	Visible bool
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

// GetPublicFriends возвращает список друзей пользователя, доступный viewer-у.
func (s *SocialService) GetPublicFriends(ctx context.Context, viewerID uuid.UUID, targetID uuid.UUID) (PublicFriendsDTO, error) {
	if targetID == uuid.Nil() {
		return PublicFriendsDTO{}, ErrValidation
	}
	if viewerID != uuid.Nil() && viewerID != targetID {
		policy := "everyone"
		if reader, ok := s.profilePolicy.(ports.FriendsVisibilityReader); ok {
			value, err := reader.GetFriendsVisibility(ctx, targetID)
			if err != nil {
				return PublicFriendsDTO{}, err
			}
			policy = value
		}
		switch policy {
		case "nobody":
			return PublicFriendsDTO{Friends: []uuid.UUID{}, Visible: false}, nil
		case "friends", "friends_of_friends":
			access, err := s.GetRelationship(ctx, viewerID, targetID)
			if err != nil {
				return PublicFriendsDTO{}, err
			}
			if !access.IsFriend && (policy == "friends" || !access.IsFriendOfFriend) {
				return PublicFriendsDTO{Friends: []uuid.UUID{}, Visible: false}, nil
			}
		case "everyone", "":
		default:
			return PublicFriendsDTO{Friends: []uuid.UUID{}, Visible: false}, nil
		}
	}
	friends, err := s.relationships.ListFriends(ctx, targetID)
	if err != nil {
		return PublicFriendsDTO{}, err
	}
	return PublicFriendsDTO{Friends: friends, Visible: true}, nil
}
