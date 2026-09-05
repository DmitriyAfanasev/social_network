package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"uuid"

	"general-project/identity/internal/domain"
	"general-project/identity/internal/ports"
)

var (
	// ErrValidation означает, что входные данные не соответствуют контракту сценария.
	ErrValidation = errors.New("validation failed")
	// ErrConflict означает, что email уже занят.
	ErrConflict = errors.New("identity conflict")
	// ErrInvalidCredentials означает, что пара email и пароль не прошла проверку.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken означает, что refresh-токен отсутствует или недействителен.
	ErrInvalidToken = errors.New("invalid refresh token")
	// ErrInvalidVerificationToken означает, что одноразовый verification-токен недействителен.
	ErrInvalidVerificationToken = errors.New("invalid verification token")
)

const (
	registrationConfirmationPurpose = "registration_confirmation"
	passwordResetPurpose            = "password_reset"
)

// RegisterInput содержит входные данные сценария регистрации.
type RegisterInput struct {
	Email    string
	Password string
}

// LoginInput содержит входные данные сценария входа.
type LoginInput struct {
	Email    string
	Password string
}

// PasswordResetInput содержит email и новый пароль для сброса пароля.
type PasswordResetInput struct {
	Token       string
	NewPassword string
}

// VerificationConfig содержит зависимости и TTL для confirmation/reset сценариев.
type VerificationConfig struct {
	Tokens           ports.VerificationTokenStore
	Notifications    ports.NotificationSender
	ConfirmationTTL  time.Duration
	PasswordResetTTL time.Duration
	FrontendBaseURL  string
}

// Authenticator описывает доступные HTTP-сценарии аутентификации.
type Authenticator interface {
	Register(ctx context.Context, input RegisterInput) (RegistrationDTO, error)
	RequestRegistrationConfirmation(ctx context.Context, email string) (MessageDTO, error)
	ConfirmRegistration(ctx context.Context, token string) (AuthDTO, error)
	RequestPasswordReset(ctx context.Context, email string) (MessageDTO, error)
	ResetPassword(ctx context.Context, input PasswordResetInput) (AuthDTO, error)
	Login(ctx context.Context, input LoginInput) (AuthDTO, error)
	Refresh(ctx context.Context, refreshToken string) (AuthDTO, error)
	Logout(ctx context.Context, refreshToken string) error
	Permissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}

// AuthService реализует регистрацию и вход пользователя.
type AuthService struct {
	users        ports.UserRepository
	hasher       ports.PasswordHasher
	tokens       ports.TokenIssuer
	refresh      ports.RefreshTokenStore
	accessTTL    time.Duration
	refreshTTL   time.Duration
	verification VerificationConfig
}

// NewAuthService создаёт application-сервис аутентификации.
func NewAuthService(users ports.UserRepository, hasher ports.PasswordHasher, tokens ports.TokenIssuer, refresh ports.RefreshTokenStore, accessTTL time.Duration, refreshTTL time.Duration, verification VerificationConfig) *AuthService {
	return &AuthService{users: users, hasher: hasher, tokens: tokens, refresh: refresh, accessTTL: accessTTL, refreshTTL: refreshTTL, verification: verification}
}

// Register создаёт неподтверждённого пользователя и отправляет confirmation-ссылку.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (RegistrationDTO, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !strings.Contains(email, "@") || len(input.Password) < 8 {
		return RegistrationDTO{}, ErrValidation
	}
	if err := s.ensureVerificationConfigured(s.verificationConfirmationTTL()); err != nil {
		return RegistrationDTO{}, err
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return RegistrationDTO{}, err
	}
	user := domain.User{
		ID:     uuid.New(),
		Email:  email,
		Status: domain.UserStatusPending,
	}
	now := time.Now().UTC()
	payload, err := json.Marshal(map[string]string{"user_id": user.ID.String(), "email": user.Email})
	if err != nil {
		return RegistrationDTO{}, err
	}
	event := &ports.OutboxEvent{
		ID:          uuid.New(),
		EventType:   "identity.user.registered",
		AggregateID: &user.ID,
		Payload:     payload,
		AvailableAt: now,
		CreatedAt:   now,
	}
	created, err := s.users.Create(ctx, user, passwordHash, event)
	if errors.Is(err, ports.ErrAlreadyExists) {
		return RegistrationDTO{}, ErrConflict
	}
	if err != nil {
		return RegistrationDTO{}, err
	}
	if err := s.sendVerification(ctx, created, registrationConfirmationPurpose, s.verificationConfirmationTTL()); err != nil {
		return RegistrationDTO{}, err
	}
	return RegistrationDTO{User: toUserDTO(created), Message: "confirmation email sent"}, nil
}

