package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/identity/internal/domain"
	"general-project/identity/internal/ports"
)

type fakeUserRepository struct {
	createdUser    domain.User
	createdHash    string
	findByEmail    domain.AuthUser
	findByEmailErr error
	findByID       domain.User
	findByIDErr    error
	createErr      error
	activateUser   domain.User
	activateErr    error
	updatedUserID  uuid.UUID
	updatedHash    string
	permissions    []string
}

func (f *fakeUserRepository) FindByID(_ context.Context, _ uuid.UUID) (domain.User, error) {
	return f.findByID, f.findByIDErr
}

func (f *fakeUserRepository) FindByEmail(_ context.Context, _ string) (domain.AuthUser, error) {
	return f.findByEmail, f.findByEmailErr
}

func (f *fakeUserRepository) Create(_ context.Context, user domain.User, passwordHash string) (domain.User, error) {
	f.createdUser = user
	f.createdHash = passwordHash
	if f.createErr != nil {
		return domain.User{}, f.createErr
	}
	user.CreatedAt = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	user.UpdatedAt = user.CreatedAt
	return user, nil
}

func (f *fakeUserRepository) Activate(_ context.Context, _ uuid.UUID) (domain.User, error) {
	return f.activateUser, f.activateErr
}

func (f *fakeUserRepository) UpdatePassword(_ context.Context, userID uuid.UUID, passwordHash string) error {
	f.updatedUserID = userID
	f.updatedHash = passwordHash
	return nil
}

func (f *fakeUserRepository) ListPermissions(_ context.Context, _ uuid.UUID) ([]string, error) {
	return f.permissions, nil
}

type fakePasswordHasher struct {
	compareErr error
}

func (f fakePasswordHasher) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (f fakePasswordHasher) Compare(_ string, _ string) error {
	return f.compareErr
}

type fakeTokenIssuer struct{}

func (fakeTokenIssuer) IssueAccess(_ uuid.UUID) (string, error) {
	return "access-token", nil
}

type fakeRefreshTokenStore struct {
	savedUserID uuid.UUID
	savedHash   string
	rotateID    uuid.UUID
	rotateErr   error
	revokeErr   error
	revokedAll  uuid.UUID
}

func (f *fakeRefreshTokenStore) Save(_ context.Context, userID uuid.UUID, tokenHash string, _ time.Time) error {
	f.savedUserID = userID
	f.savedHash = tokenHash
	return nil
}

func (f *fakeRefreshTokenStore) Rotate(_ context.Context, _ string, _ string, _ time.Time) (uuid.UUID, error) {
	return f.rotateID, f.rotateErr
}

func (f *fakeRefreshTokenStore) Revoke(_ context.Context, _ string) error {
	return f.revokeErr
}

func (f *fakeRefreshTokenStore) RevokeAll(_ context.Context, userID uuid.UUID) error {
	f.revokedAll = userID
	return nil
}

type fakeVerificationTokenStore struct {
	savedUserID     uuid.UUID
	savedPurpose    string
	savedHash       string
	consumeID       uuid.UUID
	consumeErr      error
	consumedPurpose string
}

func (f *fakeVerificationTokenStore) Save(_ context.Context, userID uuid.UUID, purpose string, tokenHash string, _ time.Time) error {
	f.savedUserID = userID
	f.savedPurpose = purpose
	f.savedHash = tokenHash
	return nil
}

func (f *fakeVerificationTokenStore) Consume(_ context.Context, purpose string, _ string) (uuid.UUID, error) {
	f.consumedPurpose = purpose
	return f.consumeID, f.consumeErr
}

type fakeNotificationSender struct {
	notification ports.Notification
}

func (f *fakeNotificationSender) Send(_ context.Context, notification ports.Notification) error {
	f.notification = notification
	return nil
}

func verificationConfig() VerificationConfig {
	return VerificationConfig{
		Tokens:           &fakeVerificationTokenStore{},
		Notifications:    &fakeNotificationSender{},
		ConfirmationTTL:  30 * time.Minute,
		PasswordResetTTL: 10 * time.Minute,
	}
}

