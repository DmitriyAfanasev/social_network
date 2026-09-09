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

	"general-project/profiles/internal/domain"
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
	profiles      ports.ProfileRepository
	cache         ports.ProfileCache
	relationships ports.RelationshipReader
}

// UpdateProfileInput содержит изменяемые поля профиля.
type UpdateProfileInput struct {
	Bio     string
	Details domain.ProfileDetails
}

// UpdatePrivacyInput содержит политики доступа профиля.
type UpdatePrivacyInput struct {
	Privacy domain.ProfilePrivacy
}

// NewProfileService создаёт application-сервис профилей.
func NewProfileService(profiles ports.ProfileRepository, cache ports.ProfileCache, relationships ...ports.RelationshipReader) *ProfileService {
	var relationshipReader ports.RelationshipReader
	if len(relationships) > 0 {
		relationshipReader = relationships[0]
	}
	return &ProfileService{profiles: profiles, cache: cache, relationships: relationshipReader}
}

// GetByHandle возвращает публичный профиль и использует короткоживущий cache.
func (s *ProfileService) GetByHandle(ctx context.Context, handle string, viewerIDs ...uuid.UUID) (ProfileDTO, error) {
	handle = normalizeHandle(handle)
	if !handlePattern.MatchString(handle) {
		return ProfileDTO{}, ErrValidation
	}

	cacheKey := profileCacheKey(handle)
	if s.cache != nil {
		if payload, err := s.cache.Get(ctx, cacheKey); err == nil && payload != nil {
			var cached ProfileDTO
			if json.Unmarshal(payload, &cached) == nil {
				return s.visible(ctx, cached, firstViewer(viewerIDs)), nil
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
	return s.visible(ctx, dto, firstViewer(viewerIDs)), nil
}

// GetMine возвращает профиль пользователя из access-токена.
func (s *ProfileService) GetMine(ctx context.Context, userID uuid.UUID) (ProfileDTO, error) {
	profile, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return ProfileDTO{}, err
	}
	return toProfileDTO(profile), nil
}

// GetByUserID возвращает публичный профиль по UUID пользователя.
func (s *ProfileService) GetByUserID(ctx context.Context, userID uuid.UUID, viewerIDs ...uuid.UUID) (ProfileDTO, error) {
	profile, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return ProfileDTO{}, err
	}
	return s.visible(ctx, toProfileDTO(profile), firstViewer(viewerIDs)), nil
}

func firstViewer(viewerIDs []uuid.UUID) uuid.UUID {
	if len(viewerIDs) > 0 {
		return viewerIDs[0]
	}
	return uuid.Nil()
}

func (s *ProfileService) visible(ctx context.Context, profile ProfileDTO, viewerID uuid.UUID) ProfileDTO {
	if viewerID == uuid.Nil() || viewerID == profile.UserID {
		return profile
	}
	access := ports.RelationshipAccess{}
	if s.relationships != nil {
		if result, err := s.relationships.GetRelationship(ctx, viewerID, profile.UserID); err == nil {
			access = result
		}
	}
	if !allowed(profile.Privacy.ProfileVisibility, access) {
		profile.Bio = ""
		profile.Details = ProfileDetailsDTO{}
		return profile
	}
	if !allowed(profile.Privacy.PhoneVisibility, access) {
		profile.Details.PhoneNumber = ""
	}
	if !allowed(profile.Privacy.BirthDateVisibility, access) {
		profile.Details.BirthDate = ""
	}
	if !allowed(profile.Privacy.GenderVisibility, access) {
		profile.Details.Gender = ""
	}
	if !allowed(profile.Privacy.LocationVisibility, access) {
		profile.Details.Country, profile.Details.City, profile.Details.Street = "", "", ""
	}
	if !allowed(profile.Privacy.StatusVisibility, access) {
		profile.Details.Status = ""
	}
	return profile
}

func allowed(policy string, access ports.RelationshipAccess) bool {
	switch policy {
	case "everyone", "":
		return true
	case "friends":
		return access.IsFriend
	case "friends_of_friends":
		return access.IsFriend || access.IsFriendOfFriend
	case "nobody":
		return false
	default:
		return false
	}
}

// EnsureProfile создаёт базовый профиль пользователя и безопасен для повторного вызова.
func (s *ProfileService) EnsureProfile(ctx context.Context, userID uuid.UUID) error {
	_, err := s.profiles.EnsureByUserID(ctx, userID)
	return err
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

// UpdatePublicProfile изменяет описание профиля.
func (s *ProfileService) UpdatePublicProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (ProfileDTO, error) {
	bio := strings.TrimSpace(input.Bio)
	if len([]rune(bio)) > 2000 {
		return ProfileDTO{}, ErrValidation
	}
	previous, previousErr := s.profiles.FindByUserID(ctx, userID)
	if previousErr != nil {
		return ProfileDTO{}, previousErr
	}
	profile, err := s.profiles.UpdatePublicProfile(ctx, userID, bio)
	if err != nil {
		return ProfileDTO{}, err
	}
	if s.cache != nil && previous.Handle != nil {
		_ = s.cache.Delete(ctx, profileCacheKey(*previous.Handle))
	}
	return toProfileDTO(profile), nil
}

// UpdateProfile изменяет публичное описание и дополнительные сведения профиля.
func (s *ProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (ProfileDTO, error) {
	bio := strings.TrimSpace(input.Bio)
	if len([]rune(bio)) > 2000 || !validDetails(input.Details) {
		return ProfileDTO{}, ErrValidation
	}
	previous, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return ProfileDTO{}, err
	}
	if _, err := s.profiles.UpdatePublicProfile(ctx, userID, bio); err != nil {
		return ProfileDTO{}, err
	}
	profile, err := s.profiles.UpdateProfileDetails(ctx, userID, input.Details)
	if err != nil {
		return ProfileDTO{}, err
	}
	if s.cache != nil && previous.Handle != nil {
		_ = s.cache.Delete(ctx, profileCacheKey(*previous.Handle))
	}
	return toProfileDTO(profile), nil
}

// GetPrivacy возвращает настройки приватности текущего пользователя.
func (s *ProfileService) GetPrivacy(ctx context.Context, userID uuid.UUID) (ProfilePrivacyDTO, error) {
	profile, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return ProfilePrivacyDTO{}, err
	}
	return toProfileDTO(profile).Privacy, nil
}

// UpdatePrivacy сохраняет политики доступа текущего пользователя.
func (s *ProfileService) UpdatePrivacy(ctx context.Context, userID uuid.UUID, input UpdatePrivacyInput) (ProfilePrivacyDTO, error) {
	if !validPrivacy(input.Privacy) {
		return ProfilePrivacyDTO{}, ErrValidation
	}
	previous, err := s.profiles.FindByUserID(ctx, userID)
	if err != nil {
		return ProfilePrivacyDTO{}, err
	}
	profile, err := s.profiles.UpdateProfilePrivacy(ctx, userID, input.Privacy)
	if err != nil {
		return ProfilePrivacyDTO{}, err
	}
	if s.cache != nil && previous.Handle != nil {
		_ = s.cache.Delete(ctx, profileCacheKey(*previous.Handle))
	}
	return toProfileDTO(profile).Privacy, nil
}

func validDetails(details domain.ProfileDetails) bool {
	return len([]rune(details.FirstName)) <= 80 && len([]rune(details.LastName)) <= 80 &&
		len([]rune(details.MiddleName)) <= 80 && len([]rune(details.BirthDate)) <= 10 &&
		len([]rune(details.Gender)) <= 32 && len([]rune(details.PhoneNumber)) <= 32 &&
		len([]rune(details.Country)) <= 80 && len([]rune(details.City)) <= 80 &&
		len([]rune(details.Street)) <= 160 && len([]rune(details.Status)) <= 160
}

func validPrivacy(privacy domain.ProfilePrivacy) bool {
	return validVisibility(privacy.ProfileVisibility, false) &&
		validVisibility(privacy.FriendRequestPolicy, true) &&
		validVisibility(privacy.MessagePolicy, true) &&
		validVisibility(privacy.PhoneVisibility, false) &&
		validVisibility(privacy.BirthDateVisibility, false) &&
		validVisibility(privacy.GenderVisibility, false) &&
		validVisibility(privacy.LocationVisibility, false) &&
		validVisibility(privacy.StatusVisibility, false) &&
		validVisibility(privacy.FriendsVisibility, false) &&
		validVisibility(privacy.PostsVisibility, false) &&
		validVisibility(privacy.MusicVisibility, false)
}

func validVisibility(value string, allowNobody bool) bool {
	switch value {
	case "everyone", "friends", "friends_of_friends":
		return true
	case "nobody":
		return allowNobody || value == "nobody"
	default:
		return false
	}
}

func normalizeHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(handle))
}

func profileCacheKey(handle string) string {
	return fmt.Sprintf("profiles:v1:handle:%s", handle)
}
