package httptransport

import (
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestParseMediaIDs(t *testing.T) {
	// Arrange
	first, second := uuid.New(), uuid.New()
	values := []string{first.String(), second.String()}

	// Act
	got, err := parseMediaIDs(&values)

	// Assert
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{first, second}, got)
}

func TestParseMediaIDsRejectsInvalidUUID(t *testing.T) {
	// Arrange
	invalid := []string{"not-a-uuid"}

	// Act
	_, err := parseMediaIDs(&invalid)

	// Assert
	require.Error(t, err)
}

func TestParseMediaIDsAcceptsNil(t *testing.T) {
	// Act
	got, err := parseMediaIDs(nil)

	// Assert
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	// Arrange
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"body":"ok","unknown":true}`))
	var request postRequest

	// Act
	err := decodeJSON(r, &request)

	// Assert
	require.Error(t, err)
}
