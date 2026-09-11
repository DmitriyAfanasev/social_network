package application

import (
	"context"
	"testing"

	"uuid"

	"github.com/stretchr/testify/require"

	"general-project/identity/internal/domain"
	"general-project/identity/internal/ports"
)

func TestAuthServiceRequestRegistrationConfirmationUsesSafeResponses(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tests := []struct {
		name       string
		repository *fakeUserRepository
		expectSave bool
	}{
		{name: "unknown email", repository: &fakeUserRepository{findByEmailErr: ports.ErrNotFound}},
		{name: "active user", repository: &fakeUserRepository{findByEmail: domain.AuthUser{User: domain.User{ID: userID, Status: domain.UserStatusActive}}}},
		{name: "pending user", repository: &fakeUserRepository{findByEmail: domain.AuthUser{User: domain.User{ID: userID, Email: "alice@example.com", Status: domain.UserStatusPending}}}, expectSave: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokens := &fakeVerificationTokenStore{}
			notifications := &fakeNotificationSender{}
			service := NewAuthService(tt.repository, fakePasswordHasher{}, fakeTokenIssuer{}, &fakeRefreshTokenStore{}, 0, 0, VerificationConfig{Tokens: tokens, Notifications: notifications})

			result, err := service.RequestRegistrationConfirmation(context.Background(), " Alice@Example.COM ")

			require.NoError(t, err)
			require.Equal(t, "confirmation email sent", result.Message)
			if tt.expectSave {
				require.Equal(t, userID, tokens.savedUserID)
				require.Equal(t, registrationConfirmationPurpose, tokens.savedPurpose)
				require.Equal(t, "alice@example.com", notifications.notification.Email)
				require.NotEmpty(t, notifications.notification.Token)
			} else {
				require.Equal(t, uuid.Nil(), tokens.savedUserID)
				require.Empty(t, notifications.notification.Token)
			}
		})
	}
}

func TestAuthServiceRequestRegistrationConfirmationRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	service := NewAuthService(&fakeUserRepository{}, fakePasswordHasher{}, fakeTokenIssuer{}, &fakeRefreshTokenStore{}, 0, 0, verificationConfig())

	_, err := service.RequestRegistrationConfirmation(context.Background(), "invalid-email")

	require.ErrorIs(t, err, ErrValidation)
}