func TestAuthServiceRegisterNormalizesEmailAndMapsDTO(t *testing.T) {
	t.Parallel()

	repository := &fakeUserRepository{}
	refresh := &fakeRefreshTokenStore{}
	service := NewAuthService(repository, fakePasswordHasher{}, fakeTokenIssuer{}, refresh, 30*time.Minute, 7*24*time.Hour, verificationConfig())

	result, err := service.Register(context.Background(), RegisterInput{
		Email:    " Alice@Example.COM ",
		Password: "secret123",
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil(), repository.createdUser.ID)
	require.Equal(t, "alice@example.com", repository.createdUser.Email)
	require.Equal(t, "hash:secret123", repository.createdHash)
	require.Equal(t, domain.UserStatusPending, repository.createdUser.Status)
	require.Equal(t, "alice@example.com", result.User.Email)
	require.Equal(t, "confirmation email sent", result.Message)
	require.Equal(t, uuid.Nil(), refresh.savedUserID)
}

func TestAuthServiceRegisterReturnsConflict(t *testing.T) {
	t.Parallel()

	service := NewAuthService(
		&fakeUserRepository{createErr: ports.ErrAlreadyExists},
		fakePasswordHasher{},
		fakeTokenIssuer{},
		&fakeRefreshTokenStore{},
		30*time.Minute,
		7*24*time.Hour,
		verificationConfig(),
	)

	_, err := service.Register(context.Background(), RegisterInput{
		Email:    "alice@example.com",
		Password: "secret123",
	})

	require.ErrorIs(t, err, ErrConflict)
}

func TestAuthServiceLoginRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	service := NewAuthService(
		&fakeUserRepository{findByEmailErr: errors.New("database unavailable")},
		fakePasswordHasher{},
		fakeTokenIssuer{},
		&fakeRefreshTokenStore{},
		30*time.Minute,
		7*24*time.Hour,
		verificationConfig(),
	)

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "wrong",
	})

	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthServiceLoginRejectsBlockedUser(t *testing.T) {
	t.Parallel()

	service := NewAuthService(
		&fakeUserRepository{findByEmail: domain.AuthUser{
			User:         domain.User{ID: uuid.New(), Status: domain.UserStatusBlocked},
			PasswordHash: "hash",
		}},
		fakePasswordHasher{},
		fakeTokenIssuer{},
		&fakeRefreshTokenStore{},
		30*time.Minute,
		7*24*time.Hour,
		verificationConfig(),
	)

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "secret123",
	})

	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthServiceRefreshRotatesToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	service := NewAuthService(
		&fakeUserRepository{findByID: domain.User{ID: userID, Email: "alice@example.com", Status: domain.UserStatusActive}},
		fakePasswordHasher{},
		fakeTokenIssuer{},
		&fakeRefreshTokenStore{rotateID: userID},
		30*time.Minute,
		7*24*time.Hour,
		verificationConfig(),
	)

	result, err := service.Refresh(context.Background(), "old-refresh-token")

	require.NoError(t, err)
	require.Equal(t, userID, result.User.ID)
	require.Equal(t, "access-token", result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
}

func TestAuthServiceLogoutAcceptsEmptyToken(t *testing.T) {
	t.Parallel()

	service := NewAuthService(
		&fakeUserRepository{},
		fakePasswordHasher{},
		fakeTokenIssuer{},
		&fakeRefreshTokenStore{},
		30*time.Minute,
		7*24*time.Hour,
		verificationConfig(),
	)

	require.NoError(t, service.Logout(context.Background(), ""))
}

func TestAuthServiceConfirmationActivatesUserAndIssuesSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenStore := &fakeVerificationTokenStore{consumeID: userID}
	userRepository := &fakeUserRepository{activateUser: domain.User{
		ID: userID, Email: "alice@example.com", Status: domain.UserStatusActive,
	}}
	service := NewAuthService(
		userRepository, fakePasswordHasher{}, fakeTokenIssuer{}, &fakeRefreshTokenStore{},
		30*time.Minute, 7*24*time.Hour,
		VerificationConfig{Tokens: tokenStore, Notifications: &fakeNotificationSender{}},
	)

	result, err := service.ConfirmRegistration(context.Background(), "confirmation-token")

	require.NoError(t, err)
	require.Equal(t, userID, result.User.ID)
	require.Equal(t, "access-token", result.AccessToken)
	require.Equal(t, registrationConfirmationPurpose, tokenStore.consumedPurpose)
}

func TestAuthServiceResetPasswordRevokesExistingSessions(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tokenStore := &fakeVerificationTokenStore{consumeID: userID}
	userRepository := &fakeUserRepository{findByID: domain.User{
		ID: userID, Email: "alice@example.com", Status: domain.UserStatusActive,
	}}
	refresh := &fakeRefreshTokenStore{}
	service := NewAuthService(
		userRepository, fakePasswordHasher{}, fakeTokenIssuer{}, refresh,
		30*time.Minute, 7*24*time.Hour,
		VerificationConfig{Tokens: tokenStore, Notifications: &fakeNotificationSender{}},
	)

	result, err := service.ResetPassword(context.Background(), PasswordResetInput{Token: "reset-token", NewPassword: "new-secret"})

	require.NoError(t, err)
	require.Equal(t, userID, result.User.ID)
	require.Equal(t, userID, userRepository.updatedUserID)
	require.Equal(t, "hash:new-secret", userRepository.updatedHash)
	require.Equal(t, userID, refresh.revokedAll)
	require.Equal(t, passwordResetPurpose, tokenStore.consumedPurpose)
}

func TestAuthServicePasswordResetDoesNotRevealUnknownEmail(t *testing.T) {
	t.Parallel()

	sender := &fakeNotificationSender{}
	service := NewAuthService(
		&fakeUserRepository{findByEmailErr: ports.ErrNotFound}, fakePasswordHasher{}, fakeTokenIssuer{}, &fakeRefreshTokenStore{},
		30*time.Minute, 7*24*time.Hour,
		VerificationConfig{Tokens: &fakeVerificationTokenStore{}, Notifications: sender},
	)

	result, err := service.RequestPasswordReset(context.Background(), "unknown@example.com")

	require.NoError(t, err)
	require.Equal(t, "password reset email sent", result.Message)
	require.Empty(t, sender.notification.Token)
}

func TestAuthServiceReturnsEffectivePermissions(t *testing.T) {
	t.Parallel()

	wanted := []string{"comments.delete_any", "users.block"}
	service := NewAuthService(
		&fakeUserRepository{permissions: wanted}, fakePasswordHasher{}, fakeTokenIssuer{}, &fakeRefreshTokenStore{},
		30*time.Minute, 7*24*time.Hour, VerificationConfig{},
	)

	permissions, err := service.Permissions(context.Background(), uuid.New())

	require.NoError(t, err)
	require.Equal(t, wanted, permissions)
}
