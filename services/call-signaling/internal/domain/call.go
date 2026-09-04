// Package domain содержит агрегат звонка и правила его жизненного цикла.
package domain

import (
	"errors"
	"fmt"
	"uuid"
)

// CallType определяет тип media-соединения.
type CallType string

const (
	// Audio — аудиозвонок.
	Audio CallType = "audio"
	// Video — видеозвонок.
	Video CallType = "video"
)

// CallStatus описывает состояние signaling-сессии.
type CallStatus string

const (
	// Ringing — приглашение ожидает ответа.
	Ringing CallStatus = "ringing"
	// Active — звонок принят и media negotiation разрешён.
	Active CallStatus = "active"
	// Rejected — приглашение отклонено.
	Rejected CallStatus = "rejected"
	// Ended — звонок завершён участником.
	Ended CallStatus = "ended"
)

var (
	// ErrNotParticipant означает, что пользователь не является участником звонка.
	ErrNotParticipant = errors.New("user is not a participant of the call")
	// ErrInvalidState означает недопустимый переход состояния звонка.
	ErrInvalidState = errors.New("invalid call state transition")
	// ErrOnlyCallee означает, что действие доступно только вызываемому.
	ErrOnlyCallee = errors.New("only the callee may accept or reject a call")
)

// Session содержит только управляющее состояние звонка, без аудио и видео.
type Session struct {
	CallID   uuid.UUID  `json:"call_id"`
	CallerID uuid.UUID  `json:"caller_id"`
	CalleeID uuid.UUID  `json:"callee_id"`
	CallType CallType   `json:"call_type"`
	Status   CallStatus `json:"status"`
}

// NewSession создаёт приглашение в состоянии ожидания и проверяет входные данные.
func NewSession(callerID, calleeID uuid.UUID, callType CallType) (Session, error) {
	if callerID == uuid.Nil() || calleeID == uuid.Nil() {
		return Session{}, errors.New("caller and callee are required")
	}
	if callerID == calleeID {
		return Session{}, errors.New("cannot call yourself")
	}
	if callType != Audio && callType != Video {
		return Session{}, fmt.Errorf("unsupported call type %q", callType)
	}
	return Session{CallID: uuid.New(), CallerID: callerID, CalleeID: calleeID, CallType: callType, Status: Ringing}, nil
}

// Includes проверяет, входит ли пользователь в звонок.
func (s Session) Includes(userID uuid.UUID) bool { return s.CallerID == userID || s.CalleeID == userID }

// Accept переводит ожидающий звонок в активное состояние.
func (s Session) Accept(userID uuid.UUID) (Session, error) {
	if userID != s.CalleeID {
		return Session{}, ErrOnlyCallee
	}
	if s.Status != Ringing {
		return Session{}, ErrInvalidState
	}
	s.Status = Active
	return s, nil
}

// Reject закрывает ожидающее приглашение без media-соединения.
func (s Session) Reject(userID uuid.UUID) (Session, error) {
	if userID != s.CalleeID {
		return Session{}, ErrOnlyCallee
	}
	if s.Status != Ringing {
		return Session{}, ErrInvalidState
	}
	s.Status = Rejected
	return s, nil
}

// End позволяет любому участнику завершить ожидающий или активный звонок.
func (s Session) End(userID uuid.UUID) (Session, error) {
	if !s.Includes(userID) {
		return Session{}, ErrNotParticipant
	}
	if s.Status != Ringing && s.Status != Active {
		return Session{}, ErrInvalidState
	}
	s.Status = Ended
	return s, nil
}
