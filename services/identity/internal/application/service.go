// Package application содержит сценарии identity-сервиса и их DTO.
package application

import (
	"context"

	"uuid"

	"general-project/identity/internal/ports"
)

// UserQueryService выполняет сценарии чтения пользователей.
type UserQueryService struct {
	users ports.UserRepository
}

// NewUserQueryService создаёт сервис чтения пользователей.
func NewUserQueryService(users ports.UserRepository) *UserQueryService {
	return &UserQueryService{users: users}
}

// GetUser возвращает DTO пользователя, не раскрывая domain entity наружу.
func (s *UserQueryService) GetUser(ctx context.Context, id uuid.UUID) (UserDTO, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return UserDTO{}, err
	}
	return toUserDTO(user), nil
}
