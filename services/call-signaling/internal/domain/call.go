/*
Package domain contains the call aggregate and its lifecycle rules.

It intentionally knows nothing about WebSockets, Redis or PostgreSQL. The
transport validates identity and the store persists state, while this package
answers the business question: which transition is legal for this participant?
That separation keeps handlers small and makes the state machine directly
testable.
*/
package domain

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type CallType string

const (
	Audio CallType = "audio"
	Video CallType = "video"
)

type CallStatus string

const (
	Ringing  CallStatus = "ringing"
	Active   CallStatus = "active"
	Rejected CallStatus = "rejected"
	Ended    CallStatus = "ended"
)

var (
	ErrNotParticipant = errors.New("user is not a participant of the call")
	ErrInvalidState   = errors.New("invalid call state transition")
	ErrOnlyCallee     = errors.New("only the callee may accept or reject a call")
)

type Session struct {
	// Session is control-plane data only; it never contains audio or video.
	CallID   string     `json:"call_id"`
	CallerID int64      `json:"caller_id"`
	CalleeID int64      `json:"callee_id"`
	CallType CallType   `json:"call_type"`
	Status   CallStatus `json:"status"`
}

// NewSession creates a ringing invitation and rejects malformed client input.
func NewSession(callerID, calleeID int64, callType CallType) (Session, error) {
	if callerID == calleeID {
		return Session{}, errors.New("cannot call yourself")
	}
	if callType != Audio && callType != Video {
		return Session{}, fmt.Errorf("unsupported call type %q", callType)
	}
	return Session{CallID: uuid.NewString(), CallerID: callerID, CalleeID: calleeID, CallType: callType, Status: Ringing}, nil
}

// Includes is the basic authorization predicate for call participants.
func (s Session) Includes(userID int64) bool { return s.CallerID == userID || s.CalleeID == userID }

// Accept moves ringing to active. Only the invited callee may do this.
func (s Session) Accept(userID int64) (Session, error) {
	if userID != s.CalleeID {
		return Session{}, ErrOnlyCallee
	}
	if s.Status != Ringing {
		return Session{}, ErrInvalidState
	}
	s.Status = Active
	return s, nil
}

// Reject closes a ringing invitation without creating a media connection.
func (s Session) Reject(userID int64) (Session, error) {
	if userID != s.CalleeID {
		return Session{}, ErrOnlyCallee
	}
	if s.Status != Ringing {
		return Session{}, ErrInvalidState
	}
	s.Status = Rejected
	return s, nil
}

// End lets either participant terminate ringing or active calls. Terminal
// states cannot be revived by delayed browser messages.
func (s Session) End(userID int64) (Session, error) {
	if !s.Includes(userID) {
		return Session{}, ErrNotParticipant
	}
	if s.Status != Ringing && s.Status != Active {
		return Session{}, ErrInvalidState
	}
	s.Status = Ended
	return s, nil
}
