package postgres

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"general-project/messaging/internal/domain"
	"general-project/messaging/internal/ports"
)

func TestPostgresMessagingRepository(t *testing.T) {
	// Arrange
	dsn := os.Getenv("MESSAGING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MESSAGING_TEST_DATABASE_URL не задан")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	repository := NewRepository(pool)
	userID, otherUserID := uuid.New(), uuid.New()

	// Act
	conversation, err := repository.GetOrCreateDirect(ctx, userID, otherUserID)

	// Assert
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{userID, otherUserID}, conversation.ParticipantIDs)
	conversations, err := repository.ListConversations(ctx, userID, false)
	require.NoError(t, err)
	require.Len(t, conversations, 1)

	// Act
	messageID := uuid.New()
	message, err := repository.CreateMessage(ctx, domain.Message{
		ID:             messageID,
		ConversationID: conversation.ID,
		SenderID:       userID,
		Body:           "integration message",
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, messageID, message.ID)
	messages, hasMore, err := repository.ListMessages(ctx, otherUserID, conversation.ID, 0, 20)
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Len(t, messages, 1)
	require.Equal(t, "integration message", messages[0].Body)

	// Act
	updated, err := repository.UpdateMessage(ctx, userID, messageID, "edited message")

	// Assert
	require.NoError(t, err)
	require.Equal(t, "edited message", updated.Body)

	// Act
	err = repository.DeleteMessage(ctx, userID, messageID)

	// Assert
	require.NoError(t, err)
	_, err = repository.UpdateMessage(ctx, userID, messageID, "must fail")
	require.ErrorIs(t, err, ports.ErrNotFound)
}
