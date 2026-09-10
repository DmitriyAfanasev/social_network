package mediahttp

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"general-project/content/internal/ports"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestValidateOwned(t *testing.T) {
	userID, mediaID := uuid.New(), uuid.New()
	client := NewClient("https://media.test/")
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"uploaded_by":"` + userID.String() + `"}`)), Header: make(http.Header), Request: request}, nil
	})

	err := client.ValidateOwned(t.Context(), userID, []uuid.UUID{mediaID})
	require.NoError(t, err)
}

func TestValidateOwnedMapsRemoteFailures(t *testing.T) {
	userID, mediaID := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"not found", http.StatusNotFound, `{}`, ports.ErrMediaNotFound},
		{"wrong owner", http.StatusOK, `{"uploaded_by":"` + uuid.New().String() + `"}`, ports.ErrMediaForbidden},
		{"malformed owner", http.StatusOK, `{"uploaded_by":"bad"}`, ports.ErrMediaForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			client := NewClient("https://media.test")
			client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header), Request: request}, nil
			})

			// Act
			err := client.ValidateOwned(t.Context(), userID, []uuid.UUID{mediaID})

			// Assert
			require.ErrorIs(t, err, tc.want)
		})
	}
}
