// Package domain содержит сущности и правила social-сервиса.
package domain

import (
	"time"
	"uuid"
)

// FriendRequestStatus описывает состояние заявки в друзья.
type FriendRequestStatus string

const (
	// FriendRequestPending означает, что заявка ожидает решения получателя.
	FriendRequestPending FriendRequestStatus = "pending"
	// FriendRequestAccepted означает, что заявка принята и friendship создана.
	FriendRequestAccepted FriendRequestStatus = "accepted"
	// FriendRequestDeclined означает, что получатель отказал в заявке.
	FriendRequestDeclined FriendRequestStatus = "declined"
	// FriendRequestCancelled означает, что отправитель отменил заявку.
	FriendRequestCancelled FriendRequestStatus = "cancelled" //nolint:misspell // API contract value.
)

// FriendRequest содержит направленную заявку между двумя пользователями.
type FriendRequest struct {
	ID          uuid.UUID
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Status      FriendRequestStatus
	CreatedAt   time.Time
	RespondedAt *time.Time
}

// IsPending сообщает, ожидает ли заявка решения.
func (r FriendRequest) IsPending() bool {
	return r.Status == FriendRequestPending
}

// CanBeAcceptedBy проверяет право пользователя принять заявку.
func (r FriendRequest) CanBeAcceptedBy(userID uuid.UUID) bool {
	return r.IsPending() && r.RecipientID == userID
}

// CanBeDeclinedBy проверяет право пользователя отклонить заявку.
func (r FriendRequest) CanBeDeclinedBy(userID uuid.UUID) bool {
	return r.IsPending() && r.RecipientID == userID
}

// CanBeCancelledBy проверяет право пользователя отменить заявку.
func (r FriendRequest) CanBeCancelledBy(userID uuid.UUID) bool {
	return r.IsPending() && r.SenderID == userID
}

// Pair возвращает UUID пользователей в стабильном порядке для friendship-ключа.
func Pair(first uuid.UUID, second uuid.UUID) (uuid.UUID, uuid.UUID) {
	if first.String() < second.String() {
		return first, second
	}
	return second, first
}
