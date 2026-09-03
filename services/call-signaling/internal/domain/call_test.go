package domain

import (
	"errors"
	"testing"
)

func TestSessionLifecycle(t *testing.T) {
	session, err := NewSession(10, 20, Video)
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != Ringing || !session.Includes(10) || session.Includes(99) {
		t.Fatalf("unexpected initial session: %+v", session)
	}

	active, err := session.Accept(20)
	if err != nil {
		t.Fatal(err)
	}
	if active.Status != Active {
		t.Fatalf("expected active, got %s", active.Status)
	}
	if _, err := active.Accept(20); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state, got %v", err)
	}
	ended, err := active.End(10)
	if err != nil {
		t.Fatal(err)
	}
	if ended.Status != Ended {
		t.Fatalf("expected ended, got %s", ended.Status)
	}
}

func TestOnlyCalleeCanAcceptOrReject(t *testing.T) {
	session, err := NewSession(1, 2, Audio)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Accept(1); !errors.Is(err, ErrOnlyCallee) {
		t.Fatalf("expected only callee error, got %v", err)
	}
	if _, err := session.Reject(1); !errors.Is(err, ErrOnlyCallee) {
		t.Fatalf("expected only callee error, got %v", err)
	}
}

func TestNewSessionValidation(t *testing.T) {
	if _, err := NewSession(1, 1, Audio); err == nil {
		t.Fatal("expected self-call error")
	}
	if _, err := NewSession(1, 2, CallType("screen-share")); err == nil {
		t.Fatal("expected type error")
	}
}