// RequestRegistrationConfirmation повторно отправляет confirmation-ссылку неподтверждённому пользователю.
func (s *AuthService) RequestRegistrationConfirmation(ctx context.Context, email string) (MessageDTO, error) {
	email = normalizeEmail(email)
	if !strings.Contains(email, "@") {
		return MessageDTO{}, ErrValidation
	}
	if err := s.ensureVerificationConfigured(s.verificationConfirmationTTL()); err != nil {
		return MessageDTO{}, err
	}
	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, ports.ErrNotFound) {
		return MessageDTO{Message: "confirmation email sent"}, nil
	}
	if err != nil {
		return MessageDTO{}, err
	}
	if user.User.Status == domain.UserStatusActive {
		return MessageDTO{Message: "confirmation email sent"}, nil
	}
	if err := s.sendVerification(ctx, user.User, registrationConfirmationPurpose, s.verificationConfirmationTTL()); err != nil {
		return MessageDTO{}, err
	}
	return MessageDTO{Message: "confirmation email sent"}, nil
}

// ConfirmRegistration активирует пользователя одноразовым confirmation-токеном и выдаёт токены.
func (s *AuthService) ConfirmRegistration(ctx context.Context, token string) (AuthDTO, error) {
	if strings.TrimSpace(token) == "" || s.verification.Tokens == nil {
		return AuthDTO{}, ErrInvalidVerificationToken
	}
	userID, err := s.verification.Tokens.Consume(ctx, registrationConfirmationPurpose, hashToken(token))
	if errors.Is(err, ports.ErrNotFound) {
		return AuthDTO{}, ErrInvalidVerificationToken
	}
	if err != nil {
		return AuthDTO{}, err
	}
	user, err := s.users.Activate(ctx, userID)
	if errors.Is(err, ports.ErrNotFound) {
		return AuthDTO{}, ErrInvalidVerificationToken
	}
	if err != nil {
		return AuthDTO{}, err
	}
	return s.issueAuth(ctx, user)
}

// RequestPasswordReset создаёт reset-ссылку, не раскрывая существование email.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (MessageDTO, error) {
	email = normalizeEmail(email)
	if !strings.Contains(email, "@") {
		return MessageDTO{}, ErrValidation
	}
	if err := s.ensureVerificationConfigured(s.verificationPasswordResetTTL()); err != nil {
		return MessageDTO{}, err
	}
	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, ports.ErrNotFound) {
		return MessageDTO{Message: "password reset email sent"}, nil
	}
	if err != nil {
		return MessageDTO{}, err
	}
	if user.User.Status == domain.UserStatusBlocked {
		return MessageDTO{Message: "password reset email sent"}, nil
	}
	if err := s.sendVerification(ctx, user.User, passwordResetPurpose, s.verificationPasswordResetTTL()); err != nil {
		return MessageDTO{}, err
	}
	return MessageDTO{Message: "password reset email sent"}, nil
}

// ResetPassword меняет пароль по одноразовому reset-токену и отзывает старые сессии.
func (s *AuthService) ResetPassword(ctx context.Context, input PasswordResetInput) (AuthDTO, error) {
	if strings.TrimSpace(input.Token) == "" || len(input.NewPassword) < 8 || s.verification.Tokens == nil {
		return AuthDTO{}, ErrValidation
	}
	userID, err := s.verification.Tokens.Consume(ctx, passwordResetPurpose, hashToken(input.Token))
	if errors.Is(err, ports.ErrNotFound) {
		return AuthDTO{}, ErrInvalidVerificationToken
	}
	if err != nil {
		return AuthDTO{}, err
	}
	passwordHash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return AuthDTO{}, err
	}
	if err := s.users.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return AuthDTO{}, err
	}
	if err := s.refresh.RevokeAll(ctx, userID); err != nil {
		return AuthDTO{}, err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil || user.Status != domain.UserStatusActive {
		return AuthDTO{}, ErrInvalidVerificationToken
	}
	return s.issueAuth(ctx, user)
}

// Permissions возвращает эффективные permissions пользователя из identity RBAC.
func (s *AuthService) Permissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.users.ListPermissions(ctx, userID)
}

