// Package application содержит сценарии жизненного цикла call-сессии.
package application

import (
	"context"
	"errors"
	"uuid"

	"general-project/call-signaling/internal/domain"
	"general-project/call-signaling/internal/ports"
)

var (
	// ErrCallNotActive означает, что signaling разрешён только для активного звонка.
	ErrCallNotActive = errors.New("call is not active")
	// ErrCallNotFound означает, что TTL call-сессии истёк или она не существует.
	ErrCallNotFound = errors.New("call not found")
)

// SessionDTO — безопасное представление состояния звонка на границе application.
type SessionDTO struct {
	CallID   uuid.UUID
	CallerID uuid.UUID
	CalleeID uuid.UUID
	CallType domain.CallType
	Status   domain.CallStatus
}

// Service реализует сценарии call signaling через порты хранилища и авторизации.
type Service struct {
	sessions   ports.SessionStore
	authorizer ports.CallAuthorizer
}

// NewService создаёт application-сервис звонков.
func NewService(sessions ports.SessionStore, authorizer ports.CallAuthorizer) *Service {
	return &Service{sessions: sessions, authorizer: authorizer}
}

// Subscribe проверяет доступ пользователя к прямому диалогу.
func (s *Service) Subscribe(ctx context.Context, conversationID, userID uuid.UUID) error {
	return s.authorizer.CanSubscribe(ctx, conversationID, userID)
}

// Start создаёт звонок в состоянии ожидания после проверки участников.
func (s *Service) Start(ctx context.Context, conversationID, callerID, calleeID uuid.UUID, callType domain.CallType) (SessionDTO, error) {
	if err := s.authorizer.CanStart(ctx, conversationID, callerID, calleeID); err != nil {
		return SessionDTO{}, err
	}
	session, err := domain.NewSession(callerID, calleeID, callType)
	if err != nil {
		return SessionDTO{}, err
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return SessionDTO{}, err
	}
	return toSessionDTO(session), nil
}

// Transition принимает, отклоняет или завершает звонок атомарно.
func (s *Service) Transition(ctx context.Context, conversationID, userID, callID uuid.UUID, action string) (SessionDTO, error) {
	if action != "accept" && action != "reject" && action != "end" {
		return SessionDTO{}, domain.ErrInvalidState
	}
	session, err := s.getAuthorized(ctx, conversationID, userID, callID)
	if err != nil {
		return SessionDTO{}, err
	}
	session, err = s.sessions.Transition(ctx, session.CallID, userID, action)
	if err != nil {
		return SessionDTO{}, err
	}
	return toSessionDTO(session), nil
}

// KeepAlive продлевает TTL активной call-сессии участника.
func (s *Service) KeepAlive(ctx context.Context, conversationID, userID, callID uuid.UUID) error {
	session, err := s.getAuthorized(ctx, conversationID, userID, callID)
	if err != nil {
		return err
	}
	if session.Status != domain.Active {
		return ErrCallNotActive
	}
	err = s.sessions.Refresh(ctx, callID)
	if errors.Is(err, ports.ErrNotFound) {
		return ErrCallNotFound
	}
	return err
}

// AuthorizeSignal проверяет право пересылать offer, answer или ICE.
func (s *Service) AuthorizeSignal(ctx context.Context, conversationID, userID, callID uuid.UUID) (SessionDTO, error) {
	session, err := s.getAuthorized(ctx, conversationID, userID, callID)
	if err != nil {
		return SessionDTO{}, err
	}
	if session.Status != domain.Active {
		return SessionDTO{}, ErrCallNotActive
	}
	return toSessionDTO(session), nil
}

func (s *Service) getAuthorized(ctx context.Context, conversationID, userID, callID uuid.UUID) (domain.Session, error) {
	session, err := s.sessions.Get(ctx, callID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return domain.Session{}, ErrCallNotFound
		}
		return domain.Session{}, err
	}
	if err := s.authorizer.CanUseCall(ctx, conversationID, userID, session.CallerID, session.CalleeID); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func toSessionDTO(session domain.Session) SessionDTO {
	return SessionDTO{CallID: session.CallID, CallerID: session.CallerID, CalleeID: session.CalleeID, CallType: session.CallType, Status: session.Status}
}
