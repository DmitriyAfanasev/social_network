package domain

import (
	"errors"
	"testing"
	"uuid"
)

func TestSessionLifecycle(t *testing.T) {
	caller, callee := uuid.New(), uuid.New()
	session, err := NewSession(caller, callee, Video)
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != Ringing || !session.Includes(caller) || session.Includes(uuid.New()) {
		t.Fatalf("unexpected initial session: %+v", session)
	}

	active, err := session.Accept(callee)
	if err != nil {
		t.Fatal(err)
	}
	if active.Status != Active {
		t.Fatalf("expected active, got %s", active.Status)
	}
	if _, err := active.Accept(callee); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state, got %v", err)
	}
	ended, err := active.End(caller)
	if err != nil {
		t.Fatal(err)
	}
	if ended.Status != Ended {
		t.Fatalf("expected ended, got %s", ended.Status)
	}
}

func TestOnlyCalleeCanAcceptOrReject(t *testing.T) {
	caller, callee := uuid.New(), uuid.New()
	session, err := NewSession(caller, callee, Audio)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Accept(caller); !errors.Is(err, ErrOnlyCallee) {
		t.Fatalf("expected only callee error, got %v", err)
	}
	if _, err := session.Reject(caller); !errors.Is(err, ErrOnlyCallee) {
		t.Fatalf("expected only callee error, got %v", err)
	}
}

func TestNewSessionValidation(t *testing.T) {
	userID := uuid.New()
	if _, err := NewSession(userID, userID, Audio); err == nil {
		t.Fatal("expected self-call error")
	}
	if _, err := NewSession(uuid.New(), uuid.New(), CallType("screen-share")); err == nil {
		t.Fatal("expected type error")
	}
}