// Login проверяет credentials и возвращает новый токен доступа.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthDTO, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || input.Password == "" {
		return AuthDTO{}, ErrInvalidCredentials
	}

	authUser, err := s.users.FindByEmail(ctx, email)
	if err != nil || s.hasher.Compare(input.Password, authUser.PasswordHash) != nil || authUser.User.Status != domain.UserStatusActive {
		return AuthDTO{}, ErrInvalidCredentials
	}
	return s.issueAuth(ctx, authUser.User)
}

// Refresh ротирует refresh-токен и выдает новую пару токенов.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (AuthDTO, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return AuthDTO{}, ErrInvalidToken
	}
	replacement, replacementHash, replacementExpiresAt, err := newRefreshToken(s.refreshTTL)
	if err != nil {
		return AuthDTO{}, err
	}
	userID, err := s.refresh.Rotate(ctx, hashToken(refreshToken), replacementHash, replacementExpiresAt)
	if errors.Is(err, ports.ErrNotFound) {
		return AuthDTO{}, ErrInvalidToken
	}
	if err != nil {
		return AuthDTO{}, err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil || user.Status != domain.UserStatusActive {
		return AuthDTO{}, ErrInvalidToken
	}
	return s.authDTO(user, replacement, replacementExpiresAt)
}

// Logout отзывает refresh-токен пользователя.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	return s.refresh.Revoke(ctx, hashToken(refreshToken))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *AuthService) verificationConfirmationTTL() time.Duration {
	if s.verification.ConfirmationTTL > 0 {
		return s.verification.ConfirmationTTL
	}
	return 30 * time.Minute
}

func (s *AuthService) verificationPasswordResetTTL() time.Duration {
	if s.verification.PasswordResetTTL > 0 {
		return s.verification.PasswordResetTTL
	}
	return 10 * time.Minute
}

func (s *AuthService) ensureVerificationConfigured(ttl time.Duration) error {
	if s.verification.Tokens == nil || s.verification.Notifications == nil || ttl <= 0 {
		return errors.New("identity verification is not configured")
	}
	return nil
}

func (s *AuthService) sendVerification(ctx context.Context, user domain.User, purpose string, ttl time.Duration) error {
	token, tokenHash, expiresAt, err := newOpaqueToken(ttl)
	if err != nil {
		return err
	}
	if err := s.verification.Tokens.Save(ctx, user.ID, purpose, tokenHash, expiresAt); err != nil {
		return err
	}
	return s.verification.Notifications.Send(ctx, ports.Notification{
		Purpose: purpose,
		Email:   user.Email,
		Token:   token,
		BaseURL: s.verification.FrontendBaseURL,
	})
}

func (s *AuthService) issueAuth(ctx context.Context, user domain.User) (AuthDTO, error) {
	refreshToken, refreshHash, refreshExpiresAt, err := newRefreshToken(s.refreshTTL)
	if err != nil {
		return AuthDTO{}, err
	}
	if err := s.refresh.Save(ctx, user.ID, refreshHash, refreshExpiresAt); err != nil {
		return AuthDTO{}, err
	}
	return s.authDTO(user, refreshToken, refreshExpiresAt)
}

func (s *AuthService) authDTO(user domain.User, refreshToken string, refreshExpiresAt time.Time) (AuthDTO, error) {
	accessToken, err := s.tokens.IssueAccess(user.ID)
	if err != nil {
		return AuthDTO{}, err
	}
	return AuthDTO{
		User:             toUserDTO(user),
		AccessToken:      accessToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(s.accessTTL.Seconds()),
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int64(time.Until(refreshExpiresAt).Seconds()),
	}, nil
}

func newRefreshToken(ttl time.Duration) (string, string, time.Time, error) {
	if ttl <= 0 {
		return "", "", time.Time{}, errors.New("refresh token ttl is not configured")
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", time.Time{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(raw[:])
	expiresAt := time.Now().UTC().Add(ttl)
	return value, hashToken(value), expiresAt, nil
}

func newOpaqueToken(ttl time.Duration) (string, string, time.Time, error) {
	if ttl <= 0 {
		return "", "", time.Time{}, errors.New("verification token ttl is not configured")
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", time.Time{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(raw[:])
	return value, hashToken(value), time.Now().UTC().Add(ttl), nil
}

func hashToken(value string) string {
	hash := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
