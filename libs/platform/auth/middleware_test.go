package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

type fakeVerifier struct {
	userID uuid.UUID
	err    error
}

func (f fakeVerifier) UserID(string) (uuid.UUID, error) {
	return f.userID, f.err
}

func TestBearerTokenAcceptsCaseInsensitiveSchemeAndWhitespace(t *testing.T) {
	token, ok := BearerToken("  bEaReR  access-token ")

	require.True(t, ok)
	require.Equal(t, "access-token", token)
}

func TestBearerTokenRejectsMalformedHeader(t *testing.T) {
	for _, header := range []string{"", "Basic token", "Bearer", "Bearer one two"} {
		_, ok := BearerToken(header)
		require.False(t, ok, header)
	}
}

func TestMiddlewarePutsUserIDInContext(t *testing.T) {
	wanted := uuid.New()
	handler := Middleware(fakeVerifier{userID: wanted})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Error("user ID is missing from context")
		}
		if userID != wanted {
			t.Errorf("user ID = %s, want %s", userID, wanted)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestMiddlewareReturnsConsistentUnauthorizedResponse(t *testing.T) {
	handler := Middleware(fakeVerifier{err: errors.New("invalid token")})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not be called")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/private", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"code":"unauthorized","message":"требуется действующий access-токен"}`+"\n", recorder.Body.String())
}
