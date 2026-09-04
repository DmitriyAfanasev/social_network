package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"uuid"

	"general-project/profiles/internal/ports"
)

var (
	// ErrValidation означает, что handle не соответствует правилам профиля.
	ErrValidation = errors.New("profile validation failed")
	// ErrConflict означает, что handle уже используется другим профилем.
	ErrConflict = errors.New("profile handle conflict")
	// ErrForbidden означает, что пользователь не владеет ресурсом профиля.
	ErrForbidden = errors.New("profile resource forbidden")
	// ErrMediaConflict означает повторное добавление медиаобъекта в альбом.
	ErrMediaConflict = errors.New("profile media conflict")
)

var handlePattern = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

const profileCacheTTL = 60 * time.Second

// ProfileService реализует сценарии чтения профиля и управления handle.
type ProfileService struct {
	profiles ports.ProfileRepository
	cache    ports.ProfileCache
}

// UpdateProfileInput содержит изменяемые публичные поля профиля.
type UpdateProfileInput struct {
	DisplayName string
	Bio         string
}

// NewProfileService создаёт application-сервис профилей.
func NewProfileService(profiles ports.ProfileRepository, cache ports.ProfileCache) *ProfileService {
	return &ProfileService{profiles: profiles, cache: cache}
}

// GetByHandle возвращает публичный профиль и использует короткоживущий cache.
func (s *ProfileService) GetByHandle(ctx context.Context, handle string) (ProfileDTO, error) {
	handle = normalizeHandle(handle)
	if !handlePattern.MatchString(handle) {
		return ProfileDTO{}, ErrValidation
	}

	cacheKey := profileCacheKey(handle)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached ProfileDTO
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}

	profile, err := s.profiles.FindByHandle(ctx, handle)
	if err != nil {
		return ProfileDTO{}, err
	}
	dto := toProfileDTO(profile)
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(dto); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, payload, profileCacheTTL)
		}
	}
	return dto, nil
}

// SearchByHandle возвращает профили с handle, начинающимся с указанного запроса.
func (s *ProfileService) SearchByHandle(ctx context.Context, query string, limit int) ([]ProfileDTO, error) {
	query = normalizeHandle(query)
	if query == "" || len(query) > 32 || limit < 1 || limit > 50 {
		return nil, ErrValidation
	}
	profiles, err := s.profiles.SearchByHandle(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	result := make([]ProfileDTO, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, toProfileDTO(profile))
	}
	return result, nil
}

// SetHandle назначает пользователю публичный URL-идентификатор.
func (s *ProfileService) SetHandle(ctx context.Context, userID uuid.UUID, handle string) (ProfileDTO, error) {
	handle = normalizeHandle(handle)
	if !handlePattern.MatchString(handle) {
		return ProfileDTO{}, ErrValidation
	}

	previous, previousErr := s.profiles.FindByUserID(ctx, userID)
	if previousErr != nil && !errors.Is(previousErr, ports.ErrNotFound) {
		return ProfileDTO{}, previousErr
	}
	profile, err := s.profiles.SetHandle(ctx, userID, handle)
	if errors.Is(err, ports.ErrAlreadyExists) {
		return ProfileDTO{}, ErrConflict
	}
	if err != nil {
		return ProfileDTO{}, err
	}
	if s.cache != nil {
		keys := []string{profileCacheKey(handle)}
		if previousErr == nil && previous.Handle != nil && *previous.Handle != handle {
			keys = append(keys, profileCacheKey(*previous.Handle))
		}
		_ = s.cache.Delete(ctx, keys...)
	}
	return toProfileDTO(profile), nil
}

// UpdatePublicProfile изменяет отображаемое имя и описание профиля.
func (s *ProfileService) UpdatePublicProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (ProfileDTO, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if len(displayName) > 80 || len(input.Bio) > 2000 {
		return ProfileDTO{}, ErrValidation
	}
	previous, previousErr := s.profiles.FindByUserID(ctx, userID)
	if previousErr != nil {
		return ProfileDTO{}, previousErr
	}
	profile, err := s.profiles.UpdatePublicProfile(ctx, userID, displayName, input.Bio)
	if err != nil {
		return ProfileDTO{}, err
	}
	if s.cache != nil && previous.Handle != nil {
		_ = s.cache.Delete(ctx, profileCacheKey(*previous.Handle))
	}
	return toProfileDTO(profile), nil
}

func normalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}

func profileCacheKey(handle string) string {
	return fmt.Sprintf("profiles:v1:handle:%s", handle)
}
